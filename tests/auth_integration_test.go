package tests

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"monera-digital/internal/config"
	"monera-digital/internal/models"
	"monera-digital/internal/services"
	"monera-digital/internal/utils"

	"github.com/lib/pq"
)

func getTestDB(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("RUN_AUTH_POSTGRES_INTEGRATION") != "1" {
		t.Skip("set RUN_AUTH_POSTGRES_INTEGRATION=1 to run PostgreSQL auth integration coverage")
	}
	// Load DB URL manually since we might not have viper init
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Try to read .env
		content, _ := os.ReadFile("../.env")
		for _, line := range strings.Split(string(content), "\n") {
			if strings.HasPrefix(line, "DATABASE_URL=") {
				dbURL = strings.Trim(strings.TrimPrefix(strings.TrimSpace(line), "DATABASE_URL="), "'\"")
				break
			}
		}
	}
	if dbURL == "" {
		t.Fatal("DATABASE_URL is required when RUN_AUTH_POSTGRES_INTEGRATION=1")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("Failed to connect to DB: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close DB: %v", err)
		}
	})
	return db
}

func prepareAuthIntegrationSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	schema := fmt.Sprintf("auth_integration_%d", time.Now().UnixNano())
	quotedSchema := pq.QuoteIdentifier(schema)
	if _, err := db.Exec("CREATE SCHEMA " + quotedSchema); err != nil {
		t.Fatalf("Failed to create isolated auth schema: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec("SET search_path TO public"); err != nil {
			t.Errorf("Failed to reset auth integration search path: %v", err)
		}
		if _, err := db.Exec("DROP SCHEMA " + quotedSchema + " CASCADE"); err != nil {
			t.Errorf("Failed to drop isolated auth schema: %v", err)
		}
	})

	if _, err := db.Exec("SET search_path TO " + quotedSchema); err != nil {
		t.Fatalf("Failed to select isolated auth schema: %v", err)
	}

	const schemaSQL = `
CREATE TABLE users (
    id                    SERIAL PRIMARY KEY,
    email                 VARCHAR(255) UNIQUE NOT NULL,
    password              VARCHAR(255) NOT NULL,
    status                VARCHAR(32) NOT NULL,
    two_factor_enabled    BOOLEAN NOT NULL DEFAULT FALSE,
    activation_code       VARCHAR(255),
    activation_expires_at TIMESTAMP,
    created_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE account (
    id             SERIAL PRIMARY KEY,
    user_id        INTEGER NOT NULL REFERENCES users(id),
    type           VARCHAR(32) NOT NULL,
    currency       VARCHAR(32) NOT NULL,
    balance        DECIMAL(32, 16) NOT NULL DEFAULT 0,
    frozen_balance DECIMAL(32, 16) NOT NULL DEFAULT 0,
    version        BIGINT NOT NULL DEFAULT 1,
    created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);`
	if _, err := db.Exec(schemaSQL); err != nil {
		t.Fatalf("Failed to create auth integration tables: %v", err)
	}
}

func TestRegisterAndLogin(t *testing.T) {
	db := getTestDB(t)
	prepareAuthIntegrationSchema(t, db)
	utils.SetActivationCodeKey([]byte("0123456789abcdef0123456789abcdef"))
	t.Cleanup(func() { utils.SetActivationCodeKey(nil) })

	testEmail := fmt.Sprintf("test_%d@example.com", time.Now().UnixNano())

	jwtSecret := "test-jwt-secret-for-integration-tests"
	authService := services.NewAuthService(db, jwtSecret, &config.Config{CoreAPIURL: "http://127.0.0.1:1"})

	// 1. Test Register
	req := models.RegisterRequest{
		Email:    testEmail,
		Password: "password123",
	}

	user, err := authService.Register(req)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if user.Email != testEmail {
		t.Errorf("Expected email %s, got %s", testEmail, user.Email)
	}
	if user.ID == 0 {
		t.Errorf("Expected non-zero ID")
	}
	if user.Status != models.UserStatusPending {
		t.Errorf("Expected pending user status, got %q", user.Status)
	}

	// 2. Test Login
	loginReq := models.LoginRequest{
		Email:    testEmail,
		Password: "password123",
	}

	resp, err := authService.Login(loginReq)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if !resp.RequiresActivation {
		t.Error("Expected activation requirement for a newly registered user")
	}
	if resp.Token != "" {
		t.Error("Expected no token before account activation")
	}
	if resp.User.Email != testEmail {
		t.Errorf("Expected user email %s, got %s", testEmail, resp.User.Email)
	}

	// 3. Test Invalid Login
	badReq := models.LoginRequest{
		Email:    testEmail,
		Password: "wrongpassword",
	}
	_, err = authService.Login(badReq)
	if err == nil {
		t.Errorf("Expected error for wrong password, got nil")
	}
}

func TestPasswordHashing(t *testing.T) {
	password := "securePass"
	hash, err := utils.HashPassword(password)
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}

	if !utils.CheckPasswordHash(password, hash) {
		t.Errorf("Password check failed")
	}

	if utils.CheckPasswordHash("wrong", hash) {
		t.Errorf("Password check matched wrong password")
	}
}
