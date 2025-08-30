package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Status holds the current timestamp and random string
type Status struct {
	Timestamp    string `json:"timestamp"`
	RandomString string `json:"random_string"`
}

var (
	currentStatus Status
	mu            sync.RWMutex // protects currentStatus
)

func main() {

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Generate a random UUID on startup
	randomString := uuid.New().String()
	fmt.Printf("Application started. Random string: %s\n", randomString)

	// Set the first status immediately
	updateStatus(randomString)

	// Create a ticker that fires every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Start the logging goroutine
	go func() {
		for range ticker.C {
			updateStatus(randomString)
			mu.RLock()
			fmt.Printf("%s: %s\n", currentStatus.Timestamp, currentStatus.RandomString)
			mu.RUnlock()
		}
	}()

	// Expose an HTTP endpoint for current status
	http.HandleFunc("/status", statusHandler)
	fmt.Printf("Server started on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}

func updateStatus(randomString string) {
	mu.Lock()
	defer mu.Unlock()
	currentStatus = Status{
		Timestamp:    time.Now().UTC().Format(time.RFC3339Nano),
		RandomString: randomString,
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(currentStatus)
}
