package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// Todo represents a todo item
type Todo struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
}

// TodoStore handles in-memory storage of todos
type TodoStore struct {
	mu     sync.RWMutex
	todos  map[int]Todo
	nextID int
}

// NewTodoStore creates a new todo store
func NewTodoStore() *TodoStore {
	return &TodoStore{
		todos:  make(map[int]Todo),
		nextID: 1,
	}
}

// GetAll returns all todos
func (ts *TodoStore) GetAll() []Todo {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	todos := make([]Todo, 0, len(ts.todos))
	for _, todo := range ts.todos {
		todos = append(todos, todo)
	}
	return todos
}

// Create adds a new todo
func (ts *TodoStore) Create(title, description string) Todo {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	todo := Todo{
		ID:          ts.nextID,
		Title:       title,
		Description: description,
		Completed:   false,
		CreatedAt:   time.Now(),
	}

	ts.todos[ts.nextID] = todo
	ts.nextID++

	return todo
}

// Global todo store
var store = NewTodoStore()

// CreateTodoRequest represents the request body for creating a todo
type CreateTodoRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// getTodosHandler handles GET /todos
func getTodosHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	todos := store.GetAll()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(todos); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// createTodoHandler handles POST /todos
func createTodoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

	todo := store.Create(req.Title, req.Description)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(todo); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func main() {
	// Create some sample todos
	store.Create("Buy groceries", "Milk, bread, and eggs")
	store.Create("Walk the dog", "Take Rex for a 30-minute walk")
	store.Create("Finish project", "Complete the todo backend service")

	// Set up routes
	http.HandleFunc("/todos", func(w http.ResponseWriter, r *http.Request) {
		corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				getTodosHandler(w, r)
			case http.MethodPost:
				createTodoHandler(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})(w, r)
	})

	// Health check endpoint
	http.HandleFunc("/health", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}))

	port := os.Getenv("PORT")

	fmt.Printf("Todo backend service starting on port %s\n", port)
	fmt.Printf("Endpoints:\n")
	fmt.Printf("  GET  /todos  - Fetch all todos\n")
	fmt.Printf("  POST /todos  - Create a new todo\n")
	fmt.Printf("  GET  /health - Health check\n")

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
