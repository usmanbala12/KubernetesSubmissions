package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"

	_ "github.com/lib/pq"
)

var counter uint64
var db *sql.DB

func initDB() {
	connStr := os.Getenv("DATABASE_URL")

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	// Create table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS counter (
			id SERIAL PRIMARY KEY,
			value BIGINT NOT NULL
		);
	`)
	if err != nil {
		panic(err)
	}

	// Ensure one row exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM counter").Scan(&count)
	if err != nil {
		panic(err)
	}
	if count == 0 {
		_, err = db.Exec("INSERT INTO counter (value) VALUES (0)")
		if err != nil {
			panic(err)
		}
	}

	// Load current value into memory
	err = db.QueryRow("SELECT value FROM counter WHERE id = 1").Scan(&counter)
	if err != nil {
		panic(err)
	}
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	// increment atomically
	newCount := atomic.AddUint64(&counter, 1)

	// persist to DB
	_, err := db.Exec("UPDATE counter SET value = $1 WHERE id = 1", newCount)
	if err != nil {
		http.Error(w, "DB update failed", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "pong %d", newCount)
}

func handlePings(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%d", atomic.LoadUint64(&counter))
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Log Output Service - OK\n")
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	initDB()

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/pingpong", handlePing)
	http.HandleFunc("/pings", handlePings)

	fmt.Printf("Server started on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}
