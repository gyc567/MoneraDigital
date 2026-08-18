import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { AuthService } from "./auth-service";
import { tokenManager } from "./token-manager";

function jsonResponse(body: Record<string, unknown>, ok = true): Response {
  return {
    ok,
    json: vi.fn().mockResolvedValue(body),
  } as unknown as Response;
}

describe("AuthService", () => {
  beforeEach(() => {
    tokenManager.clearTokens();
    localStorage.clear();
  });

  afterEach(() => {
    tokenManager.clearTokens();
    vi.restoreAllMocks();
  });

  describe("register", () => {
    it("surfaces the backend registration-disabled response", async () => {
      const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
        jsonResponse(
          {
            code: "REGISTRATION_DISABLED",
            error: "Registration is disabled",
          },
          false
        )
      );

      await expect(
        AuthService.register("test@example.com", "password123")
      ).rejects.toThrow("Registration is disabled");
      expect(fetchMock).toHaveBeenCalledWith("/api/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: "test@example.com", password: "password123" }),
      });
    });

    it("rejects invalid credentials before calling the API", async () => {
      const fetchMock = vi.spyOn(globalThis, "fetch");

      await expect(AuthService.register("invalid-email", "password123")).rejects.toThrow(
        "Invalid email format"
      );
      await expect(AuthService.register("test@example.com", "short")).rejects.toThrow(
        "Password must be at least 8 characters"
      );
      expect(fetchMock).not.toHaveBeenCalled();
    });
  });

  describe("login", () => {
    it("stores backend tokens and the authenticated user", async () => {
      const loginResponse = {
        accessToken: "access-token",
        refreshToken: "refresh-token",
        tokenType: "Bearer",
        expiresIn: 3600,
        user: { id: 1, email: "test@example.com" },
      };
      vi.spyOn(globalThis, "fetch").mockResolvedValue(jsonResponse(loginResponse));

      await expect(
        AuthService.login("test@example.com", "password123")
      ).resolves.toEqual(loginResponse);
      expect(tokenManager.getAccessToken()).toBe("access-token");
      expect(tokenManager.getRefreshToken()).toBe("refresh-token");
      expect(JSON.parse(localStorage.getItem("user") || "null")).toEqual(loginResponse.user);
    });

    it("surfaces backend login errors", async () => {
      vi.spyOn(globalThis, "fetch").mockResolvedValue(
        jsonResponse({ error: "Invalid email or password" }, false)
      );

      await expect(
        AuthService.login("test@example.com", "password123")
      ).rejects.toThrow("Invalid email or password");
    });

    it("rejects malformed credentials before calling the API", async () => {
      const fetchMock = vi.spyOn(globalThis, "fetch");

      await expect(AuthService.login("invalid-email", "password123")).rejects.toThrow(
        "Invalid email format"
      );
      expect(fetchMock).not.toHaveBeenCalled();
    });
  });

  it("returns no current user when there is no access token", async () => {
    await expect(AuthService.getCurrentUser()).resolves.toBeNull();
  });
});
