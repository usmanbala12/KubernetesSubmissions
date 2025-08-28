package main

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

func main() {
	// Generate a random UUID on startup
	randomString := uuid.New().String()

	fmt.Printf("Application started. Random string: %s\n", randomString)

	// Create a ticker that fires every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Output the first timestamp immediately
	fmt.Printf("%s: %s\n", time.Now().UTC().Format(time.RFC3339Nano), randomString)

	// Continue outputting every 5 seconds
	for range ticker.C {
		timestamp := time.Now().UTC().Format(time.RFC3339Nano)
		fmt.Printf("%s: %s\n", timestamp, randomString)
	}
}
