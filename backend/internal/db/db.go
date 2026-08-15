package db

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Connect verilen DSN ile PostgreSQL'e bağlanır, bağlantıyı doğrular ve
// bağlantı havuzunu makul sınırlarla yapılandırır. Sınırsız bağlantı sayısı,
// backend birden fazla instance ile çalıştırıldığında Postgres'in
// max_connections limitini kolayca doldurabilir; bu yüzden açıkça sınırlıyoruz.
func Connect(dsn string) (*sql.DB, error) {
	database, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	database.SetMaxOpenConns(20)
	database.SetMaxIdleConns(5)
	database.SetConnMaxLifetime(30 * time.Minute)
	database.SetConnMaxIdleTime(5 * time.Minute)

	if err := database.Ping(); err != nil {
		return nil, err
	}
	return database, nil
}

// Migrate migrations klasöründeki .sql dosyalarını sırayla, daha önce
// uygulanmamış olanları çalıştırır. Uygulanan migrasyonlar schema_migrations
// tablosunda takip edilir.
func Migrate(database *sql.DB) error {
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`); err != nil {
		return fmt.Errorf("schema_migrations tablosu oluşturulamadı: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		var already bool
		if err := database.QueryRow(
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, name,
		).Scan(&already); err != nil {
			return fmt.Errorf("migrasyon kontrolü başarısız (%s): %w", name, err)
		}
		if already {
			continue
		}

		content, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}

		tx, err := database.Begin()
		if err != nil {
			return fmt.Errorf("migrasyon transaction başlatılamadı (%s): %w", name, err)
		}
		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migrasyon %s başarısız: %w", name, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
			tx.Rollback()
			return fmt.Errorf("migrasyon kaydı yazılamadı (%s): %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migrasyon commit hatası (%s): %w", name, err)
		}
	}
	return nil
}
