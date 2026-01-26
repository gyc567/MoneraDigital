import { test, expect } from "@playwright/test";

test.describe("2FA QR Code Display Bug Fix", () => {
  test.beforeEach(async ({ page }) => {
    // Intercept console errors
    page.on("console", (msg) => {
      if (msg.type() === "error") {
        console.error("Browser console error:", msg.text());
      }
    });

    // Register a test user if needed
    const registerResponse = await page.request.post("/api/auth/register", {
      data: {
        email: "qrcode-test@example.com",
        password: "TestPass123!",
        confirmPassword: "TestPass123!",
      },
    });

    if (!registerResponse.ok() && registerResponse.status() !== 409) {
      console.error("Registration failed:", await registerResponse.text());
    }
  });

  test("should display QR code as image, not otpauth:// URI", async ({ page }) => {
    // Navigate to login page
    await page.goto("/login");

    // Login
    await page.fill('input[name="email"]', "qrcode-test@example.com");
    await page.fill('input[name="password"]', "TestPass123!");
    await page.click('button[type="submit"]');

    // Wait for navigation to dashboard
    await page.waitForURL("**/dashboard");

    // Navigate to security page
    await page.goto("/dashboard/security");
    await page.waitForLoadState("networkidle");

    // Collect console errors
    const errors: string[] = [];
    page.on("console", (msg) => {
      if (msg.type() === "error") {
        errors.push(msg.text());
      }
    });

    // Find and click Enable 2FA button
    const enableButton = page.locator("text=/Enable 2FA|enable2FA/i").first();
    await expect(enableButton).toBeVisible({ timeout: 10000 });
    await enableButton.click();

    // Wait for dialog to open
    await page.waitForSelector('[role="dialog"]', { timeout: 10000 });

    // Wait for QR code image to load
    const qrImage = page.locator('img[alt="2FA QR Code"]');
    await expect(qrImage).toBeVisible({ timeout: 15000 });

    // Get the src attribute
    const src = await qrImage.getAttribute("src");
    
    // Critical assertion: Must be data URL, not otpauth://
    expect(src).toBeTruthy();
    expect(src).toMatch(/^data:image\/png;base64,/);
    expect(src).not.toMatch(/^otpauth:/);

    // Verify no ERR_UNKNOWN_URL_SCHEME errors
    const hasOtpauthError = errors.some((error) =>
      error.includes("ERR_UNKNOWN_URL_SCHEME") || error.includes("otpauth://")
    );
    expect(hasOtpauthError).toBe(false);

    // Verify QR code is actually an image (not broken)
    const imageNaturalWidth = await qrImage.evaluate((img: HTMLImageElement) => img.naturalWidth);
    expect(imageNaturalWidth).toBeGreaterThan(0);

    // Verify secret key is displayed
    const secretElement = page.locator("code").first();
    await expect(secretElement).toBeVisible();
    const secretText = await secretElement.textContent();
    expect(secretText).toBeTruthy();
    expect(secretText?.length).toBeGreaterThan(10);
  });

  test("should allow copying secret key", async ({ page, context }) => {
    // Grant clipboard permissions
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);

    await page.goto("/login");
    await page.fill('input[name="email"]', "qrcode-test@example.com");
    await page.fill('input[name="password"]', "TestPass123!");
    await page.click('button[type="submit"]');
    await page.waitForURL("**/dashboard");

    await page.goto("/dashboard/security");
    await page.waitForLoadState("networkidle");

    const enableButton = page.locator("text=/Enable 2FA|enable2FA/i").first();
    await enableButton.click();

    await page.waitForSelector('[role="dialog"]');

    // Wait for secret to be visible
    const secretElement = page.locator("code").first();
    await expect(secretElement).toBeVisible();
    const secretText = await secretElement.textContent();

    // Click copy button
    const copyButton = page.locator('button[class*="h-8 w-8"]').first();
    await copyButton.click();

    // Wait a bit for clipboard operation
    await page.waitForTimeout(500);

    // Verify toast message appears (optional, depends on implementation)
    const toast = page.locator("text=/copied/i");
    await expect(toast).toBeVisible({ timeout: 5000 });
  });

  test("should show Open in App link with valid otpauth URI", async ({ page }) => {
    await page.goto("/login");
    await page.fill('input[name="email"]', "qrcode-test@example.com");
    await page.fill('input[name="password"]', "TestPass123!");
    await page.click('button[type="submit"]');
    await page.waitForURL("**/dashboard");

    await page.goto("/dashboard/security");
    await page.waitForLoadState("networkidle");

    const enableButton = page.locator("text=/Enable 2FA|enable2FA/i").first();
    await enableButton.click();

    await page.waitForSelector('[role="dialog"]');

    // Find the "Open in App" link
    const openInAppLink = page.locator('a[href^="otpauth://"]');
    await expect(openInAppLink).toBeVisible();

    const href = await openInAppLink.getAttribute("href");
    expect(href).toMatch(/^otpauth:\/\/totp\//);
    expect(href).toContain("secret=");
    expect(href).toContain("issuer=Monera");
  });

  test("should not affect other security features", async ({ page }) => {
    await page.goto("/login");
    await page.fill('input[name="email"]', "qrcode-test@example.com");
    await page.fill('input[name="password"]', "TestPass123!");
    await page.click('button[type="submit"]');
    await page.waitForURL("**/dashboard");

    await page.goto("/dashboard/security");
    await page.waitForLoadState("networkidle");

    // Verify password change section exists
    const passwordSection = page.locator("text=/dashboard.security.password/i").first();
    await expect(passwordSection).toBeVisible();

    // Verify whitelist section exists
    const whitelistSection = page.locator("text=/dashboard.security.whitelist/i");
    await expect(whitelistSection).toBeVisible();

    // Verify manage addresses button works
    const manageAddressesBtn = page.locator("text=/dashboard.security.manageAddresses/i");
    await expect(manageAddressesBtn).toBeVisible();
  });
});
