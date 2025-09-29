package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

var (
	imagePath      string       // path to cached image
	imageTimestamp time.Time    // last time image was updated
	mu             sync.RWMutex // protect access to image metadata with read-write mutex
	serveOldOnce   bool         // allow serving old image one more time
	staticPath     string       // static files directory
)

//Trigger Github actions GKE Deployment

func main() {
	// Get port from environment variable, default to 8080
	port := os.Getenv("PORT")

	staticPath = os.Getenv("STATIC_PATH")

	// Ensure static directory exists
	err := os.MkdirAll(staticPath, 0755)
	if err != nil {
		log.Fatalf("failed to create static dir: %v", err)
	}

	// Fetch initial image at startup
	if err := fetchNewImage(); err != nil {
		log.Printf("Warning: failed to fetch initial image: %v", err)
		// Don't exit - the server can still run without an initial image
	}

	mux := http.NewServeMux()

	// Static file handler
	fs := http.FileServer(http.Dir(staticPath))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/ready", handleReady)
	mux.HandleFunc("/image", handleImage)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		fmt.Printf("Server started on port %s\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Shutting down server...")

	// Give outstanding requests a 30 second deadline to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	fmt.Println("Server exited")
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	html := `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Todo App API</title>
	<style>
		body {
			font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
			background: linear-gradient(135deg, #667eea, #764ba2);
			color: #fff;
			margin: 0;
			padding: 20px;
			min-height: 100vh;
		}
		.container {
			max-width: 600px;
			margin: 0 auto;
			background: rgba(0, 0, 0, 0.4);
			padding: 2rem;
			border-radius: 1rem;
			box-shadow: 0 8px 20px rgba(0,0,0,0.3);
		}
		h1 {
			font-size: 2.5rem;
			margin-bottom: 0.5rem;
			text-align: center;
		}
		.subtitle {
			font-size: 1.1rem;
			margin: 0.5rem 0 2rem 0;
			text-align: center;
			opacity: 0.9;
		}
		.version {
			display: inline-block;
			background: #fff;
			color: #764ba2;
			padding: 0.3rem 0.8rem;
			border-radius: 999px;
			font-weight: bold;
			font-size: 0.9rem;
			margin-bottom: 2rem;
		}
		.todo-input-section {
			margin-bottom: 2rem;
		}
		.input-container {
			display: flex;
			gap: 0.5rem;
			margin-bottom: 0.5rem;
		}
		#todoInput {
			flex: 1;
			padding: 0.75rem;
			border: none;
			border-radius: 0.5rem;
			font-size: 1rem;
			background: rgba(255, 255, 255, 0.9);
			color: #333;
		}
		#todoInput:focus {
			outline: 2px solid #fff;
			background: #fff;
		}
		#descriptionInput {
			width: 100%;
			padding: 0.75rem;
			border: none;
			border-radius: 0.5rem;
			font-size: 1rem;
			background: rgba(255, 255, 255, 0.9);
			color: #333;
			margin-top: 0.5rem;
			resize: vertical;
			min-height: 60px;
		}
		#descriptionInput:focus {
			outline: 2px solid #fff;
			background: #fff;
		}
		#sendButton {
			padding: 0.75rem 1.5rem;
			border: none;
			border-radius: 0.5rem;
			background: #fff;
			color: #764ba2;
			font-weight: bold;
			cursor: pointer;
			transition: transform 0.2s;
		}
		#sendButton:hover {
			transform: translateY(-1px);
		}
		#sendButton:disabled {
			opacity: 0.6;
			cursor: not-allowed;
			transform: none;
		}
		.char-counter {
			font-size: 0.9rem;
			text-align: right;
			margin-top: 0.25rem;
			opacity: 0.8;
		}
		.char-counter.warning {
			color: #ffeb3b;
		}
		.char-counter.error {
			color: #ff5722;
		}
		.todos-section h2 {
			margin-bottom: 1rem;
			font-size: 1.5rem;
		}
		.todo-list {
			list-style: none;
			padding: 0;
		}
		.todo-item {
			background: rgba(255, 255, 255, 0.1);
			margin: 0.5rem 0;
			padding: 0.75rem 1rem;
			border-radius: 0.5rem;
			border-left: 4px solid #fff;
			backdrop-filter: blur(10px);
		}
		.todo-text {
			margin: 0 0 0.5rem 0;
			font-size: 1rem;
			line-height: 1.4;
			font-weight: 600;
		}
		.todo-description {
			margin: 0;
			font-size: 0.9rem;
			line-height: 1.3;
			opacity: 0.8;
		}
		.todo-meta {
			font-size: 0.8rem;
			opacity: 0.6;
			margin-top: 0.5rem;
			display: flex;
			justify-content: space-between;
			align-items: center;
		}
		.todo-id {
			background: rgba(255, 255, 255, 0.2);
			padding: 0.2rem 0.5rem;
			border-radius: 0.3rem;
			font-family: 'Courier New', monospace;
		}
		.loading {
			text-align: center;
			opacity: 0.7;
			font-style: italic;
		}
		.error {
			background: rgba(255, 87, 34, 0.2);
			color: #ff5722;
			padding: 1rem;
			border-radius: 0.5rem;
			margin: 1rem 0;
			border-left: 4px solid #ff5722;
		}
		.success {
			background: rgba(76, 175, 80, 0.2);
			color: #4caf50;
			padding: 1rem;
			border-radius: 0.5rem;
			margin: 1rem 0;
			border-left: 4px solid #4caf50;
		}
		.refresh-btn {
			background: rgba(255, 255, 255, 0.2);
			color: #fff;
			border: 1px solid rgba(255, 255, 255, 0.3);
			padding: 0.5rem 1rem;
			border-radius: 0.5rem;
			cursor: pointer;
			font-size: 0.9rem;
			margin-left: 1rem;
			transition: background 0.2s;
		}
		.refresh-btn:hover {
			background: rgba(255, 255, 255, 0.3);
		}
	</style>
</head>
<body>
	<div class="container">
		<h1>🚀 Todo App</h1>
		<p class="subtitle">Manage tasks, boost productivity, and stay organized.</p>
		<div class="version">v1.0.0 - Connected to Backend</div>
		
		<div class="todo-input-section">
			<h2>Add New Todo</h2>
			<div class="input-container">
				<input 
					type="text" 
					id="todoInput" 
					placeholder="What needs to be done?"
					maxlength="140"
				/>
				<button id="sendButton">Send</button>
			</div>
			<textarea 
				id="descriptionInput" 
				placeholder="Optional description..."
				maxlength="500"
			></textarea>
			<div class="char-counter" id="charCounter">0/140</div>
		</div>

		<div id="messageArea"></div>

		<div class="todos-section">
			<h2>
				Your Todos 
				<button class="refresh-btn" id="refreshButton">🔄 Refresh</button>
			</h2>
			<div id="todoContainer">
				<div class="loading">Loading todos...</div>
			</div>
		</div>

		<div class="image">
			<img src="/image" alt="Random Hourly Image" loading="lazy"/>
		</div>

	</div>

	<script>
		const API_BASE_URL = 'http://localhost:8081';
		
		const todoInput = document.getElementById('todoInput');
		const descriptionInput = document.getElementById('descriptionInput');
		const sendButton = document.getElementById('sendButton');
		const charCounter = document.getElementById('charCounter');
		const todoContainer = document.getElementById('todoContainer');
		const messageArea = document.getElementById('messageArea');
		const refreshButton = document.getElementById('refreshButton');

		function updateCharCounter() {
			const length = todoInput.value.length;
			charCounter.textContent = length + '/140';
			
			// Remove existing classes
			charCounter.classList.remove('warning', 'error');
			
			if (length >= 140) {
				charCounter.classList.add('error');
				sendButton.disabled = true;
			} else if (length >= 120) {
				charCounter.classList.add('warning');
				sendButton.disabled = false;
			} else {
				sendButton.disabled = length === 0;
			}
		}

		function showMessage(message, type = 'success') {
			messageArea.innerHTML = '<div class="' + type + '">' + message + '</div>';
			setTimeout(() => {
				messageArea.innerHTML = '';
			}, 3000);
		}

		function formatDate(dateString) {
			const date = new Date(dateString);
			return date.toLocaleString();
		}

		async function loadTodos() {
			try {
				todoContainer.innerHTML = '<div class="loading">Loading todos...</div>';
				
				const response = await fetch(API_BASE_URL + '/todos');
				if (!response.ok) {
					throw new Error('Failed to fetch todos: ' + response.statusText);
				}
				
				const todos = await response.json();
				
				if (todos.length === 0) {
					todoContainer.innerHTML = '<div class="loading">No todos yet. Add your first one!</div>';
					return;
				}
				
				const todoList = document.createElement('ul');
				todoList.className = 'todo-list';
				
				todos.sort((a, b) => b.id - a.id);
				
				todos.forEach(todo => {
					const todoItem = document.createElement('li');
					todoItem.className = 'todo-item';
					
					todoItem.innerHTML =
						'<p class="todo-text">' + escapeHtml(todo.title) + '</p>' +
						(todo.description ? '<p class="todo-description">' + escapeHtml(todo.description) + '</p>' : '') +
						'<div class="todo-meta">' +
							'<span class="todo-id">#' + todo.id + '</span>' +
							'<span>Created: ' + formatDate(todo.created_at) + '</span>' +
						'</div>';
					
					todoList.appendChild(todoItem);
				});
				
				todoContainer.innerHTML = '';
				todoContainer.appendChild(todoList);
				
			} catch (error) {
				console.error('Error loading todos:', error);
				todoContainer.innerHTML = '<div class="error">Failed to load todos: ' + error.message + '</div>';
			}
		}

		async function createTodo() {
			const title = todoInput.value.trim();
			const description = descriptionInput.value.trim();
			
			if (!title || title.length > 140) {
				showMessage('Please enter a valid title (1-140 characters)', 'error');
				return;
			}

			try {
				sendButton.disabled = true;
				sendButton.textContent = 'Sending...';

				const response = await fetch(API_BASE_URL + '/todos', {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json',
					},
					body: JSON.stringify({
						title: title,
						description: description || undefined
					})
				});
				
				if (!response.ok) {
					const errorText = await response.text();
					throw new Error("Failed to create todo: " + errorText);
				}
				
				const newTodo = await response.json();
				
				// Clear inputs
				todoInput.value = '';
				descriptionInput.value = '';
				updateCharCounter();
				
				showMessage("Todo " +newTodo.title+ " created successfully!", 'success');
				
				// Reload todos to show the new one
				await loadTodos();
				
			} catch (error) {
				console.error('Error creating todo:', error);
				showMessage("Failed to create todo: " + error.message, 'error');
			} finally {
				sendButton.disabled = false;
				sendButton.textContent = 'Send';
				updateCharCounter(); // This will re-enable if input is valid
			}
		}

		function escapeHtml(text) {
			const div = document.createElement('div');
			div.textContent = text;
			return div.innerHTML;
		}

		// Event listeners
		todoInput.addEventListener('input', updateCharCounter);
		sendButton.addEventListener('click', createTodo);
		refreshButton.addEventListener('click', loadTodos);

		// Allow Enter key to send todo (only from title input)
		todoInput.addEventListener('keypress', function(e) {
			if (e.key === 'Enter' && !sendButton.disabled) {
				createTodo();
			}
		});

		// Initialize
		updateCharCounter();
		loadTodos();

		// Check if backend is accessible
		fetch(API_BASE_URL +"/health")
			.then(response => {
				if (response.ok) {
					console.log('✅ Backend connection successful');
				} else {
					throw new Error('Backend health check failed');
				}
			})
			.catch(error => {
				console.warn('⚠️ Backend not accessible:', error);
				showMessage('Warning: Cannot connect to backend. Make sure the Go service is running on localhost:8080', 'error');
			});
	</script>
</body>
</html>`

	fmt.Fprint(w, html)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status": "healthy"}`)
}

func handleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status": "ready"}`)
}

// /image endpoint -> serves current cached image
func handleImage(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	currentImagePath := imagePath
	currentImageTimestamp := imageTimestamp
	currentServeOldOnce := serveOldOnce
	mu.Unlock()

	now := time.Now()
	needsUpdate := now.Sub(currentImageTimestamp) > 10*time.Minute

	if needsUpdate {
		if currentServeOldOnce {
			// Fetch new image in background to avoid blocking the request
			go func() {
				if err := fetchNewImage(); err != nil {
					log.Printf("Error fetching new image: %v", err)
				}
			}()
		} else {
			// Allow serving old one more time
			mu.Lock()
			serveOldOnce = true
			mu.Unlock()
		}
	}

	// Check if image file exists before serving
	if currentImagePath == "" {
		// Try to fetch a new image if none exists
		if err := fetchNewImage(); err != nil {
			http.Error(w, "No image available", http.StatusServiceUnavailable)
			return
		}
		mu.RLock()
		currentImagePath = imagePath
		mu.RUnlock()
	}

	// Verify file exists
	if _, err := os.Stat(currentImagePath); os.IsNotExist(err) {
		// Try to fetch a new image if current one is missing
		if err := fetchNewImage(); err != nil {
			http.Error(w, "Image not available", http.StatusServiceUnavailable)
			return
		}
		mu.RLock()
		currentImagePath = imagePath
		mu.RUnlock()
	}

	w.Header().Set("Content-Type", "image/jpeg")

	http.ServeFile(w, r, currentImagePath)
}

// fetchNewImage downloads a random image and saves it to static directory
func fetchNewImage() error {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get("https://picsum.photos/800/600")
	if err != nil {
		return fmt.Errorf("failed to fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Clean up old images to prevent disk space issues
	cleanupOldImages()

	// Save to static dir with timestamp
	filename := filepath.Join(staticPath, fmt.Sprintf("pic_%d.jpg", time.Now().Unix()))
	out, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(filename) // Clean up partial file on error
		return fmt.Errorf("failed to save image: %w", err)
	}

	// Update global state
	mu.Lock()
	oldImagePath := imagePath
	imagePath = filename
	imageTimestamp = time.Now()
	serveOldOnce = false
	mu.Unlock()

	// Remove old image file
	if oldImagePath != "" {
		os.Remove(oldImagePath)
	}

	return nil
}

// cleanupOldImages removes old image files to prevent disk space issues
func cleanupOldImages() {
	entries, err := os.ReadDir(staticPath)
	if err != nil {
		return
	}

	now := time.Now()
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".jpg" {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			// Remove files older than 1 hour
			if now.Sub(info.ModTime()) > time.Hour {
				os.Remove(filepath.Join(staticPath, entry.Name()))
			}
		}
	}
}
