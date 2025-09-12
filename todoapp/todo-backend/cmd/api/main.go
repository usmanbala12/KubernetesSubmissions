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
}

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
	}

	// Create sample todos if none exist
	app.createSampleTodos()

	// Set up routes
	http.HandleFunc("/todos", corsMiddleware(app.todosHandler))
	http.HandleFunc("/todos/", corsMiddleware(app.todosHandler))

	// Health check endpoint
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

	port := os.Getenv("PORT")

	fmt.Printf("Todo backend service starting on port %s\n", port)
	fmt.Printf("Endpoints:\n")
	fmt.Printf("  GET    /todos     - Fetch all todos\n")
	fmt.Printf("  POST   /todos     - Create a new todo\n")
	fmt.Printf("  GET    /health    - Health check\n")

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
