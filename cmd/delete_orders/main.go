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

	// 查找 test@test.com 用户的ID
	var userID int64
	err = db.QueryRow("SELECT id FROM users WHERE email = $1", "test@test.com").Scan(&userID)
	if err != nil {
		log.Printf("User not found: %v", err)
		return
	}
	fmt.Printf("Found user ID: %d\n", userID)

	// 查看该用户的申购记录
	rows, err := db.Query("SELECT id, product_id, amount, status, start_date, end_date FROM wealth_order WHERE user_id = $1", userID)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("\nCurrent orders:")
	count := 0
	for rows.Next() {
		var id, productID int64
		var amount string
		var status int
		var startDate, endDate string
		rows.Scan(&id, &productID, &amount, &status, &startDate, &endDate)
		fmt.Printf("  Order ID: %d, Product: %d, Amount: %s, Status: %d, Period: %s to %s\n",
			id, productID, amount, status, startDate, endDate)
		count++
	}

	if count == 0 {
		fmt.Println("  No orders found")
	}

	// 删除该用户的申购记录
	result, err := db.Exec("DELETE FROM wealth_order WHERE user_id = $1", userID)
	if err != nil {
		log.Fatal(err)
	}

	deletedCount, _ := result.RowsAffected()
	fmt.Printf("\nDeleted %d order(s)\n", deletedCount)
}
