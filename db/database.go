package db

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // This is the driver for PostgreSQL
)

// DB is a global variable we will use in our handlers to run queries
var DB *sqlx.DB

func InitDB() {
	// Load the .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Note: No .env file found, using system environment variables")
	}

	//  Get the connection string from .env
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		log.Fatal("DB_URL is not set in the .env file")
	}

	fmt.Println("DEBUG: Connecting to:", dsn)

	//  Open the connection
	DB, err = sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Test if the connection is actually alive
	err = DB.Ping()
	if err != nil {
		log.Fatalf("Database is connected but unreachable: %v", err)
	}

	fmt.Println("Successfully connected to PostgreSQL!")
}
