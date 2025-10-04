package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
)

// This version uses NATS JetStream for guaranteed message delivery
// and durable consumers for exactly-once processing semantics

type TodoMessage struct {
	Action      string `json:"action"`
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

type Config struct {
	NatsURL       string
	TelegramToken string
	TelegramChat  string
	Subject       string
	HealthPort    string
	StreamName    string
	ConsumerName  string
}

type HealthChecker struct {
	mu            sync.RWMutex
	natsConnected bool
	ready         bool
	lastNatsCheck time.Time
	lastMessage   time.Time
}

func (h *HealthChecker) SetNatsConnected(connected bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.natsConnected = connected
	h.lastNatsCheck = time.Now()
}

func (h *HealthChecker) IsNatsConnected() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.natsConnected
}

func (h *HealthChecker) SetReady(ready bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ready = ready
}

func (h *HealthChecker) IsReady() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.ready
}

func (h *HealthChecker) UpdateLastMessage() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastMessage = time.Now()
}

func (h *HealthChecker) GetLastMessageTime() time.Time {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastMessage
}

func (h *HealthChecker) GetLastNatsCheckTime() time.Time {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastNatsCheck
}

func main() {
	config := Config{
		NatsURL:       getEnv("NATS_URL", "nats://localhost:4222"),
		TelegramToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChat:  getEnv("TELEGRAM_CHAT_ID", ""),
		Subject:       getEnv("NATS_SUBJECT", "todos.events"),
		HealthPort:    getEnv("PORT", "4000"),
		StreamName:    getEnv("STREAM_NAME", "TODOS"),
		ConsumerName:  getEnv("CONSUMER_NAME", "broadcaster"),
	}

	if config.TelegramToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is required")
	}

	if config.TelegramChat == "" {
		log.Fatal("TELEGRAM_CHAT_ID environment variable is required")
	}

	healthChecker := &HealthChecker{}

	// Start health check server
	healthServer := startHealthServer(config.HealthPort, healthChecker)

	// Create Telegram client
	telegram := NewTelegramClient(config.TelegramToken, config.TelegramChat)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect and setup JetStream
	var nc *nats.Conn
	var js nats.JetStreamContext
	var sub *nats.Subscription
	var err error

	// Initial connection
	nc, js, sub, err = connectAndSubscribeJetStream(config, telegram, healthChecker)
	if err != nil {
		log.Printf("Initial connection failed: %v. Will retry...", err)
	}

	// Monitor connection
	go monitorConnectionJetStream(ctx, &nc, &js, &sub, config, telegram, healthChecker)

	log.Println("Broadcaster service is running with JetStream. Press Ctrl+C to exit.")

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down broadcaster service...")
	cancel()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Health server shutdown error: %v", err)
	}

	if nc != nil {
		if sub != nil {
			sub.Drain()
		}
		nc.Drain()
	}

	log.Println("Broadcaster service stopped")
}

func connectAndSubscribeJetStream(config Config, telegram *TelegramClient, healthChecker *HealthChecker) (*nats.Conn, nats.JetStreamContext, *nats.Subscription, error) {
	// Connect to NATS
	nc, err := nats.Connect(
		config.NatsURL,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				log.Printf("NATS disconnected: %v", err)
			}
			healthChecker.SetNatsConnected(false)
			healthChecker.SetReady(false)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("NATS reconnected to %s", nc.ConnectedUrl())
			healthChecker.SetNatsConnected(true)
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Println("NATS connection closed")
			healthChecker.SetNatsConnected(false)
			healthChecker.SetReady(false)
		}),
	)
	if err != nil {
		healthChecker.SetNatsConnected(false)
		healthChecker.SetReady(false)
		return nil, nil, nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	log.Printf("Connected to NATS at %s", config.NatsURL)
	healthChecker.SetNatsConnected(true)

	// Create JetStream context
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		healthChecker.SetNatsConnected(false)
		healthChecker.SetReady(false)
		return nil, nil, nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Ensure stream exists
	streamConfig := &nats.StreamConfig{
		Name:     config.StreamName,
		Subjects: []string{config.Subject},
		Storage:  nats.FileStorage,
		MaxAge:   24 * time.Hour,
		Replicas: 1,
	}

	stream, err := js.StreamInfo(config.StreamName)
	if err != nil {
		_, err = js.AddStream(streamConfig)
		if err != nil {
			nc.Close()
			return nil, nil, nil, fmt.Errorf("failed to create stream: %w", err)
		}
		log.Printf("Created JetStream stream: %s", config.StreamName)
	} else {
		log.Printf("Using existing JetStream stream: %s (messages: %d)", config.StreamName, stream.State.Msgs)
	}

	// Check if consumer exists and delete if incompatible
	consumerInfo, err := js.ConsumerInfo(config.StreamName, config.ConsumerName)
	if err == nil {
		// Consumer exists - check if it's pull-based or missing deliver group
		if consumerInfo.Config.DeliverSubject == "" || consumerInfo.Config.DeliverGroup == "" {
			log.Printf("Deleting incompatible consumer: %s", config.ConsumerName)
			if err := js.DeleteConsumer(config.StreamName, config.ConsumerName); err != nil {
				nc.Close()
				return nil, nil, nil, fmt.Errorf("failed to delete consumer: %w", err)
			}
		}
	}

	// Create PUSH-based durable consumer WITH deliver group
	consumerConfig := &nats.ConsumerConfig{
		Durable:        config.ConsumerName,
		DeliverPolicy:  nats.DeliverAllPolicy,
		AckPolicy:      nats.AckExplicitPolicy,
		MaxDeliver:     3,
		AckWait:        30 * time.Second,
		DeliverSubject: nats.NewInbox(),
		DeliverGroup:   "broadcaster-workers",
	}

	_, err = js.AddConsumer(config.StreamName, consumerConfig)
	if err != nil && !errors.Is(err, nats.ErrConsumerNameAlreadyInUse) {
		nc.Close()
		return nil, nil, nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	// Subscribe using QueueSubscribe (Push mode with load balancing)
	sub, err := js.QueueSubscribe(
		config.Subject,
		"broadcaster-workers", // Must match DeliverGroup
		func(msg *nats.Msg) {
			healthChecker.UpdateLastMessage()

			var todoMsg TodoMessage
			if err := json.Unmarshal(msg.Data, &todoMsg); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				msg.Nak()
				return
			}

			log.Printf("Processing todo event: %s - ID: %d", todoMsg.Action, todoMsg.ID)
			message := formatTodoMessage(todoMsg)

			if err := telegram.SendMessage(message); err != nil {
				log.Printf("Error sending to Telegram: %v", err)
				msg.Nak()
				return
			}

			log.Printf("Successfully sent message to Telegram")
			msg.Ack()
		},
		nats.Durable(config.ConsumerName),
		nats.ManualAck(),
		nats.Bind(config.StreamName, config.ConsumerName),
	)
	if err != nil {
		nc.Close()
		return nil, nil, nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	log.Printf("Subscribed to subject: %s with durable PUSH consumer: %s", config.Subject, config.ConsumerName)
	healthChecker.SetReady(true)

	return nc, js, sub, nil
}

func monitorConnectionJetStream(ctx context.Context, nc **nats.Conn, js *nats.JetStreamContext, sub **nats.Subscription, config Config, telegram *TelegramClient, healthChecker *HealthChecker) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if *nc == nil || !(*nc).IsConnected() {
				log.Println("NATS connection lost. Attempting to reconnect...")
				healthChecker.SetNatsConnected(false)
				healthChecker.SetReady(false)

				if *nc != nil {
					if *sub != nil {
						(*sub).Drain()
					}
					(*nc).Drain()
				}

				newNc, newJs, newSub, err := connectAndSubscribeJetStream(config, telegram, healthChecker)
				if err != nil {
					log.Printf("Reconnection failed: %v", err)
					continue
				}

				*nc = newNc
				*js = newJs
				*sub = newSub
				log.Println("Successfully reconnected to NATS with JetStream")
			} else {
				healthChecker.SetNatsConnected(true)
			}
		}
	}
}

func startHealthServer(port string, healthChecker *HealthChecker) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/liveness", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "alive",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("/readiness", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if !healthChecker.IsReady() || !healthChecker.IsNatsConnected() {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":         "not ready",
				"nats_connected": healthChecker.IsNatsConnected(),
				"ready":          healthChecker.IsReady(),
				"time":           time.Now().Format(time.RFC3339),
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":         "ready",
			"nats_connected": true,
			"time":           time.Now().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		lastMessage := healthChecker.GetLastMessageTime()
		lastNatsCheck := healthChecker.GetLastNatsCheckTime()

		status := map[string]interface{}{
			"status":                "healthy",
			"nats_connected":        healthChecker.IsNatsConnected(),
			"ready":                 healthChecker.IsReady(),
			"last_nats_check":       lastNatsCheck.Format(time.RFC3339),
			"last_message_received": nil,
			"time":                  time.Now().Format(time.RFC3339),
		}

		if !lastMessage.IsZero() {
			status["last_message_received"] = lastMessage.Format(time.RFC3339)
			status["seconds_since_last_message"] = time.Since(lastMessage).Seconds()
		}

		if !healthChecker.IsReady() || !healthChecker.IsNatsConnected() {
			status["status"] = "unhealthy"
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		json.NewEncoder(w).Encode(status)
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		log.Printf("Health check server started on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Health server error: %v", err)
		}
	}()

	return server
}

func formatTodoMessage(todo TodoMessage) string {
	var status string
	switch todo.Action {
	case "created":
		status = "📝 *New Todo Created*"
	case "updated":
		if todo.Completed {
			status = "✅ *Todo Completed*"
		} else {
			status = "🔄 *Todo Updated*"
		}
	default:
		status = "📋 *Todo Event*"
	}

	message := fmt.Sprintf("%s\n\n"+
		"*Title:* %s\n"+
		"*Description:* %s\n"+
		"*Status:* %s\n"+
		"*ID:* %d",
		status,
		escapeMarkdown(todo.Title),
		escapeMarkdown(todo.Description),
		getStatusEmoji(todo.Completed),
		todo.ID,
	)

	return message
}

func getStatusEmoji(completed bool) string {
	if completed {
		return "Completed ✅"
	}
	return "Pending ⏳"
}

func escapeMarkdown(text string) string {
	replacer := map[rune]string{
		'_': "\\_", '*': "\\*", '[': "\\[", ']': "\\]",
		'(': "\\(", ')': "\\)", '~': "\\~", '`': "\\`",
		'>': "\\>", '#': "\\#", '+': "\\+", '-': "\\-",
		'=': "\\=", '|': "\\|", '{': "\\{", '}': "\\}",
		'.': "\\.", '!': "\\!",
	}

	result := ""
	for _, char := range text {
		if escaped, ok := replacer[char]; ok {
			result += escaped
		} else {
			result += string(char)
		}
	}
	return result
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
