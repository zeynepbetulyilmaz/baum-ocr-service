package auth

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// RequireAuth, Authorization header'ındaki JWT'yi doğrular VE token
// içindeki token_version'ın veritabanındaki güncel değerle eşleştiğini
// kontrol eder. Bu ikinci kontrol olmadan JWT'ler "revoke edilemez" olurdu
// — süresi dolana kadar (72 saat) her zaman geçerli kalırlardı, çalınan bir
// token'ı iptal etmenin hiçbir yolu olmazdı. Bunun bedeli, her istekte bir
// ekstra (çok ucuz, indexli) DB sorgusu.
func RequireAuth(secret string, database *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "eksik veya hatalı yetkilendirme başlığı"})
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := ParseToken(secret, tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "geçersiz veya süresi dolmuş token"})
			return
		}

		var currentVersion int
		if err := database.QueryRow(
			`SELECT token_version FROM users WHERE id = $1`, claims.UserID,
		).Scan(&currentVersion); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "kullanıcı bulunamadı"})
			return
		}
		if currentVersion != claims.TokenVersion {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "oturum geçersiz kılınmış, tekrar giriş yapın"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Next()
	}
}
