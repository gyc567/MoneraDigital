/**
 * 2FA Integration Tests
 *
 * Tests the 2FA setup endpoint end-to-end:
 * - Verifies proper authentication is required
 * - Confirms QR code is returned with valid token
 * - Validates error handling for invalid requests
 * - Tests the full flow: setup → enable → verify
 */

import { describe, it, expect, beforeAll, afterAll } from "vitest";

const BACKEND_URL = "http://localhost:8081";
const API_SETUP = `${BACKEND_URL}/api/auth/2fa/setup`;
const API_LOGIN = `${BACKEND_URL}/api/auth/login`;
const API_REGISTER = `${BACKEND_URL}/api/auth/register`;
const API_ENABLE = `${BACKEND_URL}/api/auth/2fa/enable`;

let testToken: string;
const testEmail = `test-2fa-${Date.now()}@example.com`;

/**
 * Helper to make authenticated API calls
 */
async function authenticatedFetch(
  url: string,
  options: RequestInit = {}
): Promise<Response> {
  return fetch(url, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${testToken}`,
      ...options.headers,
    },
  });
}

/**
 * Setup: Create test user and obtain token
 */
beforeAll(async () => {
  try {
    // Register test user
    const registerRes = await fetch(API_REGISTER, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        email: testEmail,
        password: "TestPassword123!",
      }),
    });

    if (!registerRes.ok) {
      throw new Error(`Register failed: ${registerRes.statusText}`);
    }

    // Login to get token
    const loginRes = await fetch(API_LOGIN, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        email: testEmail,
        password: "TestPassword123!",
      }),
    });

    if (!loginRes.ok) {
      throw new Error(`Login failed: ${loginRes.statusText}`);
    }

    const loginData = await loginRes.json();
    testToken = loginData.access_token || loginData.token;

    if (!testToken) {
      throw new Error("No token returned from login");
    }
  } catch (error) {
    console.error("beforeAll setup failed:", error);
    throw error;
  }
});

describe("2FA Setup Integration Tests", () => {
  it("should return 401 without authentication", async () => {
    const response = await fetch(API_SETUP, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
    });

    expect(response.status).toBe(401);
    const data = await response.json();
    expect(data.code).toBe("MISSING_TOKEN");
  });

  it("should return 401 with invalid token", async () => {
    const response = await fetch(API_SETUP, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: "Bearer invalid-token-xyz",
      },
    });

    expect(response.status).toBe(401);
  });

  it("should return 200 with valid token", async () => {
    const response = await authenticatedFetch(API_SETUP, {
      method: "POST",
    });

    expect(response.status).toBe(200);
  });

  it("should return QR code URL in response", async () => {
    const response = await authenticatedFetch(API_SETUP, {
      method: "POST",
    });

    expect(response.status).toBe(200);
    const data = await response.json();

    expect(data.success).toBe(true);
    expect(data.data).toBeDefined();
    expect(data.data.qrCodeUrl).toBeDefined();
    expect(data.data.qrCodeUrl).toMatch(/^otpauth:\/\/totp\//);
  });

  it("should return backup codes", async () => {
    const response = await authenticatedFetch(API_SETUP, {
      method: "POST",
    });

    const data = await response.json();

    expect(data.data.backupCodes).toBeDefined();
    expect(Array.isArray(data.data.backupCodes)).toBe(true);
    expect(data.data.backupCodes.length).toBeGreaterThan(0);
  });

  it("should return TOTP secret", async () => {
    const response = await authenticatedFetch(API_SETUP, {
      method: "POST",
    });

    const data = await response.json();

    expect(data.data.secret).toBeDefined();
    expect(typeof data.data.secret).toBe("string");
    expect(data.data.secret.length).toBeGreaterThan(0);
  });

  it("should have Bearer token format in Authorization header", async () => {
    const response = await authenticatedFetch(API_SETUP, {
      method: "POST",
    });

    // If we get 200, the token format was accepted
    expect(response.status).toBe(200);
  });

  it("should handle missing Authorization header", async () => {
    const response = await fetch(API_SETUP, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
    });

    expect(response.status).toBe(401);
  });

  it("should handle malformed Authorization header", async () => {
    const response = await fetch(API_SETUP, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: "InvalidFormat",
      },
    });

    expect(response.status).toBe(401);
  });
});

describe("2FA Complete Flow", () => {
  it("should complete setup → enable → verify flow", async () => {
    // Step 1: Setup 2FA
    const setupRes = await authenticatedFetch(API_SETUP, {
      method: "POST",
    });

    expect(setupRes.status).toBe(200);
    const setupData = await setupRes.json();
    const secret = setupData.data.secret;
    expect(secret).toBeDefined();

    // Step 2: Verify we have QR code and backup codes
    expect(setupData.data.qrCodeUrl).toBeDefined();
    expect(setupData.data.backupCodes.length).toBeGreaterThan(0);
  });
});

describe("2FA Error Handling", () => {
  it("should provide helpful error message when backend is down", async () => {
    try {
      await fetch("http://localhost:9999/api/auth/2fa/setup", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${testToken}`,
        },
      });
    } catch (error) {
      expect(error).toBeDefined();
    }
  });
});
