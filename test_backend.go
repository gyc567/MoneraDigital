//go:build ignore

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "https://monera-digital--gyc567.replit.app"

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func main() {
	fmt.Println("Starting Backend Auth Tests...")
	fmt.Printf("Target: %s\n\n", baseURL)

	testRoot()
	fmt.Println("------------------------------------------------")
	testDocs()
	fmt.Println("------------------------------------------------")
	testRegister()
	fmt.Println("------------------------------------------------")
	testLogin()
	fmt.Println("\nTests Completed.")
}

func testRoot() {
	fmt.Println("[TEST] Root Endpoint (GET /)")
	url := baseURL + "/"

	start := time.Now()
	resp, err := http.Get(url)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ Failed to send request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Latency: %v\n", duration)
	fmt.Println("Headers:")
	for k, v := range resp.Header {
		fmt.Printf("  %s: %s\n", k, v)
	}
	// Only print first 200 chars of body if it's HTML
	bodyStr := string(body)
	if len(bodyStr) > 200 {
		fmt.Printf("Response (truncated): %s...\n", bodyStr[:200])
	} else {
		fmt.Printf("Response: %s\n", bodyStr)
	}

	if resp.StatusCode == 200 {
		fmt.Println("✅ Result: SUCCESS (Service is up)")
	} else {
		fmt.Println("❌ Result: FAILED")
	}
}

func testDocs() {
	fmt.Println("[TEST] Docs Endpoint (GET /api/docs)")
	url := baseURL + "/api/docs"

	start := time.Now()
	resp, err := http.Get(url)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ Failed to send request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Latency: %v\n", duration)
	fmt.Println("Headers:")
	for k, v := range resp.Header {
		fmt.Printf("  %s: %s\n", k, v)
	}
	fmt.Printf("Response: %s\n", string(body))

	if resp.StatusCode == 200 {
		fmt.Println("✅ Result: SUCCESS")
	} else {
		fmt.Println("❌ Result: FAILED")
	}
}

func testRegister() {
	fmt.Println("[TEST] Register Endpoint")
	url := baseURL + "/api/auth/register"

	reqBody := RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	jsonData, _ := json.Marshal(reqBody)

	start := time.Now()
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ Failed to send request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Latency: %v\n", duration)
	fmt.Println("Headers:")
	for k, v := range resp.Header {
		fmt.Printf("  %s: %s\n", k, v)
	}
	fmt.Printf("Response: %s\n", string(body))

	if resp.StatusCode == 200 {
		fmt.Println("✅ Result: SUCCESS (Endpoint reachable)")
	} else {
		fmt.Println("❌ Result: FAILED (Unexpected status code)")
	}
}

func testLogin() {
	fmt.Println("[TEST] Login Endpoint")
	url := baseURL + "/api/auth/login"

	reqBody := LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	jsonData, _ := json.Marshal(reqBody)

	start := time.Now()
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ Failed to send request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Latency: %v\n", duration)
	fmt.Println("Headers:")
	for k, v := range resp.Header {
		fmt.Printf("  %s: %s\n", k, v)
	}
	fmt.Printf("Response: %s\n", string(body))

	if resp.StatusCode == 200 {
		fmt.Println("✅ Result: SUCCESS (Endpoint reachable)")
	} else {
		fmt.Println("❌ Result: FAILED (Unexpected status code)")
	}
}
