package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"baum-ocr/backend/internal/auth"
)

type AuthHandler struct {
	DB        *sql.DB
	JWTSecret string
}

type registerRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	username := strings.TrimSpace(req.Username)

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "şifre işlenemedi"})
		return
	}

	id := uuid.NewString()
	_, err = h.DB.Exec(
		`INSERT INTO users (id, username, email, password_hash, token_version) VALUES ($1, $2, $3, $4, 1)`,
		id, username, req.Email, hash,
	)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "bu e-posta veya kullanıcı adı zaten kayıtlı"})
		return
	}

	token, err := auth.GenerateToken(h.JWTSecret, id, 1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token oluşturulamadı"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"token": token,
		"user":  gin.H{"id": id, "username": username, "email": req.Email},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var id, username, hash string
	var tokenVersion int
	err := h.DB.QueryRow(
		`SELECT id, username, password_hash, token_version FROM users WHERE email = $1`, req.Email,
	).Scan(&id, &username, &hash, &tokenVersion)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "e-posta veya şifre hatalı"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sunucu hatası"})
		return
	}
	if !auth.CheckPassword(hash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "e-posta veya şifre hatalı"})
		return
	}

	token, err := auth.GenerateToken(h.JWTSecret, id, tokenVersion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token oluşturulamadı"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  gin.H{"id": id, "username": username, "email": req.Email},
	})
}

// LogoutAllDevices, kullanıcının token_version'ını artırarak o kullanıcıya
// ait DAHA ÖNCE üretilmiş tüm JWT'leri (şu an tarayıcıda olan dahil) anında
// geçersiz kılar. "Hesabım ele geçirilmiş olabilir" ya da "başka bir
// cihazdan da giriş yapmıştım, hepsini kapat" senaryoları için.
func (h *AuthHandler) LogoutAllDevices(c *gin.Context) {
	userID := c.GetString("user_id")

	if _, err := h.DB.Exec(`UPDATE users SET token_version = token_version + 1 WHERE id = $1`, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "oturumlar kapatılamadı"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "tüm cihazlardaki oturumlar kapatıldı"})
}
