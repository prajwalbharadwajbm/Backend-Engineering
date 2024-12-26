package main

import (
	"fileupload/upload"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	setupDirectories()

	// Serve static files
	http.Handle("/", http.FileServer(http.Dir("static")))

	// Single file upload route
	http.HandleFunc("/api/v1/upload", upload.HandleSingleUpload)

	// Chunked upload routes
	http.HandleFunc("/api/v1/upload/init", upload.HandleInitiateUpload)
	http.HandleFunc("/api/v1/upload/chunk", upload.HandleChunkedUpload)
	http.HandleFunc("/api/v1/upload/status", upload.HandleUploadStatus)

	// Add SSH upload endpoint
	http.HandleFunc("/api/v1/ssh/upload", upload.HandleSSHUpload)
	http.HandleFunc("/api/v1/ssh/test", upload.HandleSSHTest)

	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println("- Single file upload: POST /api/v1/upload")
	fmt.Println("- Chunked upload: POST /api/v1/upload/init")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func setupDirectories() {
	dirs := []string{
		"uploads",
		"uploads/temp",
		"uploads/final",
		"static",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatal(err)
		}
	}
}
