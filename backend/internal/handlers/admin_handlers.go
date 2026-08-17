package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	DB *sql.DB
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	rows, err := h.DB.Query(`
		SELECT u.id, u.username, u.email, u.role, u.created_at,
		       COUNT(d.id) AS document_count
		FROM users u
		LEFT JOIN documents d ON d.user_id = u.id
		GROUP BY u.id
		ORDER BY u.created_at ASC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "kullanicilar alinamadi"})
		return
	}
	defer rows.Close()

	users := make([]gin.H, 0)
	for rows.Next() {
		var id, username, email, role string
		var createdAt interface{}
		var docCount int
		if err := rows.Scan(&id, &username, &email, &role, &createdAt, &docCount); err != nil {
			continue
		}
		users = append(users, gin.H{
			"id": id, "username": username, "email": email, "role": role,
			"created_at": createdAt, "document_count": docCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{"items": users})
}

func (h *AdminHandler) ListAllDocuments(c *gin.Context) {
	rows, err := h.DB.Query(`
		SELECT d.id, d.original_filename, d.status, d.page_count, d.created_at,
		       u.username, u.email
		FROM documents d
		JOIN users u ON u.id = d.user_id
		ORDER BY d.created_at DESC
		LIMIT 200
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "belgeler alinamadi"})
		return
	}
	defer rows.Close()

	docs := make([]gin.H, 0)
	for rows.Next() {
		var id, filename, status, username, email string
		var pageCount int
		var createdAt interface{}
		if err := rows.Scan(&id, &filename, &status, &pageCount, &createdAt, &username, &email); err != nil {
			continue
		}
		docs = append(docs, gin.H{
			"id": id, "original_filename": filename, "status": status,
			"page_count": pageCount, "created_at": createdAt,
			"owner_username": username, "owner_email": email,
		})
	}

	c.JSON(http.StatusOK, gin.H{"items": docs})
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	targetID := c.Param("id")
	requesterID := c.GetString("user_id")
	requesterUsername := c.GetString("username")

	if targetID == requesterID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "kendi hesabini admin panelinden silemezsin"})
		return
	}

	var targetUsername string
	if err := h.DB.QueryRow(`SELECT username FROM users WHERE id = $1`, targetID).Scan(&targetUsername); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "kullanici bulunamadi"})
		return
	}

	res, err := h.DB.Exec(`DELETE FROM users WHERE id = $1`, targetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "kullanici silinemedi"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "kullanici bulunamadi"})
		return
	}

	_, _ = h.DB.Exec(
		`INSERT INTO audit_logs (id, actor_id, actor_username, action, target_type, target_id, details)
		 VALUES (gen_random_uuid(), $1, $2, 'delete_user', 'user', $3, $4)`,
		requesterID, requesterUsername, targetID, "silinen kullanici: "+targetUsername,
	)

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *AdminHandler) ListAuditLogs(c *gin.Context) {
	rows, err := h.DB.Query(`
		SELECT id, actor_username, action, target_type, target_id, details, created_at
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT 200
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "kayitlar alinamadi"})
		return
	}
	defer rows.Close()

	logs := make([]gin.H, 0)
	for rows.Next() {
		var id, actor, action, targetType, targetID string
		var details sql.NullString
		var createdAt interface{}
		if err := rows.Scan(&id, &actor, &action, &targetType, &targetID, &details, &createdAt); err != nil {
			continue
		}
		logs = append(logs, gin.H{
			"id": id, "actor_username": actor, "action": action,
			"target_type": targetType, "target_id": targetID,
			"details": details.String, "created_at": createdAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"items": logs})
}