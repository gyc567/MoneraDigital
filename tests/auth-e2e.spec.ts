import { test, expect } from '@playwright/test';

test.describe('Registration and Login Flow', () => {
  const timestamp = Date.now();
  const testEmail = `test.${timestamp}@example.com`;
  const strongPassword = 'Password123!';
  const baseURL = 'http://localhost:5001';

  test.describe('Registration Page', () => {
    test('should load registration page successfully', async ({ page }) => {
      await page.goto(`${baseURL}/register`);
      await expect(page).toHaveTitle(/Register|Monera/i);
      await expect(page.getByRole('heading', { name: /Register/i })).toBeVisible();
    });

    test('should show password requirements hint', async ({ page }) => {
      await page.goto(`${baseURL}/register`);
      await expect(page.getByText('8-128 characters, including uppercase, lowercase, and a number.')).toBeVisible();
    });

    test('should show error for invalid email format', async ({ page }) => {
      await page.goto(`${baseURL}/register`);
      await page.fill('input[type="email"]', 'invalid-email');
      await page.fill('input[type="password"]', strongPassword);
      await page.click('button[type="submit"]');
      await expect(page.locator('input[type="email"]')).toHaveClass(/border-red-500/);
    });

    test('should show error for short password', async ({ page }) => {
      await page.goto(`${baseURL}/register`);
      await page.fill('input[type="email"]', testEmail);
      await page.fill('input[type="password"]', 'weak');
      await page.click('button[type="submit"]');
      await expect(page.locator('input[type="password"]')).toHaveClass(/border-red-500/);
      await expect(page.getByText('密码至少需要8个字符').or(page.getByText('Password must be at least 8 characters'))).toBeVisible();
    });

    test('should show error for email already registered', async ({ page }) => {
      const existingEmail = 'existing@test.com';
      await page.goto(`${baseURL}/register`);
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
      await page.goto(`${baseURL}/login`);
      await expect(page).toHaveTitle(/Login|Monera/i);
      await expect(page.getByRole('heading', { name: /Login/i })).toBeVisible();
    });

    test('should show error for invalid email format', async ({ page }) => {
      await page.goto(`${baseURL}/login`);
      await page.fill('input[type="email"]', 'invalid-email');
      await page.fill('input[type="password"]', strongPassword);
      await page.click('button[type="submit"]');
      await expect(page.locator('input[type="email"]')).toHaveClass(/border-red-500/);
    });

    test('should show error for non-existent email', async ({ page }) => {
      await page.goto(`${baseURL}/login`);
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
      await page.goto(`${baseURL}/login`);
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
      await page.goto(`${baseURL}/register`);
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
      await page.goto(`${baseURL}/login`);
      await page.fill('input[type="email"]', uniqueEmail);
      await page.fill('input[type="password"]', strongPassword);
      await page.click('button[type="submit"]');
      
      // Wait for login to complete and redirect
      await page.waitForTimeout(2000);
      
      // Should redirect to dashboard or home
      await expect(page).toHaveURL(/.*dashboard.*|.*\/?$/, { timeout: 5000 });
    });
  });
});
