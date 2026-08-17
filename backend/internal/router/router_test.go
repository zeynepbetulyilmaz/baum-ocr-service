package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"baum-ocr/backend/internal/ocr"
)

// Bu test, tek tek handler'ları değil, router.New ile kurulan TÜM zinciri
// (middleware sırası, route wiring, CORS, auth) uçtan uca doğrular. Handler
// testleri her parçanın kendi başına doğru çalıştığını gösteriyor; bu test
// parçaların BİRLİKTE doğru bağlandığını gösteriyor (ör. yanlış middleware
// sırası, yanlış route grubu gibi hatalar handler testinde yakalanmaz).
func newTestRouter(t *testing.T) (*sqlmock.Sqlmock, *httptest.Server) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock oluşturulamadı: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	r := New(Options{
		DB:             db,
		JWTSecret:      "test-secret",
		StorageDir:     t.TempDir(),
		Queue:          ocr.NewQueue(10),
		DefaultLang:    "tur+eng",
		MaxUploadMB:    25,
		FrontendOrigin: "http://localhost:3001",
	})

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return &mock, srv
}

func TestRouter_Health(t *testing.T) {
	mock, srv := newTestRouter(t)
	(*mock).ExpectPing()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("health isteği başarısız: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("beklenen 200, alınan %d", resp.StatusCode)
	}
}

func TestRouter_ProtectedRoute_NoToken(t *testing.T) {
	_, srv := newTestRouter(t)

	resp, err := http.Get(srv.URL + "/api/documents")
	if err != nil {
		t.Fatalf("istek başarısız: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("token'sız istek için beklenen 401, alınan %d", resp.StatusCode)
	}
}

func TestRouter_RegisterThenAccessProtectedRoute(t *testing.T) {
	mock, srv := newTestRouter(t)

	(*mock).ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	(*mock).ExpectExec("INSERT INTO users").WillReturnResult(sqlmock.NewResult(1, 1))

	registerResp, err := http.Post(srv.URL+"/api/auth/register", "application/json",
		strings.NewReader(`{"username":"zeynep","email":"zeynep@baum.edu.tr","password":"sifre123"}`))
	if err != nil {
		t.Fatalf("register isteği başarısız: %v", err)
	}
	defer registerResp.Body.Close()
	if registerResp.StatusCode != http.StatusCreated {
		t.Fatalf("register: beklenen 201, alınan %d", registerResp.StatusCode)
	}

	var registerBody struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(registerResp.Body).Decode(&registerBody); err != nil {
		t.Fatalf("register yanıtı parse edilemedi: %v", err)
	}
	if registerBody.Token == "" {
		t.Fatal("register yanıtında token yok")
	}

	// Korumalı rotaya bu token ile erişim, RequireAuth middleware'inin
	// token_version'ı DB'den kontrol edeceği anlamına gelir.
	(*mock).ExpectQuery("SELECT token_version, role, username FROM users").
		WillReturnRows(sqlmock.NewRows([]string{"token_version", "role", "username"}).AddRow(1, "user", "zeynep"))
	(*mock).ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	(*mock).ExpectQuery("SELECT id, original_filename").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "original_filename", "status", "page_count", "error_message", "created_at", "updated_at",
		}))

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/documents", nil)
	req.Header.Set("Authorization", "Bearer "+registerBody.Token)
	docsResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("documents isteği başarısız: %v", err)
	}
	defer docsResp.Body.Close()

	if docsResp.StatusCode != http.StatusOK {
		t.Fatalf("beklenen 200, alınan %d", docsResp.StatusCode)
	}
}

// TestRouter_AdminRoute_NonAdminForbidden, RequireAdmin middleware'inin
// sadece frontend'de linki gizlemekle kalmayıp gerçekten backend'de de
// yetkisiz erişimi engellediğini doğrular.
func TestRouter_AdminRoute_NonAdminForbidden(t *testing.T) {
	mock, srv := newTestRouter(t)

	(*mock).ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	(*mock).ExpectExec("INSERT INTO users").WillReturnResult(sqlmock.NewResult(1, 1))

	registerResp, err := http.Post(srv.URL+"/api/auth/register", "application/json",
		strings.NewReader(`{"username":"azra","email":"azra@baum.edu.tr","password":"sifre123"}`))
	if err != nil {
		t.Fatalf("register isteği başarısız: %v", err)
	}
	defer registerResp.Body.Close()

	var registerBody struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(registerResp.Body).Decode(&registerBody); err != nil {
		t.Fatalf("register yanıtı parse edilemedi: %v", err)
	}

	(*mock).ExpectQuery("SELECT token_version, role, username FROM users").
		WillReturnRows(sqlmock.NewRows([]string{"token_version", "role", "username"}).AddRow(1, "user", "azra"))

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+registerBody.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("admin isteği başarısız: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("normal kullanıcı için beklenen 403, alınan %d", resp.StatusCode)
	}
}
