package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

// Todo represents a todo item
type Todo struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
}

// TodoStore handles PostgreSQL storage of todos
type TodoStore struct {
	db *sql.DB
}

// NewTodoStore creates a new todo store with database connection
func NewTodoStore(db *sql.DB) *TodoStore {
	return &TodoStore{db: db}
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

// GetAll returns all todos from database
func (ts *TodoStore) GetAll() ([]Todo, error) {
	query := "SELECT id, title, description, completed, created_at FROM todos ORDER BY created_at DESC"
	rows, err := ts.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var todo Todo
		err := rows.Scan(&todo.ID, &todo.Title, &todo.Description, &todo.Completed, &todo.CreatedAt)
		if err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return todos, nil
}

// Create adds a new todo to database
func (ts *TodoStore) Create(title, description string) (*Todo, error) {
	query := `
		INSERT INTO todos (title, description, completed, created_at) 
		VALUES ($1, $2, $3, $4) 
		RETURNING id, title, description, completed, created_at`

	var todo Todo
	err := ts.db.QueryRow(query, title, description, false, time.Now()).Scan(
		&todo.ID, &todo.Title, &todo.Description, &todo.Completed, &todo.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &todo, nil
}

// Update updates a todo's completion status
func (ts *TodoStore) Update(id int, completed bool) (*Todo, error) {
	query := `
		UPDATE todos 
		SET completed = $1 
		WHERE id = $2 
		RETURNING id, title, description, completed, created_at`

	var todo Todo
	err := ts.db.QueryRow(query, completed, id).Scan(
		&todo.ID, &todo.Title, &todo.Description, &todo.Completed, &todo.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &todo, nil
}

// Delete removes a todo from database
func (ts *TodoStore) Delete(id int) error {
	query := "DELETE FROM todos WHERE id = $1"
	result, err := ts.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("todo with id %d not found", id)
	}

	return nil
}

// Global todo store
var store *TodoStore

// CreateTodoRequest represents the request body for creating a todo
type CreateTodoRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// UpdateTodoRequest represents the request body for updating a todo
type UpdateTodoRequest struct {
	Completed bool `json:"completed"`
}

// getTodosHandler handles GET /todos
func getTodosHandler(w http.ResponseWriter, r *http.Request) {
	todos, err := store.GetAll()
	if err != nil {
		log.Printf("Error fetching todos: %v", err)
		http.Error(w, "Failed to fetch todos", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(todos); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// createTodoHandler handles POST /todos
func createTodoHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	todo, err := store.Create(req.Title, req.Description)
	if err != nil {
		log.Printf("Error creating todo: %v", err)
		http.Error(w, "Failed to create todo", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(todo); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// todosHandler handles all /todos routes
func todosHandler(w http.ResponseWriter, r *http.Request) {
	// Parse ID from path if present
	path := r.URL.Path
	var id int
	var err error

	if len(path) > 7 { // "/todos/" is 7 characters
		idStr := path[7:] // Extract everything after "/todos/"
		id, err = strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid todo ID", http.StatusBadRequest)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		if id == 0 {
			getTodosHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case http.MethodPost:
		if id == 0 {
			createTodoHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
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
func createSampleTodos(store *TodoStore) {
	todos, err := store.GetAll()
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
			_, err := store.Create(todo.title, todo.description)
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

	// Initialize store
	store = NewTodoStore(db)

	// Create sample todos if none exist
	createSampleTodos(store)

	// Set up routes
	http.HandleFunc("/todos", corsMiddleware(todosHandler))
	http.HandleFunc("/todos/", corsMiddleware(todosHandler))

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
