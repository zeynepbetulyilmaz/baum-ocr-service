package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"

	"baum-ocr/backend/internal/ocr"
)

type fakeMultipartFile struct {
	*bytes.Reader
}

func (f fakeMultipartFile) Close() error { return nil }

func newFakeFile(content []byte) multipart.File {
	return fakeMultipartFile{bytes.NewReader(content)}
}

func TestSniffAndValidate(t *testing.T) {
	cases := []struct {
		name    string
		content []byte
		ext     string
		want    bool
	}{
		{"gecerli pdf", []byte("%PDF-1.4 rest of file"), ".pdf", true},
		{"sahte pdf (aslinda metin)", []byte("bu bir pdf degil"), ".pdf", false},
		{"gecerli png", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A}, ".png", true},
		{"sahte png", []byte("PNG degilim"), ".png", false},
		{"gecerli jpg", []byte{0xFF, 0xD8, 0xFF, 0xE0}, ".jpg", true},
		{"desteklenmeyen uzanti", []byte("herhangi bir icerik"), ".exe", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sniffAndValidate(newFakeFile(tc.content), tc.ext)
			if got != tc.want {
				t.Errorf("sniffAndValidate(%q, %q) = %v, beklenen %v", tc.content, tc.ext, got, tc.want)
			}
		})
	}
}

func TestResolveLang(t *testing.T) {
	h := &DocumentHandler{DefaultLang: "tur+eng"}

	if lang, ok := h.resolveLang(""); !ok || lang != "tur+eng" {
		t.Errorf("boş dil için varsayılan bekleniyordu, alınan: %q, ok=%v", lang, ok)
	}
	if lang, ok := h.resolveLang("eng"); !ok || lang != "eng" {
		t.Errorf("'eng' kabul edilmeliydi, alınan: %q, ok=%v", lang, ok)
	}
	if _, ok := h.resolveLang("fra"); ok {
		t.Error("desteklenmeyen dil ('fra') kabul edilmemeliydi")
	}
}

func setupDocRouter(h *DocumentHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user-1")
		c.Next()
	})
	r.POST("/documents", h.Upload)
	r.GET("/documents", h.List)
	r.DELETE("/documents/:id", h.Delete)
	return r
}

func newUploadRequest(t *testing.T, fieldContent []byte, filename, lang string) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("form dosyası oluşturulamadı: %v", err)
	}
	if _, err := fw.Write(fieldContent); err != nil {
		t.Fatalf("dosya içeriği yazılamadı: %v", err)
	}
	if lang != "" {
		if err := w.WriteField("lang", lang); err != nil {
			t.Fatalf("lang alanı yazılamadı: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("multipart writer kapatılamadı: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/documents", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestUpload_UnsupportedExtension(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	h := &DocumentHandler{DB: db, StorageDir: t.TempDir(), Queue: ocr.NewQueue(10), DefaultLang: "tur+eng"}
	r := setupDocRouter(h)

	req := newUploadRequest(t, []byte("herhangi bir icerik"), "belge.exe", "")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("beklenen 400, alınan %d: %s", w.Code, w.Body.String())
	}
}

func TestUpload_InvalidLang(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	h := &DocumentHandler{DB: db, StorageDir: t.TempDir(), Queue: ocr.NewQueue(10), DefaultLang: "tur+eng"}
	r := setupDocRouter(h)

	req := newUploadRequest(t, []byte("%PDF-1.4 icerik"), "belge.pdf", "fra")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("beklenen 400, alınan %d: %s", w.Code, w.Body.String())
	}
}

func TestUpload_ContentMismatch(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	h := &DocumentHandler{DB: db, StorageDir: t.TempDir(), Queue: ocr.NewQueue(10), DefaultLang: "tur+eng"}
	r := setupDocRouter(h)

	req := newUploadRequest(t, []byte("bu asla bir pdf olmayacak"), "sahte.pdf", "")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("beklenen 400, alınan %d: %s", w.Code, w.Body.String())
	}
}

func TestUpload_TooLarge(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	h := &DocumentHandler{
		DB: db, StorageDir: t.TempDir(), Queue: ocr.NewQueue(10),
		DefaultLang: "tur+eng", MaxUploadBytes: 10,
	}
	r := setupDocRouter(h)

	content := bytes.Repeat([]byte("A"), 1000)
	req := newUploadRequest(t, content, "buyuk.pdf", "")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("beklenen 413, alınan %d: %s", w.Code, w.Body.String())
	}
}

func TestUpload_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock oluşturulamadı: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("INSERT INTO documents").
		WillReturnResult(sqlmock.NewResult(1, 1))

	storageDir := t.TempDir()
	queue := ocr.NewQueue(10)
	h := &DocumentHandler{DB: db, StorageDir: storageDir, Queue: queue, DefaultLang: "tur+eng"}
	r := setupDocRouter(h)

	req := newUploadRequest(t, []byte("%PDF-1.4 gecerli icerik"), "rapor.pdf", "eng")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("beklenen 202, alınan %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("yanıt parse edilemedi: %v", err)
	}
	if resp.Status != "queued" {
		t.Errorf("beklenen status 'queued', alınan %q", resp.Status)
	}

	select {
	case job := <-queue.Chan():
		if job.DocumentID != resp.ID {
			t.Errorf("kuyruğa alınan iş id'si (%s) yanıttaki id'yle (%s) uyuşmuyor", job.DocumentID, resp.ID)
		}
		if job.Lang != "eng" {
			t.Errorf("beklenen dil 'eng', alınan %q", job.Lang)
		}
	default:
		t.Error("iş kuyruğa hiç eklenmemiş")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("beklenmeyen sqlmock durumu: %v", err)
	}
}

func TestList_Pagination(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock oluşturulamadı: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT COUNT").
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(12))

	rows := sqlmock.NewRows([]string{
		"id", "original_filename", "status", "page_count", "error_message", "created_at", "updated_at",
	}).AddRow("doc-1", "a.pdf", "done", 2, "", "2026-08-14T10:00:00Z", "2026-08-14T10:01:00Z")

	mock.ExpectQuery("SELECT id, original_filename").
		WithArgs("user-1", 5, 5).
		WillReturnRows(rows)

	h := &DocumentHandler{DB: db}
	r := setupDocRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/documents?page=2&page_size=5", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("beklenen 200, alınan %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Total    int `json:"total"`
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
		Items    []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("yanıt parse edilemedi: %v", err)
	}
	if resp.Total != 12 || resp.Page != 2 || resp.PageSize != 5 {
		t.Errorf("beklenmeyen sayfalama: total=%d page=%d page_size=%d", resp.Total, resp.Page, resp.PageSize)
	}
	if len(resp.Items) != 1 || resp.Items[0].ID != "doc-1" {
		t.Errorf("beklenmeyen items: %+v", resp.Items)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("beklenmeyen sqlmock durumu: %v", err)
	}
}

func TestDelete_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock oluşturulamadı: %v", err)
	}
	defer db.Close()

	storageDir := t.TempDir()
	uploadPath := filepath.Join(storageDir, "uploads")
	resultsPath := filepath.Join(storageDir, "results", "doc-1")
	if err := os.MkdirAll(uploadPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(resultsPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uploadPath, "doc-1.pdf"), []byte("icerik"), 0o644); err != nil {
		t.Fatal(err)
	}

	mock.ExpectQuery("SELECT stored_filename FROM documents").
		WithArgs("doc-1", "user-1").
		WillReturnRows(sqlmock.NewRows([]string{"stored_filename"}).AddRow("doc-1.pdf"))
	mock.ExpectExec("DELETE FROM documents").
		WithArgs("doc-1", "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	h := &DocumentHandler{DB: db, StorageDir: storageDir}
	r := setupDocRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/documents/doc-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("beklenen 200, alınan %d: %s", w.Code, w.Body.String())
	}

	if _, err := os.Stat(filepath.Join(uploadPath, "doc-1.pdf")); !os.IsNotExist(err) {
		t.Error("yüklenen dosya silinmemiş")
	}
	if _, err := os.Stat(resultsPath); !os.IsNotExist(err) {
		t.Error("sonuç dizini silinmemiş")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("beklenmeyen sqlmock durumu: %v", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock oluşturulamadı: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT stored_filename FROM documents").
		WithArgs("doc-yok", "user-1").
		WillReturnError(sql.ErrNoRows)

	h := &DocumentHandler{DB: db, StorageDir: t.TempDir()}
	r := setupDocRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/documents/doc-yok", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("beklenen 404, alınan %d: %s", w.Code, w.Body.String())
	}
}
