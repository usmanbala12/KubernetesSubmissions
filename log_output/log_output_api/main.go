package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Generate a random UUID on startup
	randomString := uuid.New().String()
	fmt.Printf("Application started. Random string: %s\n", randomString)

	// Expose an HTTP endpoint for current status
	http.HandleFunc("/status", statusHandler)
	fmt.Printf("Server started on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	logPath := os.Getenv("LOG_PATH")
	if logPath == "" {
		logPath = "../logoutput.txt"
	}

	pingPath := os.Getenv("PING_PATH")
	if pingPath == "" {
		pingPath = "../pingoutput.txt"
	}

	// Read log file
	logData, err := os.ReadFile(logPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read log file: %v", err), http.StatusInternalServerError)
		return
	}

	// Read ping file
	pingData, err := os.ReadFile(pingPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read ping file: %v", err), http.StatusInternalServerError)
		return
	}

	// Format output: first log line, then ping info
	combined := fmt.Sprintf("%s\n%s", string(logData), string(pingData))

	// Write directly as plain text
	w.Write([]byte(combined))
}
