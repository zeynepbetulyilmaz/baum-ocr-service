package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port           string
	DatabaseURL    string
	JWTSecret      string
	StorageDir     string
	TesseractLang  string
	MaxUploadMB    int
	FrontendOrigin string
}

func Load() Config {
	return Config{
		Port:           getEnv("PORT", "8080"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://baum:baum@localhost:5432/baum_ocr?sslmode=disable"),
		JWTSecret:      getEnv("JWT_SECRET", "change-me-in-production"),
		StorageDir:     getEnv("STORAGE_DIR", "/data/storage"),
		TesseractLang:  getEnv("TESSERACT_LANG", "tur+eng"),
		MaxUploadMB:    getEnvInt("MAX_UPLOAD_MB", 25),
		FrontendOrigin: getEnv("FRONTEND_ORIGIN", "http://localhost:3001"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
