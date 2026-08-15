package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"baum-ocr/backend/internal/config"
	"baum-ocr/backend/internal/db"
	"baum-ocr/backend/internal/ocr"
	"baum-ocr/backend/internal/router"
)

func main() {
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	cfg := config.Load()

	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("veritabanına bağlanılamadı: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		log.Fatalf("migrasyon hatası: %v", err)
	}

	queue := ocr.NewQueue(100)
	processor := &ocr.Processor{DB: database, StorageDir: cfg.StorageDir}
	queue.StartWorkers(2, processor.Process)

	if err := ocr.Reconcile(database, cfg.StorageDir, queue); err != nil {
		log.Printf("kuyruk onarımı sırasında hata: %v", err)
	}

	r := router.New(router.Options{
		DB:             database,
		JWTSecret:      cfg.JWTSecret,
		StorageDir:     cfg.StorageDir,
		Queue:          queue,
		DefaultLang:    cfg.TesseractLang,
		MaxUploadMB:    cfg.MaxUploadMB,
		FrontendOrigin: cfg.FrontendOrigin,
	})

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// Sunucuyu ayrı bir goroutine'de başlatıyoruz ki ana goroutine
	// kapanış sinyalini (SIGTERM/SIGINT) bekleyebilsin. "docker compose
	// down"/"docker stop" bir SIGTERM gönderir; bunu yakalamazsak sunucu
	// anında (mid-request) kesilir. Yakalayınca, devam eden HTTP
	// isteklerinin bitmesi için (en fazla 10 saniye) bekliyoruz.
	go func() {
		log.Printf("sunucu :%s portunda başlıyor", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("sunucu başlatılamadı: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("kapanış sinyali alındı, devam eden istekler tamamlanıyor...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("temiz kapanış başarısız oldu, zorla kapatılıyor: %v", err)
	}
	log.Println("sunucu kapatıldı")
}
