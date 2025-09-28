package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

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
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/status", statusHandler)
	fmt.Printf("Server started on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Log Output Service - OK\n")
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	logPath := os.Getenv("LOG_PATH")
	if logPath == "" {
		logPath = "../logoutput.txt"
	}

	configPath := os.Getenv("CONFIG_FILE_PATH")
	if configPath == "" {
		configPath = "../information.txt"
	}

	message := os.Getenv("MESSAGE")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// --- Call pingpong service ---
	resp, err := client.Get("http://pingpong-svc:80/pings")
	if err != nil {
		slog.Error("failed to call pingpong service",
			"error", err,
			"service", "pingpong-svc",
			"url", "http://pingpong-svc:80/pings",
		)
		http.Error(w, "failed to reach pingpong service", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("unexpected response from pingpong service",
			"status", resp.StatusCode,
		)
		http.Error(w, fmt.Sprintf("unexpected status: %d", resp.StatusCode), resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("failed to read pingpong response", "error", err)
		http.Error(w, "failed to read response", http.StatusInternalServerError)
		return
	}

	// --- Read log file ---
	logData, err := os.ReadFile(logPath)
	if err != nil {
		slog.Error("failed to read log file", "path", logPath, "error", err)
		http.Error(w, "failed to read log file", http.StatusInternalServerError)
		return
	}

	// --- Read config file ---
	configData, err := os.ReadFile(configPath)
	if err != nil {
		slog.Error("failed to read config file", "path", configPath, "error", err)
		http.Error(w, "failed to read config file", http.StatusInternalServerError)
		return
	}

	combined := fmt.Sprintf(
		"file content: %s\n env variable: MESSAGE=%s\n %s\nPing / Pongs: %s",
		string(configData), message, string(logData), string(body),
	)

	_, _ = w.Write([]byte(combined))
}
