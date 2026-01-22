package db

import (
	"database/sql"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func InitDB(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(300)

	if err = db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	log.Println("Database connected successfully (using pgx driver)")
	return db, nil
}
