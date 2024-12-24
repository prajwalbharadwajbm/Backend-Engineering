package upload

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func HandleSingleUpload(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")
	// file   - contains the actual file data (like a stream of bytes)
	// header - contains metadata (filename, size, etc.)
	// err    - will contain any error that occurred

	if err != nil {
		http.Error(w, "Error getting file", http.StatusBadRequest)
	}
	defer file.Close()

	uploadDir := "uploads"
	os.MkdirAll(uploadDir, os.ModePerm)

	dst, err := os.Create(filepath.Join(uploadDir, header.Filename))
	if err != nil {
		http.Error(w, "Error Creating file", http.StatusInternalServerError)
	}
	defer dst.Close()

	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(dst, hash), file)
	if err != nil {
		http.Error(w, "Error Saving File", http.StatusInternalServerError)
		return
	}

	if written != header.Size {
		http.Error(w, "File size mismatch, upload may be corrupted", http.StatusInternalServerError)
		os.Remove(filepath.Join(uploadDir, header.Filename))
		return
	}

	checksum := hex.EncodeToString(hash.Sum(nil))
	/*1. Original content: "Hello"

	  2. SHA-256 hash (in bytes):
	     [185][248][219][50]...(more bytes)

	  3. Converted to hex string:
	     "185f8db32271fe25f561a6fc938b2e264306ec304eda518007d1764826381969"*/

	response := fmt.Sprintf("File Uploaded Successfully. Size: %d bytes, Checksum: %s", written, checksum)
	w.Write([]byte(response))
}
