package companyfund

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAirwallexClient_ValidatesExactScopedAccountIdentity(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	loginAsValues := make([]string, 0, 1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v1/authentication/login":
			loginAsValues = append(loginAsValues, request.Header.Get("x-login-as"))
			_, _ = response.Write([]byte(airwallexTestLoginResponse("candidate-token", now.Add(time.Hour))))
		case "/api/v1/account":
			if request.Header.Get("Authorization") != "Bearer candidate-token" ||
				request.Header.Get("x-api-version") != airwallexTestAPIVersion {
				t.Errorf("unexpected account request headers: %#v", request.Header)
			}
			_, _ = response.Write([]byte(`{"id":"Acct_Case","account_name":"Sandbox Candidate"}`))
		default:
			t.Errorf("unexpected path %q", request.URL.Path)
		}
	}))
	defer server.Close()

	client := newTestAirwallexClientWithLoginAs(
		t, server.URL, server.Client(), func() time.Time { return now }, time.Minute, "old-current",
	)
	summary, err := client.ValidateAirwallexAccountIdentity(context.Background(), "Acct_Case")
	if err != nil {
		t.Fatal(err)
	}
	if summary.ProviderAccountID != "Acct_Case" || summary.AccountName != "Sandbox Candidate" ||
		len(loginAsValues) != 1 || loginAsValues[0] != "Acct_Case" {
		t.Fatalf("summary=%#v loginAs=%v", summary, loginAsValues)
	}
	if client.PinnedLoginAsScope() != "old-current" {
		t.Fatalf("validation mutated shared client scope: %q", client.PinnedLoginAsScope())
	}
}

func TestAirwallexClient_RejectsWhitespaceAndMismatchedAccountIdentity(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v1/authentication/login":
			_, _ = response.Write([]byte(airwallexTestLoginResponse("token", now.Add(time.Hour))))
		case "/api/v1/account":
			_, _ = response.Write([]byte(`{"id":"different"}`))
		}
	}))
	defer server.Close()
	client := newTestAirwallexClient(t, server.URL, server.Client(), func() time.Time { return now }, time.Minute)

	for _, key := range []string{"", " spaced", "spaced ", "different-id"} {
		if _, err := client.ValidateAirwallexAccountIdentity(context.Background(), key); err == nil {
			t.Fatalf("ValidateAirwallexAccountIdentity(%q) accepted", key)
		}
	}
}
