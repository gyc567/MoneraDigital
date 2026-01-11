package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not found")
	}

	dbName := extractDBName(dbURL)
	fmt.Printf("Target database: %s\n", dbName)

	fmt.Printf("Connecting to %s...\n", dbName)
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		fmt.Printf("Cannot connect to %s: %v\n", dbName, err)
		fmt.Println("Attempting to create via neondb...")

		mainDBURL := strings.Replace(dbURL, "/"+dbName+"?", "/neondb?", 1)
		mainDb, err := sql.Open("postgres", mainDBURL)
		if err != nil {
			log.Fatalf("Error opening main database: %v", err)
		}
		defer mainDb.Close()

		if err := mainDb.Ping(); err != nil {
			log.Fatalf("Error connecting to main database: %v", err)
		}

		_, err = mainDb.Exec("CREATE DATABASE " + dbName)
		if err != nil {
			if strings.Contains(err.Error(), "already exists") {
				fmt.Printf("Database %s already exists.\n", dbName)
			} else {
				log.Fatalf("Error creating database: %v", err)
			}
		} else {
			fmt.Printf("Database %s created.\n", dbName)
		}
	}

	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Error connecting: %v", err)
	}
	fmt.Println("Database connection verified.")

	sqlFilePath := "docs/静态理财/需求文档MD/数据库建表脚本.sql"
	fmt.Printf("Reading SQL file: %s\n", sqlFilePath)

	content, err := os.ReadFile(sqlFilePath)
	if err != nil {
		log.Fatalf("Error reading SQL file: %v", err)
	}

	fmt.Println("Executing SQL script...")

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Error starting transaction: %v", err)
	}

	_, err = tx.Exec(string(content))
	if err != nil {
		tx.Rollback()
		log.Fatalf("Error executing SQL: %v", err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("Error committing: %v", err)
	}

	fmt.Println("Successfully executed database migration script!")
	fmt.Printf("All tables created in '%s' database.\n", dbName)
}

func extractDBName(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) >= 4 {
		dbQuery := parts[3]
		if idx := strings.Index(dbQuery, "?"); idx != -1 {
			return dbQuery[:idx]
		}
		return dbQuery
	}
	return "neondb"
}
