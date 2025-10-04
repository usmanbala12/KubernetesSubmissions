package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"todo-backend/internal/data"
)

type CreateTodoRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// UpdateTodoRequest represents the request body for updating a todo
type UpdateTodoRequest struct {
	Completed bool `json:"completed"`
}

type TodoMessage struct {
	Action      string `json:"action"`
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

// getTodosHandler handles GET /todos
func (app *application) getTodosHandler(w http.ResponseWriter, r *http.Request) {
	todos, err := app.store.GetAll()
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(todos); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) createTodoHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	validationMessage := make(map[string]string)

	// Validate required fields
	if req.Title == "" {
		validationMessage["title"] = "title is required"
	}

	if req.Description == "" {
		validationMessage["description"] = "Description is required"
	}

	if len(req.Description) > 140 {
		validationMessage["description"] = "Description cannot exceed 140 characters"
	}

	if len(validationMessage) > 0 {
		app.failedValidationResponse(w, r, validationMessage)
		return
	}

	todo, err := app.store.Create(req.Title, req.Description)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Publish todo creation event to NATS
	if err := app.publishTodoEvent("created", todo); err != nil {
		// Log the error but don't fail the request
		// You might want to use a proper logger here
		fmt.Printf("Warning: failed to publish todo event: %v\n", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(todo); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) updateTodoHandler(w http.ResponseWriter, r *http.Request, id int) {
	var req UpdateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	todo, err := app.store.Update(id, req.Completed)
	if err != nil {
		if err.Error() == fmt.Sprintf("todo with id %d not found", id) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	// Publish todo update event to NATS
	if err := app.publishTodoEvent("updated", todo); err != nil {
		// Log the error but don't fail the request
		fmt.Printf("Warning: failed to publish todo event: %v\n", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(todo); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) publishTodoEvent(action string, todo *data.Todo) error {
	msg := TodoMessage{
		Action:      action,
		ID:          todo.ID,
		Title:       todo.Title,
		Description: todo.Description,
		Completed:   todo.Completed,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal todo message: %w", err)
	}

	// Publish to NATS JetStream with acknowledgment
	// JetStream ensures the message is persisted before returning
	pubAck, err := app.js.Publish("todos.events", data)
	if err != nil {
		return fmt.Errorf("failed to publish to NATS JetStream: %w", err)
	}

	// Log successful publish with stream sequence number
	fmt.Printf("Published todo event to JetStream: stream=%s, seq=%d\n",
		pubAck.Stream, pubAck.Sequence)

	return nil
}

// todosHandler handles all /todos routes
func (app *application) todosHandler(w http.ResponseWriter, r *http.Request) {
	// Parse ID from path if present
	path := r.URL.Path
	var id int
	var err error

	if len(path) > 7 { // "/todos/" is 7 characters
		idStr := path[7:] // Extract everything after "/todos/"
		id, err = strconv.Atoi(idStr)
		if err != nil {
			app.badRequestResponse(w, r, err)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		if id == 0 {
			app.getTodosHandler(w, r)
		} else {
			app.methodNotAllowedResponse(w, r)
		}
	case http.MethodPost:
		if id == 0 {
			app.createTodoHandler(w, r)
		} else {
			app.methodNotAllowedResponse(w, r)
		}
	case http.MethodPatch:
		if id != 0 {
			app.updateTodoHandler(w, r, id)
		} else {
			app.methodNotAllowedResponse(w, r)
		}
	default:
		app.methodNotAllowedResponse(w, r)
	}
}
