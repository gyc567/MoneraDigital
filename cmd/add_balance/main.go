package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	dsn := "postgresql://neondb_owner:npg_4zuq7JQNWFDB@ep-bold-cloud-adfpuk12-pooler.c-2.us-east-1.aws.neon.tech/neondb?sslmode=require"

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	userID := int64(64)

	currencies := []struct {
		currency string
		balance  string
	}{
		{"USDT", "50000"},
		{"USDC", "10000"},
		{"BTC", "5"},
		{"ETH", "50"},
	}

	fmt.Printf("User ID: %d\n\n", userID)

	for _, c := range currencies {
		var accountID int64
		err = db.QueryRow(`
			SELECT id FROM account 
			WHERE user_id = $1 AND currency = $2 AND type = 'FUND'
		`, userID, c.currency).Scan(&accountID)

		if err == sql.ErrNoRows {
			err = db.QueryRow(`
				INSERT INTO account (user_id, type, currency, balance, frozen_balance, version, created_at, updated_at)
				VALUES ($1, 'FUND', $2, $3, '0', 1, NOW(), NOW())
				RETURNING id
			`, userID, c.currency, c.balance).Scan(&accountID)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Created %s: %s (ID: %d)\n", c.currency, c.balance, accountID)
		} else if err != nil {
			log.Fatal(err)
		} else {
			_, err = db.Exec(`
				UPDATE account SET balance = $1, updated_at = NOW()
				WHERE id = $2
			`, c.balance, accountID)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Updated %s: %s (ID: %d)\n", c.currency, c.balance, accountID)
		}
	}

	fmt.Println("\n--- Account Balances ---")
	rows, err := db.Query(`
		SELECT currency, balance, frozen_balance 
		FROM account 
		WHERE user_id = $1 AND type = 'FUND'
	`, userID)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var currency, balance, frozen string
		rows.Scan(&currency, &balance, &frozen)
		fmt.Printf("  %s: Balance=%s, Frozen=%s\n", currency, balance, frozen)
	}
}
