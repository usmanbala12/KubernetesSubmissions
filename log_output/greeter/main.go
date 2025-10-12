package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	// Get version from environment variable, default to "1" if not set
	version := os.Getenv("VERSION")
	if version == "" {
		version = "1"
	}

	// Create handler function
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		response := fmt.Sprintf("hello from version %s", version)
		fmt.Fprint(w, response)
	})

	// Get port from environment variable, default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start server
	log.Printf("Server starting on port %s with version %s", port, version)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
