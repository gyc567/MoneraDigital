package main

import (
	"strings"
	"testing"
)

func TestLoadSmokeConfigRequiresExplicitDatabaseURL(t *testing.T) {
	t.Setenv("AIRWALLEX_BASE_URL", "https://api.sandbox.airwallex.com")
	t.Setenv("AIRWALLEX_CLIENT_ID", "sandbox-client")
	t.Setenv("AIRWALLEX_API_KEY", "sandbox-key")
	t.Setenv("AIRWALLEX_LOGIN_AS", "sandbox-account")
	t.Setenv("MONERA_DATABASE_URL", "")
	t.Setenv("DATABASE_URL", "")

	_, err := loadSmokeConfig(false)
	if err == nil {
		t.Fatal("smoke config accepted a missing database URL")
	}
	if message := err.Error(); !strings.Contains(message, "MONERA_DATABASE_URL") || !strings.Contains(message, "DATABASE_URL") {
		t.Fatalf("missing database error = %q", message)
	}
}
