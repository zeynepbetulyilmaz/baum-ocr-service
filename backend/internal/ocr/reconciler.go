package ocr

import (
	"database/sql"
	"log"
	"path/filepath"
)

func Reconcile(database *sql.DB, storageDir string, queue *Queue) error {
	rows, err := database.Query(
		`SELECT id, stored_filename, lang FROM documents
		 WHERE status IN ('queued', 'processing') AND stored_filename IS NOT NULL`,
	)
	if err != nil {
		return err
	}

	type pending struct {
		id, storedFilename, lang string
	}
	var items []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.id, &p.storedFilename, &p.lang); err != nil {
			continue
		}
		items = append(items, p)
	}
	rows.Close()

	for _, p := range items {
		ext := filepath.Ext(p.storedFilename)
		filePath := filepath.Join(storageDir, "uploads", p.storedFilename)

		if _, err := database.Exec(
			`UPDATE documents SET status='queued', updated_at=now() WHERE id=$1`, p.id,
		); err != nil {
			log.Printf("belge %s yeniden kuyruğa alınamadı: %v", p.id, err)
			continue
		}

		queue.Enqueue(Job{DocumentID: p.id, FilePath: filePath, Ext: ext, Lang: p.lang})
		log.Printf("belge %s yeniden kuyruğa alındı (sunucu yeniden başlatıldı)", p.id)
	}

	if len(items) > 0 {
		log.Printf("%d belge yeniden kuyruğa alındı", len(items))
	}
	return nil
}
