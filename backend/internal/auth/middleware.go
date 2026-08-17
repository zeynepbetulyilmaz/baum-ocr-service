package auth

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func RequireAuth(secret string, database *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "yetkilendirme basligi eksik"})
			return
		}

		tokenString := strings.TrimPrefix(header, "Bearer ")

		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "gecersiz veya suresi dolmus oturum"})
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "gecersiz oturum"})
			return
		}

		var currentTokenVersion int
		var role string
		var username string
		err = database.QueryRow(
			`SELECT token_version, role, username FROM users WHERE id = $1`,
			claims.UserID,
		).Scan(&currentTokenVersion, &role, &username)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "kullanici bulunamadi"})
			return
		}

		if currentTokenVersion != claims.TokenVersion {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "oturum gecersiz kilindi, tekrar giris yapin"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", role)
		c.Set("username", username)
		c.Next()
	}
}

// RequireAdmin, RequireAuth'tan SONRA kullanılmalı (role context'te olmalı).
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString("role") != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "bu işlem için yönetici yetkisi gerekiyor"})
			return
		}
		c.Next()
	}
}