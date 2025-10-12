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

	// --- Resolve environment paths ---
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
	pingpongResp, err := client.Get("http://pingpong-svc:80/pings")
	if err != nil {
		slog.Error("failed to call pingpong service",
			"error", err,
			"service", "pingpong-svc",
			"url", "http://pingpong-svc:80/pings",
		)
		http.Error(w, "failed to reach pingpong service", http.StatusBadGateway)
		return
	}
	defer pingpongResp.Body.Close()

	if pingpongResp.StatusCode != http.StatusOK {
		slog.Warn("unexpected response from pingpong service",
			"status", pingpongResp.StatusCode,
		)
		http.Error(w, fmt.Sprintf("unexpected status from pingpong: %d", pingpongResp.StatusCode), pingpongResp.StatusCode)
		return
	}

	pingpongBody, err := io.ReadAll(pingpongResp.Body)
	if err != nil {
		slog.Error("failed to read pingpong response", "error", err)
		http.Error(w, "failed to read pingpong response", http.StatusInternalServerError)
		return
	}

	// --- Call greeter service ---
	greeterResp, err := client.Get("http://greeter-svc:80")
	if err != nil {
		slog.Error("failed to call greeter service",
			"error", err,
			"service", "greeter-svc",
			"url", "http://greeter-svc:80",
		)
		http.Error(w, "failed to reach greeter service", http.StatusBadGateway)
		return
	}
	defer greeterResp.Body.Close()

	if greeterResp.StatusCode != http.StatusOK {
		slog.Warn("unexpected response from greeter service",
			"status", greeterResp.StatusCode,
		)
		http.Error(w, fmt.Sprintf("unexpected status from greeter: %d", greeterResp.StatusCode), greeterResp.StatusCode)
		return
	}

	greeterBody, err := io.ReadAll(greeterResp.Body)
	if err != nil {
		slog.Error("failed to read greeter response", "error", err)
		http.Error(w, "failed to read greeter response", http.StatusInternalServerError)
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

	// --- Combine output ---
	combined := fmt.Sprintf(
		"Config file content: %s\nMessage (env): %s\nLog file content:\n%s\nPing/Pongs: %s\nGreetings: %s\n",
		string(configData),
		message,
		string(logData),
		string(pingpongBody),
		string(greeterBody),
	)

	_, _ = w.Write([]byte(combined))
}
