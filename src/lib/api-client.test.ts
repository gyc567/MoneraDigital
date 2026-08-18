/**
 * API Client Tests
 *
 * Tests for api-client.ts including 401 error handling,
 * token management, and request/response handling.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { apiRequest, getApiUrl, getApiBaseUrl } from "./api-client";
import { tokenManager } from "./token-manager";

const TEST_API_BASE_URL = "http://127.0.0.1:8081";

describe("api-client", () => {
  beforeEach(() => {
    vi.spyOn(global, "fetch").mockReset();
    tokenManager.clearTokens();
  });

  afterEach(() => {
    tokenManager.clearTokens();
    vi.restoreAllMocks();
  });

  describe("getApiUrl", () => {
    it("should prefix paths with the configured backend URL", () => {
      const result = getApiUrl("api/wallet/info");
      expect(result).toBe(`${TEST_API_BASE_URL}/api/wallet/info`);
    });

    it("should handle path with leading slash", () => {
      const result = getApiUrl("/api/wallet/info");
      expect(result).toBe(`${TEST_API_BASE_URL}/api/wallet/info`);
    });
  });

  describe("getApiBaseUrl", () => {
    it("should return the configured backend URL", () => {
      const result = getApiBaseUrl();
      expect(result).toBe(TEST_API_BASE_URL);
    });
  });

  describe("apiRequest", () => {
    const createMockResponse = (options: {
      ok: boolean;
      status: number;
      statusText?: string;
      jsonData?: Record<string, unknown>;
      contentType?: string;
    }) => {
      const headers = new Map<string, string>();
      if (options.contentType) {
        headers.set("content-type", options.contentType);
      }
      return {
        ok: options.ok,
        status: options.status,
        statusText: options.statusText || (options.ok ? "OK" : "Error"),
        headers: {
          get: (key: string) => headers.get(key) || null,
        },
        json: options.jsonData ? vi.fn().mockResolvedValue(options.jsonData) : vi.fn(),
      };
    };

    it("should make request with Authorization header when token exists", async () => {
      const mockResponse = createMockResponse({
        ok: true,
        status: 200,
        jsonData: { status: "SUCCESS" },
        contentType: "application/json",
      });
      vi.spyOn(global, "fetch").mockResolvedValue(mockResponse as unknown as Response);
      tokenManager.setTokens({
        accessToken: "test-token-123",
        refreshToken: "refresh-token-123",
        tokenType: "Bearer",
        expiresIn: 3600,
      });

      const result = await apiRequest("/api/wallet/info");

      expect(global.fetch).toHaveBeenCalledWith(
        `${TEST_API_BASE_URL}/api/wallet/info`,
        expect.objectContaining({
          headers: expect.objectContaining({
            "Content-Type": "application/json",
            "Authorization": "Bearer test-token-123",
          }),
        })
      );
      expect(result).toEqual({ status: "SUCCESS" });
    });

    it("should make request without Authorization header when no token", async () => {
      const mockResponse = createMockResponse({
        ok: true,
        status: 200,
        jsonData: { status: "NONE" },
        contentType: "application/json",
      });
      vi.spyOn(global, "fetch").mockResolvedValue(mockResponse as unknown as Response);
      const result = await apiRequest("/api/wallet/info");

      expect(global.fetch).toHaveBeenCalledWith(
        `${TEST_API_BASE_URL}/api/wallet/info`,
        expect.objectContaining({
          headers: expect.objectContaining({
            "Content-Type": "application/json",
          }),
        })
      );
      expect(result).toEqual({ status: "NONE" });
    });

    it("should throw error with message on 401 Unauthorized", async () => {
      const mockResponse = createMockResponse({
        ok: false,
        status: 401,
        statusText: "Unauthorized",
        jsonData: {
          code: "MISSING_TOKEN",
          message: "Authentication required",
          error: "Unauthorized",
        },
        contentType: "application/json",
      });
      vi.spyOn(global, "fetch").mockResolvedValue(mockResponse as unknown as Response);
      tokenManager.setTokens({
        accessToken: "test-token",
        refreshToken: "",
        tokenType: "Bearer",
        expiresIn: 3600,
      });

      await expect(apiRequest("/api/wallet/info")).rejects.toThrow(
        "Session expired. Please log in again."
      );
    });

    it("should throw error with message on other HTTP errors", async () => {
      const mockResponse = createMockResponse({
        ok: false,
        status: 500,
        statusText: "Internal Server Error",
        jsonData: {
          code: "INTERNAL_ERROR",
          message: "Something went wrong",
        },
        contentType: "application/json",
      });
      vi.spyOn(global, "fetch").mockResolvedValue(mockResponse as unknown as Response);
      await expect(apiRequest("/api/wallet/info")).rejects.toThrow("Something went wrong");
    });

    it("should throw generic error on non-JSON error response", async () => {
      const mockResponse = createMockResponse({
        ok: false,
        status: 500,
        statusText: "Internal Server Error",
        contentType: "text/plain",
      });
      vi.spyOn(global, "fetch").mockResolvedValue(mockResponse as unknown as Response);
      await expect(apiRequest("/api/wallet/info")).rejects.toThrow("Internal Server Error");
    });

    it("should merge custom headers with default headers", async () => {
      const mockResponse = createMockResponse({
        ok: true,
        status: 200,
        jsonData: { status: "SUCCESS" },
        contentType: "application/json",
      });
      vi.spyOn(global, "fetch").mockResolvedValue(mockResponse as unknown as Response);
      tokenManager.setTokens({
        accessToken: "test-token",
        refreshToken: "refresh-token",
        tokenType: "Bearer",
        expiresIn: 3600,
      });

      await apiRequest("/api/wallet/info", {
        headers: {
          "X-Custom-Header": "custom-value",
        },
      });

      expect(global.fetch).toHaveBeenCalledWith(
        `${TEST_API_BASE_URL}/api/wallet/info`,
        expect.objectContaining({
          headers: expect.objectContaining({
            "Content-Type": "application/json",
            "Authorization": "Bearer test-token",
            "X-Custom-Header": "custom-value",
          }),
        })
      );
    });

    it("should pass through request body for POST requests", async () => {
      const mockResponse = createMockResponse({
        ok: true,
        status: 200,
        jsonData: { status: "SUCCESS" },
        contentType: "application/json",
      });
      vi.spyOn(global, "fetch").mockResolvedValue(mockResponse as unknown as Response);
      const requestBody = {
        productCode: "X_FINANCE",
        currency: "TRON",
      };

      await apiRequest("/api/wallet/create", {
        method: "POST",
        body: JSON.stringify(requestBody),
      });

      expect(global.fetch).toHaveBeenCalledWith(
        `${TEST_API_BASE_URL}/api/wallet/create`,
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify(requestBody),
        })
      );
    });
  });
});
