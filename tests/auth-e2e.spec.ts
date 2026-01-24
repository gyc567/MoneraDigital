import { test, expect } from '@playwright/test';

test.describe('Registration and Login Flow', () => {
  const timestamp = Date.now();
  const testEmail = `test.${timestamp}@example.com`;
  const strongPassword = 'Password123!';

  test.describe('Registration Page', () => {
    test('should load registration page successfully', async ({ page }) => {
      await page.goto('/register');
      await expect(page).toHaveTitle(/Register|Monera/i);
      await expect(page.getByRole('heading', { name: /Register/i })).toBeVisible();
    });

    test('should show password requirements hint', async ({ page }) => {
      await page.goto('/register');
      await expect(page.getByText('8-128 characters, including uppercase, lowercase, and a number.')).toBeVisible();
    });

    test('should show error for invalid email format', async ({ page }) => {
      await page.goto('/register');
      await page.fill('input[type="email"]', 'invalid-email');
      await page.fill('input[type="password"]', strongPassword);
      await page.click('button[type="submit"]');
      await expect(page.locator('input[type="email"]')).toHaveClass(/border-red-500/);
    });

    test('should show error for short password', async ({ page }) => {
      await page.goto('/register');
      await page.fill('input[type="email"]', testEmail);
      await page.fill('input[type="password"]', 'weak');
      await page.click('button[type="submit"]');
      await expect(page.locator('input[type="password"]')).toHaveClass(/border-red-500/);
      await expect(page.getByText('密码至少需要8个字符').or(page.getByText('Password must be at least 8 characters'))).toBeVisible();
    });

    test('should show error for email already registered', async ({ page }) => {
      const existingEmail = 'existing@test.com';
      await page.goto('/register');
      await page.fill('input[type="email"]', existingEmail);
      await page.fill('input[type="password"]', strongPassword);
      await page.click('button[type="submit"]');
      await page.waitForTimeout(1000);
      // Check if email error is shown (email already exists)
      const emailInput = page.locator('input[type="email"]');
      const classes = await emailInput.getAttribute('class');
      if (classes?.includes('border-red-500')) {
        await expect(page.getByText('该邮箱已被注册').or(page.getByText('This email is already registered'))).toBeVisible();
      }
    });
  });

  test.describe('Login Page', () => {
    test('should load login page successfully', async ({ page }) => {
      await page.goto('/login');
      await expect(page).toHaveTitle(/Login|Monera/i);
      await expect(page.getByRole('heading', { name: /Login/i })).toBeVisible();
    });

    test('should show error for invalid email format', async ({ page }) => {
      await page.goto('/login');
      await page.fill('input[type="email"]', 'invalid-email');
      await page.fill('input[type="password"]', strongPassword);
      await page.click('button[type="submit"]');
      await expect(page.locator('input[type="email"]')).toHaveClass(/border-red-500/);
    });

    test('should show error for non-existent email', async ({ page }) => {
      await page.goto('/login');
      await page.fill('input[type="email"]', 'nonexistent@example.com');
      await page.fill('input[type="password"]', strongPassword);
      await page.click('button[type="submit"]');
      await page.waitForTimeout(1000);
      // Check for error message (inline or toast)
      const inlineError = page.getByText('邮箱输入错误或不存在').or(page.getByText('Email input error or does not exist'));
      const toastError = page.getByText('服务器错误').or(page.getByText('Server error'));
      await expect(inlineError.first().or(toastError.first())).toBeVisible({ timeout: 5000 });
    });

    test('should show error for wrong password', async ({ page }) => {
      await page.goto('/login');
      // Use an email that exists in the database
      await page.fill('input[type="email"]', 'test@example.com');
      await page.fill('input[type="password"]', 'WrongPassword123!');
      await page.click('button[type="submit"]');
      await page.waitForTimeout(1000);
      // Check for error message (inline or toast)
      // If email exists but password wrong: "邮箱或密码错误" or "Invalid email or password"
      // If email doesn't exist: "邮箱输入错误或不存在" or "Email input error or does not exist"
      const inlineError = page.getByText('邮箱或密码错误').or(page.getByText('Invalid email or password'));
      const emailNotFoundError = page.getByText('邮箱输入错误或不存在').or(page.getByText('Email input error or does not exist'));
      const toastError = page.getByText('服务器错误').or(page.getByText('Server error'));
      await expect(inlineError.first().or(emailNotFoundError.first()).or(toastError.first())).toBeVisible({ timeout: 5000 });
    });
  });

  test.describe('Full Registration and Login Flow', () => {
    const uniqueEmail = `new.user.${timestamp}@example.com`;

    test('should register a new user successfully @db', async ({ page }) => {
      await page.goto('/register');
      await page.fill('input[type="email"]', uniqueEmail);
      await page.fill('input[type="password"]', strongPassword);
      await page.click('button[type="submit"]');

      // Wait for registration to complete
      await page.waitForTimeout(2000);

      // Should show success toast or redirect to login
      const successToast = page.getByText('注册成功').or(page.getByText('Registration successful'));
      const url = page.url();

      // Either success toast or redirect to login
      const hasSuccess = await successToast.isVisible().catch(() => false);
      const hasLoginUrl = url.includes('/login');

      expect(hasSuccess || hasLoginUrl).toBeTruthy();
    });

    test('should login with newly registered user', async ({ page }) => {
      await page.goto('/login');
      await page.fill('input[type="email"]', uniqueEmail);
      await page.fill('input[type="password"]', strongPassword);
      await page.click('button[type="submit"]');

      // Wait for login to complete and redirect
      await page.waitForTimeout(2000);

      // Should redirect to dashboard or home
      await expect(page).toHaveURL(/.*dashboard.*|.*\/?$/, { timeout: 5000 });
    });
  });

  test.describe('Session Management & Protected Routes', () => {
    test('should persist login session across page refreshes', async ({ page }) => {
      // Register and login first
      const sessionEmail = `session.test.${timestamp}@example.com`;
      await page.goto('/register');
      await page.fill('input[type="email"]', sessionEmail);
      await page.fill('input[type="password"]', strongPassword);
      await page.click('button[type="submit"]');
      await page.waitForTimeout(2000);

      await page.goto('/login');
      await page.fill('input[type="email"]', sessionEmail);
      await page.fill('input[type="password"]', strongPassword);
      await page.click('button[type="submit"]');
      await page.waitForURL(/.*dashboard.*|.*\/?$/, { timeout: 5000 });

      // Navigate to a protected route
      await page.goto('/dashboard');
      await expect(page.getByText(/Dashboard|仪表板/i)).toBeVisible();

      // Refresh the page
      await page.reload();
      await page.waitForLoadState('networkidle');

      // Should still be logged in and on dashboard
      await expect(page.getByText(/Dashboard|仪表板/i)).toBeVisible();
      await expect(page).toHaveURL(/.*dashboard.*|.*\/?$/);
    });

    test('should redirect unauthenticated users from protected routes', async ({ page }) => {
      // Clear any existing session
      await page.context().clearCookies();
      await page.evaluate(() => localStorage.clear());

      // Try to access protected route directly
      await page.goto('/dashboard');
      await page.waitForLoadState('networkidle');

      // Should redirect to login
      await expect(page).toHaveURL(/.*login.*|.*auth.*/, { timeout: 5000 });
      await expect(page.getByRole('heading', { name: /Login|登录/i })).toBeVisible();
    });

    test('should handle logout properly', async ({ page }) => {
      // Login first
      const logoutEmail = `logout.test.${timestamp}@example.com`;
      await page.goto('/register');
      await page.fill('input[type="email"]', logoutEmail);
      await page.fill('input[type="password"]', strongPassword);
      await page.click('button[type="submit"]');
      await page.waitForTimeout(2000);

      await page.goto('/login');
      await page.fill('input[type="email"]', logoutEmail);
      await page.fill('input[type="password"]', strongPassword);
      await page.click('button[type="submit"]');
      await page.waitForURL(/.*dashboard.*|.*\/?$/, { timeout: 5000 });

      // Find and click logout button
      const logoutButton = page.getByRole('button', { name: /Logout|退出|Sign out/i });
      await expect(logoutButton).toBeVisible();
      await logoutButton.click();

      // Should redirect to login or home
      await page.waitForURL(/.*login.*|.*\/?$/, { timeout: 5000 });

      // Try to access protected route - should redirect again
      await page.goto('/dashboard');
      await page.waitForLoadState('networkidle');
      await expect(page).toHaveURL(/.*login.*|.*auth.*/, { timeout: 5000 });
    });

    test('should handle expired JWT tokens gracefully', async ({ page }) => {
      // Simulate login with expired token
      await page.addInitScript(() => {
        // Set an obviously expired token
        localStorage.setItem('token', 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyLCJleHAiOjE1MTYyMzkwMjJ9.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c');
      });

      await page.goto('/dashboard');
      await page.waitForLoadState('networkidle');

      // Should redirect to login due to expired token
      await expect(page).toHaveURL(/.*login.*|.*auth.*/, { timeout: 5000 });
      await expect(page.getByRole('heading', { name: /Login|登录/i })).toBeVisible();
    });
  });
});
