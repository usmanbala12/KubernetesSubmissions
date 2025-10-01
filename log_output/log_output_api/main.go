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
	http.HandleFunc("/readiness", readinessHandler)
	fmt.Printf("Server started on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}
func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Log Output Service - OK\n")
}

// Readiness probe endpoint
func readinessHandler(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Check if we can reach the pingpong service
	resp, err := client.Get("http://pingpong-svc:80/pings")
	if err != nil {
		slog.Warn("readiness check failed: cannot reach pingpong service", "error", err)
		http.Error(w, fmt.Sprintf("Pingpong service not reachable: %v", err), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// Check if the response is successful
	if resp.StatusCode != http.StatusOK {
		slog.Warn("readiness check failed: pingpong service returned non-OK status", "status", resp.StatusCode)
		http.Error(w, fmt.Sprintf("Pingpong service not ready: status %d", resp.StatusCode), http.StatusServiceUnavailable)
		return
	}

	// Verify we can read the response
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		slog.Warn("readiness check failed: cannot read pingpong response", "error", err)
		http.Error(w, "Cannot read pingpong response", http.StatusServiceUnavailable)
		return
	}

	// All checks passed
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ready")
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
