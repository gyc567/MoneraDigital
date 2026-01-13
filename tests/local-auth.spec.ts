import { test, expect } from '@playwright/test';

test.describe('Local Authentication Flow', () => {
  const timestamp = Date.now();
  const email = `local.test.${timestamp}@example.com`;
  const password = 'Password123!';
  const baseURL = 'http://localhost:5001';

  test('should register and login successfully', async ({ page }) => {
    // 1. Register
    console.log(`Navigating to ${baseURL}/register`);
    await page.goto(`${baseURL}/register`);
    
    console.log(`Registering with email: ${email}`);
    await page.fill('input[type="email"]', email);
    await page.fill('input[type="password"]', password);
    await page.click('button[type="submit"]');

    // Wait for network idle or navigation
    await page.waitForLoadState('networkidle');

    // Verification strategies for Registration Success:
    // Option A: Redirects to login
    // Option B: Redirects to dashboard (auto-login)
    // Option C: Shows a success message
    
    // Check for "Account created" or redirect
    const successMessage = page.getByText('Account created successfully');
    const loginRedirect = page.url().includes('/login');
    
    console.log('Checking registration outcome...');
    // We expect either a success message OR a redirect to login
    if (await successMessage.isVisible()) {
        console.log('Success message visible');
    }

    // 2. Login
    console.log(`Navigating to ${baseURL}/login`);
    await page.goto(`${baseURL}/login`);
    
    console.log('Logging in...');
    await page.fill('input[type="email"]', email);
    await page.fill('input[type="password"]', password);
    await page.click('button[type="submit"]');
    
    // Wait for response
    await page.waitForLoadState('networkidle');

    // 3. Verify Login
    console.log('Verifying login...');
    // Expect redirect to dashboard or home
    await expect(page).toHaveURL(`${baseURL}/dashboard`);
    
    // Check for dashboard element (e.g., "Overview" or "Dashboard")
    // Adjust selector based on actual dashboard content
    await expect(page.getByText('Portfolio Performance')).toBeVisible({ timeout: 10000 });
    
    console.log('Login verified successfully');
  });
});
