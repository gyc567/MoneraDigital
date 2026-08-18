import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { TwoFactorService } from "./two-factor-service";

function jsonResponse(body: Record<string, unknown>, ok = true): Response {
  return {
    ok,
    json: vi.fn().mockResolvedValue(body),
  } as unknown as Response;
}

describe("TwoFactorService", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it("requires authentication before setup", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch");

    await expect(TwoFactorService.setup2FA()).rejects.toThrow("Not authenticated");
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("sets up 2FA through the authenticated API", async () => {
    localStorage.setItem("token", "access-token");
    const setup = { secret: "secret", otpauth: "otpauth://totp/example" };
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(jsonResponse(setup));

    await expect(TwoFactorService.setup2FA()).resolves.toEqual(setup);
    expect(fetchMock).toHaveBeenCalledWith("/api/auth/2fa/setup", {
      method: "POST",
      headers: { Authorization: "Bearer access-token" },
    });
  });

  it("surfaces setup errors from the backend", async () => {
    localStorage.setItem("token", "access-token");
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      jsonResponse({ error: "2FA setup unavailable" }, false)
    );

    await expect(TwoFactorService.setup2FA()).rejects.toThrow("2FA setup unavailable");
  });

  it("enables 2FA with the validated token", async () => {
    localStorage.setItem("token", "access-token");
    const enabled = { enabled: true };
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(jsonResponse(enabled));

    await expect(TwoFactorService.enable2FA("123456")).resolves.toEqual(enabled);
    expect(fetchMock).toHaveBeenCalledWith("/api/auth/2fa/enable", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: "Bearer access-token",
      },
      body: JSON.stringify({ token: "123456" }),
    });
  });

  it("rejects malformed enable tokens before calling the API", async () => {
    localStorage.setItem("token", "access-token");
    const fetchMock = vi.spyOn(globalThis, "fetch");

    await expect(TwoFactorService.enable2FA("12345")).rejects.toThrow(
      "2FA code must be 6 digits"
    );
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("disables 2FA through the authenticated API", async () => {
    localStorage.setItem("token", "access-token");
    const disabled = { enabled: false };
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(jsonResponse(disabled));

    await expect(TwoFactorService.disable2FA("654321")).resolves.toEqual(disabled);
    expect(fetchMock).toHaveBeenCalledWith("/api/auth/2fa/disable", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: "Bearer access-token",
      },
      body: JSON.stringify({ token: "654321" }),
    });
  });

  it("returns a disabled status without authentication", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch");

    await expect(TwoFactorService.get2FAStatus()).resolves.toEqual({ enabled: false });
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("falls back to disabled when the status request fails", async () => {
    localStorage.setItem("token", "access-token");
    vi.spyOn(globalThis, "fetch").mockResolvedValue(jsonResponse({}, false));

    await expect(TwoFactorService.get2FAStatus()).resolves.toEqual({ enabled: false });
  });

  it("verifies a 2FA token through the authenticated API", async () => {
    localStorage.setItem("token", "access-token");
    const verification = { valid: true };
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      jsonResponse(verification)
    );

    await expect(TwoFactorService.verify2FAToken("123456")).resolves.toEqual(verification);
    expect(fetchMock).toHaveBeenCalledWith("/api/auth/2fa/verify", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: "Bearer access-token",
      },
      body: JSON.stringify({ token: "123456" }),
    });
  });
});
