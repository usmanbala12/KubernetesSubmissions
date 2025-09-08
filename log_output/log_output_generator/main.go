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

	for {
		currentStatus := fmt.Sprintf(
			"%s : %s\n",
			time.Now().UTC().Format(time.RFC3339Nano),
			randomString,
		)

		// Open file in truncate mode
		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Println("Error opening file:", err)
			return
		}

		_, err = f.WriteString(currentStatus)
		if err != nil {
			fmt.Println("Error writing to file:", err)
			f.Close()
			return
		}
		f.Close()

		fmt.Print("Wrote: ", currentStatus) // optional console log
		time.Sleep(5 * time.Second)
	}
}
