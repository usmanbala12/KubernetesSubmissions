package main

import (
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
)

var counter uint64 // in-memory counter
var filePath string

func handlePing(w http.ResponseWriter, r *http.Request) {
	// increment atomically to avoid race conditions
	count := atomic.AddUint64(&counter, 1) - 1

	// Write the current count to file (replacing old content)
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to open file: %v", err), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	content := fmt.Sprintf("Ping / Pongs: %d", count)
	_, err = f.WriteString(content)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to write file: %v", err), http.StatusInternalServerError)
		return
	}

	// Respond to client
	fmt.Fprintf(w, "pong %d", count)
}

func main() {
	filePath = os.Getenv("FILE_PATH")
	if filePath == "" {
		filePath = "../pingoutput.txt"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", handlePing)

	fmt.Printf("Server started on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}
