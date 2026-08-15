package ocr

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const commandTimeout = 3 * time.Minute

type Processor struct {
	DB         *sql.DB
	StorageDir string
}

func (p *Processor) Process(j Job) {
	workDir := filepath.Join(p.StorageDir, "work", j.DocumentID)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		p.setStatus(j.DocumentID, "failed", "çalışma dizini oluşturulamadı")
		return
	}
	defer os.RemoveAll(workDir)

	p.setStatus(j.DocumentID, "processing", "")

	var pageImages []string
	var err error

	if j.Ext == ".pdf" {
		pageImages, err = rasterizePDF(j.FilePath, workDir)
		if err != nil {
			p.setStatus(j.DocumentID, "failed", err.Error())
			return
		}
	} else {
		pageImages = []string{j.FilePath}
	}

	if len(pageImages) == 0 {
		p.setStatus(j.DocumentID, "failed", "belgede sayfa bulunamadı")
		return
	}

	lang := j.Lang
	if lang == "" {
		lang = "tur+eng"
	}

	var textParts []string
	var pagePDFs []string

	for i, img := range pageImages {
		outBase := filepath.Join(workDir, fmt.Sprintf("page-%03d", i+1))

		ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
		cmd := exec.CommandContext(ctx, "tesseract", img, outBase, "-l", lang, "txt", "pdf")
		out, cmdErr := cmd.CombinedOutput()
		cancel()
		if cmdErr != nil {
			p.setStatus(j.DocumentID, "failed", fmt.Sprintf("tesseract hatası (sayfa %d): %s", i+1, string(out)))
			return
		}

		textBytes, readErr := os.ReadFile(outBase + ".txt")
		if readErr != nil {
			p.setStatus(j.DocumentID, "failed", fmt.Sprintf("sayfa %d metni okunamadı", i+1))
			return
		}

		pagePDFPath := outBase + ".pdf"
		if _, statErr := os.Stat(pagePDFPath); statErr != nil {
			p.setStatus(j.DocumentID, "failed", fmt.Sprintf(
				"sayfa %d için aranabilir PDF üretilemedi (tessdata içinde configs/pdf ve pdf.ttf bulunduğundan emin olun)", i+1,
			))
			return
		}

		textParts = append(textParts, string(textBytes))
		pagePDFs = append(pagePDFs, pagePDFPath)
	}

	finalText := strings.Join(textParts, "\n\n----- sayfa sonu -----\n\n")

	resultsDir := filepath.Join(p.StorageDir, "results", j.DocumentID)
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		p.setStatus(j.DocumentID, "failed", "sonuç dizini oluşturulamadı")
		return
	}

	textPath := filepath.Join(resultsDir, "result.txt")
	if err := os.WriteFile(textPath, []byte(finalText), 0o644); err != nil {
		p.setStatus(j.DocumentID, "failed", "metin dosyası yazılamadı")
		return
	}

	pdfPath := filepath.Join(resultsDir, "result.pdf")
	if len(pagePDFs) == 1 {
		if err := copyFile(pagePDFs[0], pdfPath); err != nil {
			p.setStatus(j.DocumentID, "failed", "pdf dosyası kopyalanamadı")
			return
		}
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
		args := append(append([]string{}, pagePDFs...), pdfPath)
		cmd := exec.CommandContext(ctx, "pdfunite", args...)
		out, cmdErr := cmd.CombinedOutput()
		cancel()
		if cmdErr != nil {
			p.setStatus(j.DocumentID, "failed", fmt.Sprintf("pdf birleştirme hatası: %s", string(out)))
			return
		}
	}

	_, err = p.DB.Exec(
		`UPDATE documents SET status='done', page_count=$1, text_path=$2, pdf_path=$3, updated_at=now() WHERE id=$4`,
		len(pageImages), textPath, pdfPath, j.DocumentID,
	)
	if err != nil {
		p.setStatus(j.DocumentID, "failed", "veritabanı güncellenemedi")
	}
}

func (p *Processor) setStatus(docID, status, errMsg string) {
	if errMsg == "" {
		_, _ = p.DB.Exec(`UPDATE documents SET status=$1, updated_at=now() WHERE id=$2`, status, docID)
		return
	}
	_, _ = p.DB.Exec(`UPDATE documents SET status=$1, error_message=$2, updated_at=now() WHERE id=$3`, status, errMsg, docID)
}

func rasterizePDF(pdfPath, workDir string) ([]string, error) {
	prefix := filepath.Join(workDir, "src")

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "pdftoppm", "-r", "300", "-png", pdfPath, prefix)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("pdftoppm hatası: %s", string(out))
	}

	matches, err := filepath.Glob(prefix + "*.png")
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}
