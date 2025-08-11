package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

// Database configuration struct
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

// Global DB variable
var db *sql.DB

func main() {
	// Initialize database configuration
	config := DBConfig{
		Host:     "127.0.0.1",
		Port:     "5432",
		User:     "worker_user",
		Password: "y8JZgF2h0bdCPiE2FAndrbJrfxg7k9sy",
		DBName:   "node_art_slot_games",
	}

	// Connect to database
	err := connectDB(config)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test the connection
	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	fmt.Println("Successfully connected to database!")

	// Example query
	if err := executeExampleQuery(); err != nil {
		log.Fatalf("Failed to execute query: %v", err)
	}
}

// connectDB establishes a connection to the PostgreSQL database
func connectDB(config DBConfig) error {
	// Create connection string
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.DBName,
	)

	// Open database connection
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("error opening database: %v", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return nil
}

// executeExampleQuery demonstrates how to execute a simple query
func executeExampleQuery() error {
	// Example: Create a table
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS example_table (
            id SERIAL PRIMARY KEY,
            name VARCHAR(100),
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `)
	if err != nil {
		return fmt.Errorf("error creating table: %v", err)
	}

	// Example: Insert data
	_, err = db.Exec(`
        INSERT INTO example_table (name) VALUES ($1)
    `, "Test Record")
	if err != nil {
		return fmt.Errorf("error inserting data: %v", err)
	}

	// Example: Query data
	rows, err := db.Query("SELECT id, name, created_at FROM example_table")
	if err != nil {
		return fmt.Errorf("error querying data: %v", err)
	}
	defer rows.Close()

	// Iterate through results
	for rows.Next() {
		var (
			id        int
			name      string
			createdAt string
		)
		if err := rows.Scan(&id, &name, &createdAt); err != nil {
			return fmt.Errorf("error scanning row: %v", err)
		}
		fmt.Printf("Record: ID=%d, Name=%s, CreatedAt=%s\n", id, name, createdAt)
	}

	return rows.Err()
}
