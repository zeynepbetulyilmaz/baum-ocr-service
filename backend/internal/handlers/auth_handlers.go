package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"baum-ocr/backend/internal/auth"
	"baum-ocr/backend/internal/mailer"
)

const (
	maxFailedLoginAttempts   = 5
	accountLockoutDuration   = 15 * time.Minute
	passwordResetTokenExpiry = 30 * time.Minute
)

type AuthHandler struct {
	DB             *sql.DB
	JWTSecret      string
	StorageDir     string
	FrontendOrigin string
	Mailer         mailer.Config
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

	// Sistemde hiç kullanıcı yoksa (ilk kayıt), bu kullanıcı otomatik admin
	// olur. Sonraki kayıtlar normal 'user' rolüyle başlar.
	var userCount int
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sunucu hatası"})
		return
	}
	role := "user"
	if userCount == 0 {
		role = "admin"
	}

	id := uuid.NewString()
	_, err = h.DB.Exec(
		`INSERT INTO users (id, username, email, password_hash, role, token_version) VALUES ($1, $2, $3, $4, $5, 1)`,
		id, username, req.Email, hash, role,
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
		"user":  gin.H{"id": id, "username": username, "email": req.Email, "role": role},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var id, username, hash, role string
	var tokenVersion int
	var failedAttempts int
	var lockedUntil sql.NullTime

	err := h.DB.QueryRow(
		`SELECT id, username, password_hash, token_version, role, failed_login_attempts, locked_until
		 FROM users WHERE email = $1`, req.Email,
	).Scan(&id, &username, &hash, &tokenVersion, &role, &failedAttempts, &lockedUntil)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "e-posta veya şifre hatalı"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sunucu hatası"})
		return
	}

	if lockedUntil.Valid && lockedUntil.Time.After(time.Now()) {
		remaining := time.Until(lockedUntil.Time).Round(time.Minute)
		c.JSON(http.StatusLocked, gin.H{
			"error": fmt.Sprintf("çok fazla başarısız deneme, hesap kilitli. yaklaşık %s sonra tekrar deneyin", remaining),
		})
		return
	}

	if !auth.CheckPassword(hash, req.Password) {
		failedAttempts++
		if failedAttempts >= maxFailedLoginAttempts {
			lockUntil := time.Now().Add(accountLockoutDuration)
			_, _ = h.DB.Exec(
				`UPDATE users SET failed_login_attempts = $1, locked_until = $2 WHERE id = $3`,
				failedAttempts, lockUntil, id,
			)
			c.JSON(http.StatusLocked, gin.H{"error": "çok fazla başarısız deneme, hesap 15 dakika kilitlendi"})
			return
		}
		_, _ = h.DB.Exec(`UPDATE users SET failed_login_attempts = $1 WHERE id = $2`, failedAttempts, id)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "e-posta veya şifre hatalı"})
		return
	}

	_, _ = h.DB.Exec(`UPDATE users SET failed_login_attempts = 0, locked_until = NULL WHERE id = $1`, id)

	token, err := auth.GenerateToken(h.JWTSecret, id, tokenVersion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token oluşturulamadı"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  gin.H{"id": id, "username": username, "email": req.Email, "role": role},
	})
}

type updateProfileRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Email    string `json:"email" binding:"required,email"`
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetString("user_id")

	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	username := strings.TrimSpace(req.Username)

	_, err := h.DB.Exec(
		`UPDATE users SET username = $1, email = $2 WHERE id = $3`,
		username, req.Email, userID,
	)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "bu e-posta veya kullanıcı adı zaten kullanılıyor"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{"id": userID, "username": username, "email": req.Email},
	})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := c.GetString("user_id")

	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var currentHash string
	if err := h.DB.QueryRow(`SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&currentHash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sunucu hatası"})
		return
	}
	if !auth.CheckPassword(currentHash, req.CurrentPassword) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "mevcut şifre hatalı"})
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "şifre işlenemedi"})
		return
	}

	// Şifre değişince token_version'ı da artırıyoruz — o ana kadarki tüm
	// oturumlar (çalınmış bir token dahil) otomatik geçersiz olur.
	if _, err := h.DB.Exec(
		`UPDATE users SET password_hash = $1, token_version = token_version + 1 WHERE id = $2`,
		newHash, userID,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "şifre güncellenemedi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "şifre güncellendi, tekrar giriş yapmalısın"})
}

func (h *AuthHandler) LogoutAllDevices(c *gin.Context) {
	userID := c.GetString("user_id")

	if _, err := h.DB.Exec(`UPDATE users SET token_version = token_version + 1 WHERE id = $1`, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "oturumlar kapatılamadı"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "tüm cihazlardaki oturumlar kapatıldı"})
}

type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ForgotPassword bir sıfırlama token'ı üretir ve DB'ye kaydeder. Not: bu
// projede gerçek bir SMTP sağlayıcısı bağlı değil (bkz. README "Bilinen
// Sınırlamalar"), bu yüzden bağlantı e-posta yerine backend loglarına
// yazılıyor — `docker compose logs backend` ile görülebilir. Gerçek bir
// ortamda buradaki log.Printf satırı yerine net/smtp (veya SendGrid vb.)
// ile gerçek e-posta gönderimi eklenmeli.
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var userID string
	err := h.DB.QueryRow(`SELECT id FROM users WHERE email = $1`, req.Email).Scan(&userID)
	if err != nil {
		// Kullanıcı bulunamasa bile aynı yanıtı dönüyoruz — aksi halde bu uç
		// nokta hangi e-postaların sistemde kayıtlı olduğunu tahmin etmek
		// için kullanılabilir (user enumeration saldırısı).
		c.JSON(http.StatusOK, gin.H{"status": "eğer bu e-posta kayıtlıysa, sıfırlama bağlantısı gönderildi"})
		return
	}

	token := uuid.NewString()
	expiresAt := time.Now().Add(passwordResetTokenExpiry)
	if _, err := h.DB.Exec(
		`INSERT INTO password_reset_tokens (token, user_id, expires_at) VALUES ($1, $2, $3)`,
		token, userID, expiresAt,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sunucu hatası"})
		return
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", h.FrontendOrigin, token)

	if h.Mailer.Configured() {
		subject := "BAUM PDF OCR Servisi - Şifre Sıfırlama"
		body := fmt.Sprintf(
			"Merhaba,\n\nŞifreni sıfırlamak için aşağıdaki bağlantıya tıkla (30 dakika geçerlidir):\n\n%s\n\nBu isteği sen yapmadıysan bu e-postayı görmezden gelebilirsin.",
			resetLink,
		)
		if err := h.Mailer.Send(req.Email, subject, body); err != nil {
			// E-posta gönderimi başarısız olsa bile kullanıcıya aynı jenerik
			// yanıtı döneriz (enumeration'ı önlemek için); hatayı ve
			// bağlantıyı sadece loglara yazarız ki geliştirme sırasında
			// takip edilebilsin.
			log.Printf("[şifre sıfırlama] %s için e-posta gönderilemedi (%v), bağlantı: %s", req.Email, err, resetLink)
		}
	} else {
		log.Printf("[şifre sıfırlama] SMTP yapılandırılmamış — %s için bağlantı: %s (30 dakika geçerli)", req.Email, resetLink)
	}

	c.JSON(http.StatusOK, gin.H{"status": "eğer bu e-posta kayıtlıysa, sıfırlama bağlantısı gönderildi"})
}

type resetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var userID string
	var expiresAt time.Time
	var used bool
	err := h.DB.QueryRow(
		`SELECT user_id, expires_at, used FROM password_reset_tokens WHERE token = $1`, req.Token,
	).Scan(&userID, &expiresAt, &used)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "geçersiz veya süresi dolmuş bağlantı"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sunucu hatası"})
		return
	}
	if used || time.Now().After(expiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "geçersiz veya süresi dolmuş bağlantı"})
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "şifre işlenemedi"})
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sunucu hatası"})
		return
	}

	if _, err := tx.Exec(
		`UPDATE users SET password_hash = $1, token_version = token_version + 1,
		 failed_login_attempts = 0, locked_until = NULL WHERE id = $2`,
		newHash, userID,
	); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "şifre güncellenemedi"})
		return
	}
	if _, err := tx.Exec(`UPDATE password_reset_tokens SET used = TRUE WHERE token = $1`, req.Token); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "şifre güncellenemedi"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "şifre güncellenemedi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "şifre güncellendi, şimdi giriş yapabilirsin"})
}

// DeleteMyAccount kullanıcının kendi hesabını ve tüm belgelerini (DB kaydı +
// diskteki dosyalar) kalıcı olarak siler. KVKK'nin "unutulma hakkı" ilkesine
// karşılık gelen, kullanıcının kendi başlattığı bir işlemdir (admin panelinden
// yapılan silme ile karıştırılmamalı, bu ayrı ve bağımsız bir uç nokta).
func (h *AuthHandler) DeleteMyAccount(c *gin.Context) {
	userID := c.GetString("user_id")

	rows, err := h.DB.Query(`SELECT id, stored_filename FROM documents WHERE user_id = $1`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sunucu hatası"})
		return
	}
	type ownedDoc struct {
		id             string
		storedFilename sql.NullString
	}
	var owned []ownedDoc
	for rows.Next() {
		var d ownedDoc
		if err := rows.Scan(&d.id, &d.storedFilename); err == nil {
			owned = append(owned, d)
		}
	}
	rows.Close()

	if _, err := h.DB.Exec(`DELETE FROM documents WHERE user_id = $1`, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "belgeler silinemedi"})
		return
	}
	if _, err := h.DB.Exec(`DELETE FROM users WHERE id = $1`, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hesap silinemedi"})
		return
	}

	for _, d := range owned {
		if d.storedFilename.Valid {
			_ = os.Remove(filepath.Join(h.StorageDir, "uploads", d.storedFilename.String))
		}
		_ = os.RemoveAll(filepath.Join(h.StorageDir, "results", d.id))
	}

	c.JSON(http.StatusOK, gin.H{"status": "hesabın ve tüm verilerin kalıcı olarak silindi"})
}