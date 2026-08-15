package router

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"baum-ocr/backend/internal/auth"
	"baum-ocr/backend/internal/handlers"
	"baum-ocr/backend/internal/middleware"
	"baum-ocr/backend/internal/ocr"
)

type Options struct {
	DB             *sql.DB
	JWTSecret      string
	StorageDir     string
	Queue          *ocr.Queue
	DefaultLang    string
	MaxUploadMB    int
	FrontendOrigin string
}

func New(opt Options) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger())

	if err := r.SetTrustedProxies([]string{"172.16.0.0/12"}); err != nil {
		panic(err)
	}

	corsConfig := cors.Config{
		AllowOrigins:     []string{opt.FrontendOrigin},
		AllowMethods:     []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"X-Request-Id"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
	r.Use(cors.New(corsConfig))

	authHandler := &handlers.AuthHandler{DB: opt.DB, JWTSecret: opt.JWTSecret}
	docHandler := &handlers.DocumentHandler{
		DB:             opt.DB,
		StorageDir:     opt.StorageDir,
		Queue:          opt.Queue,
		DefaultLang:    opt.DefaultLang,
		MaxUploadBytes: int64(opt.MaxUploadMB) * 1024 * 1024,
	}

	r.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := opt.DB.PingContext(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "error": "veritabanına ulaşılamıyor"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	authLimiter := middleware.NewRateLimiter(10, time.Minute)

	api := r.Group("/api")
	{
		authGroup := api.Group("/auth")
		authGroup.Use(authLimiter.Middleware())
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)

		secured := api.Group("/")
		secured.Use(auth.RequireAuth(opt.JWTSecret, opt.DB))
		{
			secured.POST("/auth/logout-all", authHandler.LogoutAllDevices)
			secured.POST("/documents", docHandler.Upload)
			secured.GET("/documents", docHandler.List)
			secured.GET("/documents/:id", docHandler.Get)
			secured.GET("/documents/:id/text", docHandler.DownloadText)
			secured.GET("/documents/:id/pdf", docHandler.DownloadPDF)
			secured.DELETE("/documents/:id", docHandler.Delete)
		}
	}

	return r
}
