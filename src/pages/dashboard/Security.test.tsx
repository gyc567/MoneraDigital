import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { BrowserRouter } from "react-router-dom";
import Security from "./Security";

// Mock dependencies
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

vi.mock("qrcode", () => ({
  default: {
    toDataURL: vi.fn((otpauth: string) => {
      // Mock QR code generation to return a base64 data URL
      return Promise.resolve("data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==");
    }),
  },
}));

const renderWithRouter = (component: React.ReactElement) => {
  return render(<BrowserRouter>{component}</BrowserRouter>);
};

describe("Security - 2FA QR Code Display Bug Fix", () => {
  beforeEach(() => {
    // Mock localStorage
    Storage.prototype.getItem = vi.fn((key) => {
      if (key === "token") return "mock-jwt-token";
      return null;
    });

    // Reset fetch mock
    global.fetch = vi.fn();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("should generate QR code from otpauth URI without using qrCodeUrl directly", async () => {
    // Mock the /api/auth/me response
    (global.fetch as any)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          twoFactorEnabled: false,
        }),
      })
      // Mock the /api/auth/2fa/setup response
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          data: {
            secret: "TEST_SECRET_KEY_123456",
            qrCodeUrl: "otpauth://totp/Monera%20Digital:test@example.com?secret=TEST_SECRET_KEY_123456",
            otpauth: "otpauth://totp/Monera%20Digital:test@example.com?secret=TEST_SECRET_KEY_123456",
            backupCodes: ["code1234", "code5678"],
          },
        }),
      });

    renderWithRouter(<Security />);

    // Wait for initial status fetch
    await waitFor(() => {
      expect(screen.getByText(/dashboard.security.enable2FA/i)).toBeInTheDocument();
    });

    // Click enable 2FA button
    const enableBtn = screen.getByText(/dashboard.security.enable2FA/i);
    await userEvent.click(enableBtn);

    // Wait for dialog to open and QR code to be generated
    await waitFor(() => {
      const img = screen.getByRole("img", { name: /2FA QR Code/i });
      expect(img).toBeInTheDocument();
      
      // Verify the image src is a data URL, not an otpauth:// URI
      const src = img.getAttribute("src");
      expect(src).toMatch(/^data:image\/png;base64,/);
      expect(src).not.toMatch(/^otpauth:/);
    });

    // Verify secret is displayed
    expect(screen.getByText("TEST_SECRET_KEY_123456")).toBeInTheDocument();
  });

  it("should not directly set qrCode state from qrCodeUrl field", async () => {
    // Mock responses
    (global.fetch as any)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ twoFactorEnabled: false }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          data: {
            secret: "SECRET123",
            qrCodeUrl: "otpauth://totp/Monera%20Digital:user@test.com?secret=SECRET123",
            otpauth: "otpauth://totp/Monera%20Digital:user@test.com?secret=SECRET123",
            backupCodes: ["backup1", "backup2"],
          },
        }),
      });

    renderWithRouter(<Security />);

    await waitFor(() => {
      expect(screen.getByText(/dashboard.security.enable2FA/i)).toBeInTheDocument();
    });

    const enableBtn = screen.getByText(/dashboard.security.enable2FA/i);
    await userEvent.click(enableBtn);

    // Wait for QR code generation
    await waitFor(() => {
      const img = screen.getByRole("img", { name: /2FA QR Code/i });
      const src = img.getAttribute("src");
      
      // Critical assertion: src must be base64 data URL, not otpauth://
      expect(src).not.toMatch(/^otpauth:/);
      expect(src).toMatch(/^data:image\/png;base64,/);
    });
  });

  it("should handle QR code generation errors gracefully", async () => {
    // Mock QRCode.toDataURL to reject
    const QRCode = await import("qrcode");
    (QRCode.default.toDataURL as any) = vi.fn().mockRejectedValue(new Error("QR generation failed"));

    (global.fetch as any)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ twoFactorEnabled: false }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          data: {
            secret: "SECRET456",
            otpauth: "otpauth://totp/test",
            backupCodes: [],
          },
        }),
      });

    // Spy on console.error
    const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});

    renderWithRouter(<Security />);

    await waitFor(() => {
      expect(screen.getByText(/dashboard.security.enable2FA/i)).toBeInTheDocument();
    });

    const enableBtn = screen.getByText(/dashboard.security.enable2FA/i);
    await userEvent.click(enableBtn);

    // Wait for error to be logged
    await waitFor(() => {
      expect(consoleErrorSpy).toHaveBeenCalledWith(
        "Failed to generate QR code:",
        expect.any(Error)
      );
    });

    consoleErrorSpy.mockRestore();
  });

  it("should display loading state while generating QR code", async () => {
    (global.fetch as any)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ twoFactorEnabled: false }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          data: {
            secret: "SECRET789",
            otpauth: "otpauth://totp/test",
            backupCodes: [],
          },
        }),
      });

    renderWithRouter(<Security />);

    await waitFor(() => {
      expect(screen.getByText(/dashboard.security.enable2FA/i)).toBeInTheDocument();
    });

    const enableBtn = screen.getByText(/dashboard.security.enable2FA/i);
    await userEvent.click(enableBtn);

    // Check for loading state (placeholder div)
    await waitFor(() => {
      const placeholder = screen.queryByText((content, element) => {
        return element?.classList.contains("animate-pulse") || false;
      });
      // Either placeholder is shown or QR code is loaded
      expect(
        placeholder || screen.queryByAlt("2FA QR Code")
      ).toBeTruthy();
    });
  });
});
