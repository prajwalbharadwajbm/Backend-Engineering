package upload

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type ChunkedUpload struct {
	ID                string       // Upload session ID
	Filename          string       // Original filename
	ReceivedChunks    map[int]bool // Track received chunks
	ConcurrentUploads int          // Maximum parallel uploads
	UploadedSize      int64        // Track total bytes uploaded
	ChunkSize         int64        // Size of each chunk
	TotalChunks       int          // Total number of chunks
	mutex             sync.RWMutex // For thread-safe operations
}

var uploadsMutex sync.RWMutex
var activeUploads = make(map[string]*ChunkedUpload)

func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func HandleChunkedUpload(w http.ResponseWriter, r *http.Request) {
	uploadID := r.URL.Query().Get("uploadId")
	chunkNum := parseInt(r.URL.Query().Get("chunkNum"))

	// Read Lock: Multiple readers OK
	uploadsMutex.RLock()
	upload, exists := activeUploads[uploadID]
	uploadsMutex.RUnlock()

	if !exists {
		http.Error(w, "Upload session not found", http.StatusNotFound)
		return
	}

	// Write Lock: Only one writer at a time
	upload.mutex.Lock()
	if upload.ReceivedChunks[chunkNum] {
		upload.mutex.Unlock()
		w.WriteHeader(http.StatusOK) // Already have this chunk
		return
	}
	upload.mutex.Unlock()

	if err := processChunk(upload, chunkNum, r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	upload.mutex.Lock()
	upload.ReceivedChunks[chunkNum] = true                         // Mark chunk as received
	upload.UploadedSize += upload.ChunkSize                        // Update total bytes
	isComplete := len(upload.ReceivedChunks) == upload.TotalChunks // Check if done
	upload.mutex.Unlock()

	if isComplete {
		go mergeChunks(upload) //Background Merge
	}

	w.WriteHeader(http.StatusOK)
}

func processChunk(upload *ChunkedUpload, chunkNum int, r *http.Request) error {
	file, _, err := r.FormFile("chunk")
	if err != nil {
		return err
	}
	defer file.Close()

	chunkPath := filepath.Join("uploads", "temp", upload.ID, fmt.Sprintf("chunk_%d", chunkNum))
	chunk, err := os.Create(chunkPath)
	if err != nil {
		return err
	}
	defer chunk.Close()

	_, err = io.Copy(chunk, file)
	return err
}

func mergeChunks(upload *ChunkedUpload) error {
	// Create final directory if it doesn't exist
	os.MkdirAll(filepath.Join("uploads", "final"), 0755)

	// Use original filename for the final file
	finalPath := filepath.Join("uploads", "final", upload.Filename)
	finalFile, err := os.Create(finalPath)
	if err != nil {
		return err
	}
	defer finalFile.Close()

	// Merge chunks in order
	for i := 0; i < upload.TotalChunks; i++ {
		chunkPath := filepath.Join("uploads", "temp", upload.ID, fmt.Sprintf("chunk_%d", i))
		chunk, err := os.Open(chunkPath)
		if err != nil {
			return fmt.Errorf("failed to open chunk %d: %v", i, err)
		}

		_, err = io.Copy(finalFile, chunk)
		chunk.Close()
		if err != nil {
			return fmt.Errorf("failed to copy chunk %d: %v", i, err)
		}

		os.Remove(chunkPath)
	}

	os.RemoveAll(filepath.Join("uploads", "temp", upload.ID))

	// Remove from active uploads
	uploadsMutex.Lock()
	delete(activeUploads, upload.ID)
	uploadsMutex.Unlock()

	return nil
}

func HandleInitiateUpload(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Filename    string `json:"filename"`
		TotalSize   int64  `json:"totalSize"`
		ChunkSize   int64  `json:"chunkSize"`
		TotalChunks int    `json:"totalChunks"`
		Replace     bool   `json:"replace"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if file already exists
	finalPath := filepath.Join("uploads", "final", req.Filename)
	if _, err := os.Stat(finalPath); err == nil && !req.Replace {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "exists",
			"message":  "File already exists",
			"filename": req.Filename,
		})
		return
	}

	uploadID := fmt.Sprintf("%d", time.Now().UnixNano())
	upload := &ChunkedUpload{
		ID:             uploadID,
		Filename:       req.Filename,
		ReceivedChunks: make(map[int]bool),
		ChunkSize:      req.ChunkSize,
		TotalChunks:    req.TotalChunks,
	}

	uploadsMutex.Lock()
	activeUploads[uploadID] = upload
	uploadsMutex.Unlock()

	os.MkdirAll(filepath.Join("uploads", "temp", uploadID), 0755)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"uploadId": uploadID,
		"status":   "initiated",
	})
}

func HandleUploadStatus(w http.ResponseWriter, r *http.Request) {
	uploadID := r.URL.Query().Get("uploadId")

	uploadsMutex.RLock()
	upload, exists := activeUploads[uploadID]
	uploadsMutex.RUnlock()

	if !exists {
		http.Error(w, "Upload not found", http.StatusNotFound)
		return
	}

	upload.mutex.RLock()
	defer upload.mutex.RUnlock()

	// Check if file exists in final directory
	finalPath := filepath.Join("uploads", "final", upload.Filename)
	if _, err := os.Stat(finalPath); err == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uploadId":   uploadID,
			"status":     "completed",
			"filePath":   finalPath,
			"isComplete": true,
		})
		return
	}

	status := map[string]interface{}{
		"uploadId":       upload.ID,
		"receivedChunks": len(upload.ReceivedChunks),
		"totalChunks":    upload.TotalChunks,
		"isComplete":     len(upload.ReceivedChunks) == upload.TotalChunks,
		"tempPath":       filepath.Join("uploads", "temp", upload.ID),
		"status":         "in_progress",
	}

	json.NewEncoder(w).Encode(status)
}
