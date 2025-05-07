package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

func main() {
	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, "/SSE", HandleSSE)

	// Setup a standard file server for static files
	fileServer := http.FileServer(http.Dir("."))
	router.NotFound = fileServer

	server := &http.Server{
		Addr:    "localhost:8080",
		Handler: router,
	}

	fmt.Println("Server started at http://localhost:8080")
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func HandleSSE(w http.ResponseWriter, r *http.Request) {
	// set header to type text/event-stream which tells browser that it still is request but
	// never ending response, you can to parse mini events and wait until end response.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	// Not needed for modern browsers tho.
	w.Header().Set("Connection", "keep-alive")

	// Create flusher instance
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	for i := 0; i < 10; i++ {
		// For using SSE we must make sure the response events are sent in Syntax as:
		// data: [content] \n\n
		// As Browser will parse in the same format
		b := []byte(fmt.Sprintf("data: %d \n\n", i))
		// the buffered data may not reach the client until the response completes.
		// Hence we need to flush it as soon as we write it.
		w.Write(b)
		// Immediately flushes any buffered response to the client.
		flusher.Flush()
		time.Sleep(1 * time.Second)
	}
}
