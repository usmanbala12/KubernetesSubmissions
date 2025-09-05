package main

import (
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
)

var counter uint64 // in-memory counter

func handlePing(w http.ResponseWriter, r *http.Request) {
	// increment atomically to avoid race conditions
	count := atomic.AddUint64(&counter, 1) - 1
	fmt.Fprintf(w, "pong %d", count)
}

func main() {

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
