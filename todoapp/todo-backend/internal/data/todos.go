package data

import (
	"database/sql"
	"fmt"
	"time"
)

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
func NewTodoStore(db *sql.DB) TodoStore {
	return TodoStore{db: db}
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
