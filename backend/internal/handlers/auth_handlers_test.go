package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"

	"baum-ocr/backend/internal/auth"
)

func setupAuthRouter(db *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &AuthHandler{DB: db, JWTSecret: "test-secret"}
	r.POST("/api/auth/register", h.Register)
	r.POST("/api/auth/login", h.Login)
	r.POST("/api/auth/logout-all", func(c *gin.Context) {
		c.Set("user_id", "user-1")
		h.LogoutAllDevices(c)
	})
	return r
}

func doJSON(r *gin.Engine, method, path string, payload map[string]string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRegister_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock oluşturulamadı: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("INSERT INTO users").
		WithArgs(sqlmock.AnyArg(), "zeynep", "zeynep@baum.edu.tr", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	r := setupAuthRouter(db)
	w := doJSON(r, http.MethodPost, "/api/auth/register", map[string]string{
		"username": "zeynep", "email": "zeynep@baum.edu.tr", "password": "sifre123",
	})

	if w.Code != http.StatusCreated {
		t.Fatalf("beklenen 201, alınan %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("beklenmeyen sqlmock durumu: %v", err)
	}
}

func TestRegister_DuplicateConflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock oluşturulamadı: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("INSERT INTO users").
		WillReturnError(fmt.Errorf("duplicate key value violates unique constraint"))

	r := setupAuthRouter(db)
	w := doJSON(r, http.MethodPost, "/api/auth/register", map[string]string{
		"username": "zeynep", "email": "zeynep@baum.edu.tr", "password": "sifre123",
	})

	if w.Code != http.StatusConflict {
		t.Fatalf("beklenen 409, alınan %d", w.Code)
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	r := setupAuthRouter(db)
	w := doJSON(r, http.MethodPost, "/api/auth/register", map[string]string{
		"username": "zeynep", "email": "zeynep@baum.edu.tr", "password": "123",
	})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("beklenen 400, alınan %d", w.Code)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock oluşturulamadı: %v", err)
	}
	defer db.Close()

	hash, _ := auth.HashPassword("dogru-sifre")
	rows := sqlmock.NewRows([]string{"id", "username", "password_hash", "token_version"}).
		AddRow("user-1", "zeynep", hash, 1)
	mock.ExpectQuery("SELECT id, username, password_hash, token_version FROM users").
		WithArgs("zeynep@baum.edu.tr").
		WillReturnRows(rows)

	r := setupAuthRouter(db)
	w := doJSON(r, http.MethodPost, "/api/auth/login", map[string]string{
		"email": "zeynep@baum.edu.tr", "password": "yanlis-sifre",
	})

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("beklenen 401, alınan %d", w.Code)
	}
}

func TestLogin_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock oluşturulamadı: %v", err)
	}
	defer db.Close()

	hash, _ := auth.HashPassword("dogru-sifre")
	rows := sqlmock.NewRows([]string{"id", "username", "password_hash", "token_version"}).
		AddRow("user-1", "zeynep", hash, 1)
	mock.ExpectQuery("SELECT id, username, password_hash, token_version FROM users").
		WithArgs("zeynep@baum.edu.tr").
		WillReturnRows(rows)

	r := setupAuthRouter(db)
	w := doJSON(r, http.MethodPost, "/api/auth/login", map[string]string{
		"email": "zeynep@baum.edu.tr", "password": "dogru-sifre",
	})

	if w.Code != http.StatusOK {
		t.Fatalf("beklenen 200, alınan %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		User struct {
			Username string `json:"username"`
		} `json:"user"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("yanıt parse edilemedi: %v", err)
	}
	if resp.User.Username != "zeynep" {
		t.Errorf("beklenen username 'zeynep', alınan '%s'", resp.User.Username)
	}
}

func TestLogoutAllDevices(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock oluşturulamadı: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("UPDATE users SET token_version").
		WithArgs("user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := setupAuthRouter(db)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout-all", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("beklenen 200, alınan %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("beklenmeyen sqlmock durumu: %v", err)
	}
}
