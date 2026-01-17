import { test, expect } from '@playwright/test';

test.describe('Market Data and Global Stats', () => {
  const timestamp = Date.now();
  const testEmail = `market.test.${timestamp}@example.com`;
  const strongPassword = 'Password123!';

  test.beforeEach(async ({ page }) => {
    // Register
    await page.goto('/register');
    await page.fill('input[type="email"]', testEmail);
    await page.fill('input[type="password"]', strongPassword);
    await page.click('button[type="submit"]');
    
    // Wait for potential redirect or toast
    await page.waitForTimeout(2000);

    // Login
    await page.goto('/login');
    await page.fill('input[type="email"]', testEmail);
    await page.fill('input[type="password"]', strongPassword);
    await page.click('button[type="submit"]');
    
    // Wait for dashboard redirect
    await page.waitForURL(/\/dashboard/, { timeout: 15000 });
  });

  test('should display global crypto stats on dashboard', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Verify BTC and ETH are mentioned (Global Stats)
    // We expect to see "BTC" and "ETH" labels or tickers
    await expect(page.locator('body')).toContainText('BTC');
    await expect(page.locator('body')).toContainText('ETH');
    
    // Verify Price indications (roughly)
    // Looking for a dollar sign which indicates values are rendering
    await expect(page.locator('body')).toContainText('$');
  });

  test('should verify market monitor components', async ({ page }) => {
    await page.goto('/dashboard');
    
    // "Market Cap" and "Volume" should be visible in the stats bar
    await expect(page.locator('body')).toContainText('Market Cap');
    await expect(page.locator('body')).toContainText('Volume');
    
    // Check if there are values associated (not empty)
    // This is a bit heuristic, but we check if the stats container exists
    // Assuming standard Shadcn cards or stats display
    const stats = page.locator('.gap-4').first(); // Common grid class
    await expect(stats).toBeVisible();
  });
});
