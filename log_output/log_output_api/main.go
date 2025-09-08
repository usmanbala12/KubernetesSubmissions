package main

import (
	"encoding/json"
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
	w.Header().Set("Content-Type", "application/json")

	filePath := os.Getenv("FILE_PATH")
	if filePath == "" {
		filePath = "../logoutput.txt"
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to read log file: %v", err),
		})
		return
	}

	// Respond with JSON containing the log content
	json.NewEncoder(w).Encode(string(data))
}
