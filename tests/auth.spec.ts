import { test, expect } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

test.describe('Authentication', () => {
  const email = `test-${Date.now()}@example.com`;
  const password = 'Password123!';

  test('should register a new user', async ({ page }) => {
    await page.goto('/register');
    await page.fill('#email', email);
    await page.fill('#password', password);
    await page.click('button[type="submit"]');

    // Should redirect to login page
    await expect(page).toHaveURL(/.*login/);
    // Should show success toast
    await expect(page.getByText('Account created successfully')).toBeVisible();
  });

  test('should login with the new user', async ({ page }) => {
    await page.goto('/login');
    await page.fill('#email', email);
    await page.fill('#password', password);
    await page.click('button[type="submit"]');

    // Should redirect to home page
    await expect(page).toHaveURL('http://127.0.0.1:3000/');
    // Should show success toast
    await expect(page.getByText('Logged in successfully')).toBeVisible();
  });
});
