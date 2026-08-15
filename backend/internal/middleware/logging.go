package middleware

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type logLine struct {
	Time      string `json:"time"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Status    int    `json:"status"`
	LatencyMs int64  `json:"latency_ms"`
	RequestID string `json:"request_id"`
	ClientIP  string `json:"client_ip"`
}

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		reqID := uuid.NewString()
		c.Set("request_id", reqID)
		c.Writer.Header().Set("X-Request-Id", reqID)

		c.Next()

		line := logLine{
			Time:      time.Now().UTC().Format(time.RFC3339),
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			Status:    c.Writer.Status(),
			LatencyMs: time.Since(start).Milliseconds(),
			RequestID: reqID,
			ClientIP:  c.ClientIP(),
		}
		if b, err := json.Marshal(line); err == nil {
			log.Println(string(b))
		}
	}
}
