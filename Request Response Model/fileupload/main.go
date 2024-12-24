package main

import (
	"fileupload/upload"
	"fmt"
	"log"
	"net/http"
)

func main() {
	setupServer()

	http.HandleFunc("/api/v1/upload", upload.HandleSingleUpload)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func setupServer() {
	fmt.Println("Initializing File Upload Server...")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Authorization")

		if r.Method == "OPTIONS" {
			return
		}

		// Handle 404 for undefined routes
		http.NotFound(w, r)
	})
}
