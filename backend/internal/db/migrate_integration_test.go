//go:build integration

package db

import (
	"os"
	"testing"
)

// TestMigrate_RealPostgres, migration'ların gerçek bir Postgres'e karşı
// baştan sona sorunsuz uygulandığını doğrular. sqlmock ile yazılan unit
// testler SQL'in sözdizimini hiç çalıştırmadan "doğru" sayabilir; bu test
// gerçek kısıtları (foreign key, tip uyuşmazlığı, vb.) yakalar.
//
// Çalıştırmak için gerçek bir Postgres'e ihtiyaç var:
//
//	docker compose up -d postgres
//	TEST_DATABASE_URL="postgres://baum:baum@localhost:5432/baum?sslmode=disable" \
//	  go test -tags=integration ./internal/db/...
func TestMigrate_RealPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL ayarlı değil, entegrasyon testi atlanıyor")
	}

	database, err := Connect(dsn)
	if err != nil {
		t.Fatalf("veritabanına bağlanılamadı: %v", err)
	}
	defer database.Close()

	if err := Migrate(database); err != nil {
		t.Fatalf("migrasyon başarısız: %v", err)
	}

	// İkinci çalıştırma idempotent olmalı: zaten uygulanmış migration'lar
	// schema_migrations tablosu sayesinde tekrar çalıştırılmamalı.
	if err := Migrate(database); err != nil {
		t.Fatalf("ikinci migrasyon çalıştırması başarısız olmamalıydı: %v", err)
	}

	requiredTables := []string{
		"users", "documents", "schema_migrations", "audit_logs", "password_reset_tokens",
	}
	for _, table := range requiredTables {
		var exists bool
		err := database.QueryRow(
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)`, table,
		).Scan(&exists)
		if err != nil {
			t.Fatalf("%s tablosu kontrol edilemedi: %v", table, err)
		}
		if !exists {
			t.Errorf("beklenen tablo yok: %s", table)
		}
	}

	requiredUserColumns := []string{"role", "token_version", "failed_login_attempts", "locked_until"}
	for _, col := range requiredUserColumns {
		var exists bool
		err := database.QueryRow(
			`SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = $1)`,
			col,
		).Scan(&exists)
		if err != nil {
			t.Fatalf("users.%s kontrol edilemedi: %v", col, err)
		}
		if !exists {
			t.Errorf("users tablosunda beklenen sütun yok: %s", col)
		}
	}
}
