package main

import (
	"log"

	"github.com/joho/godotenv"
	"monera-digital/internal/config"
	"monera-digital/internal/db"
	"monera-digital/internal/migration"
	"monera-digital/internal/migration/migrations"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize database
	database, err := db.InitDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer database.Close()

	// Initialize migrator
	migrator := migration.NewMigrator(database)

	// Register migrations
	migrator.Register(&migrations.CreateUsersTable{})
	migrator.Register(&migrations.CreateLendingPositionsTable{})
	migrator.Register(&migrations.CreateWithdrawalTables{})
	migrator.Register(&migrations.AddTwoFactorColumnsMigration{})
	migrator.Register(&migrations.AddTwoFactorTimestampMigration{})
	migrator.Register(&migrations.UpdateWalletRequestsTable{})

	// Initialize migrations table
	if err := migrator.Init(); err != nil {
		log.Fatal("Failed to initialize migrations:", err)
	}

	// Check status
	status, err := migrator.GetStatus()
	if err != nil {
		log.Fatal("Failed to get migration status:", err)
	}

	for _, s := range status {
		log.Printf("Migration %s: %s (%s)\n", s.Version, s.Name, s.Status)
	}

	// Run migrations
	log.Println("Running migrations...")
	if err := migrator.Migrate(); err != nil {
		log.Fatal("Migration failed:", err)
	}

	log.Println("Migrations completed successfully")
}
