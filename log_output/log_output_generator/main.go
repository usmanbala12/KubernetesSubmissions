package main

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
)

func main() {
	filePath := os.Getenv("FILE_PATH")
	if filePath == "" {
		filePath = "../logoutput.txt"
	}

	randomString := uuid.New().String()
	fmt.Printf("Application started. Random string: %s\n", randomString)

	// Open file once (append mode)
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer f.Close()

	for {
		currentStatus := fmt.Sprintf(
			"%s : %s\n",
			time.Now().UTC().Format(time.RFC3339Nano),
			randomString,
		)

		_, err := f.WriteString(currentStatus)
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}

		fmt.Print("Wrote: ", currentStatus) // optional console log
		time.Sleep(5 * time.Second)
	}
}
