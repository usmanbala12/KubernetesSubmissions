package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"todo-backend/internal/data"
	"todo-backend/internal/jsonlog"

	_ "github.com/lib/pq"
)

type application struct {
	logger *jsonlog.Logger
	store  data.TodoStore
	wg     sync.WaitGroup
	db     *sql.DB
}

// Trigger Github actions GKE Deployment IV
// corsMiddleware adds CORS headers
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}
func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Todo App backend - OK\n")
}

// createSampleTodos creates some sample todos if the table is empty
func (app *application) createSampleTodos() {
	todos, err := app.store.GetAll()
	if err != nil {
		log.Printf("Error checking for existing todos: %v", err)
		return
	}
	if len(todos) == 0 {
		sampleTodos := []struct {
			title       string
			description string
		}{
			{"Buy groceries", "Milk, bread, and eggs"},
			{"Walk the dog", "Take Rex for a 30-minute walk"},
			{"Finish project", "Complete the todo backend service"},
		}
		for _, todo := range sampleTodos {
			_, err := app.store.Create(todo.title, todo.description)
			if err != nil {
				log.Printf("Error creating sample todo: %v", err)
			}
		}
	}
}

// Readiness probe - checks if the app is ready to serve traffic
func (app *application) readinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check database connection
	if app.db == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "not ready",
			"reason": "database not initialized",
		})
		return
	}

	// Verify database is reachable
	if err := app.db.Ping(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "not ready",
			"reason": "database connection failed",
			"error":  err.Error(),
		})
		return
	}

	// Verify we can query the database
	var count int
	err := app.db.QueryRow("SELECT COUNT(*) FROM todos").Scan(&count)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "not ready",
			"reason": "database query failed",
			"error":  err.Error(),
		})
		return
	}

	// All checks passed
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "ready",
		"todo_count": count,
	})
}

// Liveness probe - checks if the app is alive
func (app *application) livenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "alive",
	})
}

func main() {
	// Initialize database
	db, err := InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)
	app := &application{
		logger: logger,
		store:  data.NewTodoStore(db),
		db:     db,
	}
	// Create sample todos if none exist
	app.createSampleTodos()
	// Set up routes
	http.HandleFunc("/", corsMiddleware(rootHandler))
	http.HandleFunc("/todos", corsMiddleware(app.todosHandler))
	http.HandleFunc("/todos/", corsMiddleware(app.todosHandler))

	// Health check endpoint (legacy)
	http.HandleFunc("/health", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// Test database connection
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "unhealthy",
				"error":  "database connection failed",
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}))

	http.HandleFunc("/readiness", app.readinessHandler)
	http.HandleFunc("/liveness", app.livenessHandler)

	port := os.Getenv("PORT")
	fmt.Printf("Todo backend service starting on port %s\n", port)
	fmt.Printf("Endpoints:\n")
	fmt.Printf("  GET    /todos       - Fetch all todos\n")
	fmt.Printf("  POST   /todos       - Create a new todo\n")
	fmt.Printf("  PATCH  /todos/{id}  - Update todo completion status\n")
	fmt.Printf("  GET    /health      - Health check\n")
	fmt.Printf("  GET    /readiness   - Readiness probe\n")
	fmt.Printf("  GET    /liveness    - Liveness probe\n")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// InitDB initializes the database connection and creates tables
func InitDB() (*sql.DB, error) {
	dbURL := os.Getenv("DATABASE_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}
	// Create table if it doesn't exist
	createTableSQL := `
    CREATE TABLE IF NOT EXISTS todos (
        id SERIAL PRIMARY KEY,
        title VARCHAR(255) NOT NULL,
        description TEXT,
        completed BOOLEAN DEFAULT FALSE,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %v", err)
	}
	return db, nil
}
