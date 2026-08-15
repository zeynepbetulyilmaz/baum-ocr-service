package handlers

import (
	"bytes"
	"database/sql"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"baum-ocr/backend/internal/ocr"
)

type DocumentHandler struct {
	DB             *sql.DB
	StorageDir     string
	Queue          *ocr.Queue
	DefaultLang    string
	MaxUploadBytes int64
}

var allowedExt = map[string]bool{
	".pdf": true, ".png": true, ".jpg": true, ".jpeg": true, ".tif": true, ".tiff": true,
}

var allowedLangs = map[string]bool{"tur": true, "eng": true, "tur+eng": true}

func (h *DocumentHandler) resolveLang(requested string) (string, bool) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		if h.DefaultLang == "" {
			return "tur+eng", true
		}
		return h.DefaultLang, true
	}
	if !allowedLangs[requested] {
		return "", false
	}
	return requested, true
}

func sniffAndValidate(file multipart.File, ext string) bool {
	header := make([]byte, 8)
	n, _ := file.Read(header)
	header = header[:n]
	_, _ = file.Seek(0, 0)

	switch ext {
	case ".pdf":
		return bytes.HasPrefix(header, []byte("%PDF"))
	case ".png":
		return bytes.HasPrefix(header, []byte{0x89, 0x50, 0x4E, 0x47})
	case ".jpg", ".jpeg":
		return bytes.HasPrefix(header, []byte{0xFF, 0xD8, 0xFF})
	case ".tif", ".tiff":
		return bytes.HasPrefix(header, []byte{0x49, 0x49, 0x2A, 0x00}) ||
			bytes.HasPrefix(header, []byte{0x4D, 0x4D, 0x00, 0x2A})
	}
	return false
}

func (h *DocumentHandler) Upload(c *gin.Context) {
	userID := c.GetString("user_id")

	if h.MaxUploadBytes > 0 {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.MaxUploadBytes)
	}

	file, err := c.FormFile("file")
	if err != nil {
		if strings.Contains(err.Error(), "too large") {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("dosya çok büyük (maksimum %d MB)", h.MaxUploadBytes/(1024*1024)),
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "dosya bulunamadı"})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedExt[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "desteklenmeyen dosya türü (pdf, png, jpg, tiff kabul edilir)"})
		return
	}

	lang, ok := h.resolveLang(c.PostForm("lang"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "desteklenmeyen dil (tur, eng, tur+eng kabul edilir)"})
		return
	}

	opened, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "dosya açılamadı"})
		return
	}
	valid := sniffAndValidate(opened, ext)
	opened.Close()
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dosya içeriği belirtilen türle uyuşmuyor"})
		return
	}

	docID := uuid.NewString()
	storedFilename := docID + ext
	uploadDir := filepath.Join(h.StorageDir, "uploads")
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "depolama alanı hazırlanamadı"})
		return
	}
	destPath := filepath.Join(uploadDir, storedFilename)

	if err := c.SaveUploadedFile(file, destPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "dosya kaydedilemedi"})
		return
	}

	_, err = h.DB.Exec(
		`INSERT INTO documents (id, user_id, original_filename, stored_filename, lang, status)
		 VALUES ($1, $2, $3, $4, $5, 'queued')`,
		docID, userID, file.Filename, storedFilename, lang,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "veritabanına kaydedilemedi"})
		return
	}

	h.Queue.Enqueue(ocr.Job{DocumentID: docID, FilePath: destPath, Ext: ext, Lang: lang})

	c.JSON(http.StatusAccepted, gin.H{"id": docID, "status": "queued"})
}

func (h *DocumentHandler) List(c *gin.Context) {
	userID := c.GetString("user_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	var total int
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM documents WHERE user_id=$1`, userID).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "belgeler alınamadı"})
		return
	}

	rows, err := h.DB.Query(
		`SELECT id, original_filename, status, page_count, error_message, created_at, updated_at
		 FROM documents WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, pageSize, offset,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "belgeler alınamadı"})
		return
	}
	defer rows.Close()

	docs := make([]gin.H, 0)
	for rows.Next() {
		var id, filename, status string
		var pageCount int
		var errMsg sql.NullString
		var createdAt, updatedAt interface{}

		if err := rows.Scan(&id, &filename, &status, &pageCount, &errMsg, &createdAt, &updatedAt); err != nil {
			continue
		}

		docs = append(docs, gin.H{
			"id":                id,
			"original_filename": filename,
			"status":            status,
			"page_count":        pageCount,
			"error_message":     errMsg.String,
			"created_at":        createdAt,
			"updated_at":        updatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     docs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *DocumentHandler) Get(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")

	var filename, status string
	var pageCount int
	var errMsg, textPath, pdfPath sql.NullString

	err := h.DB.QueryRow(
		`SELECT original_filename, status, page_count, error_message, text_path, pdf_path
		 FROM documents WHERE id=$1 AND user_id=$2`,
		id, userID,
	).Scan(&filename, &status, &pageCount, &errMsg, &textPath, &pdfPath)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "belge bulunamadı"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sunucu hatası"})
		return
	}

	var text string
	if textPath.Valid {
		if data, readErr := os.ReadFile(textPath.String); readErr == nil {
			text = string(data)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                id,
		"original_filename": filename,
		"status":            status,
		"page_count":        pageCount,
		"error_message":     errMsg.String,
		"text":              text,
		"has_pdf":           pdfPath.Valid,
	})
}

func (h *DocumentHandler) Delete(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")

	var storedFilename sql.NullString
	err := h.DB.QueryRow(
		`SELECT stored_filename FROM documents WHERE id=$1 AND user_id=$2`, id, userID,
	).Scan(&storedFilename)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "belge bulunamadı"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sunucu hatası"})
		return
	}

	if _, err := h.DB.Exec(`DELETE FROM documents WHERE id=$1 AND user_id=$2`, id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "belge silinemedi"})
		return
	}

	if storedFilename.Valid {
		_ = os.Remove(filepath.Join(h.StorageDir, "uploads", storedFilename.String))
	}
	_ = os.RemoveAll(filepath.Join(h.StorageDir, "results", id))

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *DocumentHandler) DownloadText(c *gin.Context) {
	h.download(c, "text_path", ".txt")
}

func (h *DocumentHandler) DownloadPDF(c *gin.Context) {
	h.download(c, "pdf_path", ".pdf")
}

func (h *DocumentHandler) download(c *gin.Context, column, ext string) {
	userID := c.GetString("user_id")
	id := c.Param("id")

	var path sql.NullString
	var filename string
	query := fmt.Sprintf(`SELECT %s, original_filename FROM documents WHERE id=$1 AND user_id=$2`, column)
	err := h.DB.QueryRow(query, id, userID).Scan(&path, &filename)
	if err != nil || !path.Valid {
		c.JSON(http.StatusNotFound, gin.H{"error": "sonuç bulunamadı"})
		return
	}

	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	c.FileAttachment(path.String, base+ext)
}
