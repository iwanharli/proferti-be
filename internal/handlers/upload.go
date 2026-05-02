package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxUploadSize = 5 * 1024 * 1024 // 5MB
const uploadDir = "./uploads"

// POST /api/upload
func (a *API) UploadFile(w http.ResponseWriter, r *http.Request) {
	// 1. Limit upload size
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		errJSON(w, http.StatusBadRequest, "File terlalu besar. Maksimal 5MB.")
		return
	}

	// 2. Get file from form
	file, handler, err := r.FormFile("file")
	if err != nil {
		errJSON(w, http.StatusBadRequest, "Tidak ada file yang diunggah.")
		return
	}
	defer file.Close()

	// 3. Validate file type
	ext := strings.ToLower(filepath.Ext(handler.Filename))
	allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	if !allowedExts[ext] {
		errJSON(w, http.StatusBadRequest, "Format file tidak didukung. Gunakan JPG, PNG, atau WebP.")
		return
	}

	// 4. Ensure upload directory exists
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		_ = os.MkdirAll(uploadDir, 0755)
	}

	// 5. Create unique filename
	newFilename := fmt.Sprintf("%d-%s%s", time.Now().Unix(), uuid.New().String()[:8], ext)
	dstPath := filepath.Join(uploadDir, newFilename)

	// 6. Save file
	dst, err := os.Create(dstPath)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, "Gagal menyimpan file di server.")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		errJSON(w, http.StatusInternalServerError, "Gagal menyalin data file.")
		return
	}

	// 7. Return URL
	// Note: Di production, base URL ini harus diambil dari config/env.
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	fileURL := fmt.Sprintf("%s://%s/uploads/%s", scheme, r.Host, newFilename)

	writeJSON(w, http.StatusOK, map[string]string{
		"url":      fileURL,
		"filename": newFilename,
	})
}
