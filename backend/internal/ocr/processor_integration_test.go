//go:build integration

package ocr

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestProcessor_Process_RealTesseract(t *testing.T) {
	if _, err := exec.LookPath("tesseract"); err != nil {
		t.Skip("tesseract kurulu değil, integration testi atlanıyor")
	}

	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "input.png")

	img := image.NewRGBA(image.Rect(0, 0, 100, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.White)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png encode hatası: %v", err)
	}
	if err := os.WriteFile(imgPath, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("test görüntüsü yazılamadı: %v", err)
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock oluşturulamadı: %v", err)
	}
	defer db.Close()
	mock.MatchExpectationsInOrder(false)
	mock.ExpectExec("UPDATE documents SET status=").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE documents SET status='done'").WillReturnResult(sqlmock.NewResult(0, 1))

	p := &Processor{DB: db, StorageDir: tmpDir}
	p.Process(Job{DocumentID: "test-doc", FilePath: imgPath, Ext: ".png", Lang: "eng"})

	resultTxt := filepath.Join(tmpDir, "results", "test-doc", "result.txt")
	if _, err := os.Stat(resultTxt); err != nil {
		t.Errorf("result.txt oluşturulmadı: %v", err)
	}
	resultPdf := filepath.Join(tmpDir, "results", "test-doc", "result.pdf")
	if _, err := os.Stat(resultPdf); err != nil {
		t.Errorf("result.pdf oluşturulmadı: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("beklenmeyen sqlmock durumu: %v", err)
	}
}
