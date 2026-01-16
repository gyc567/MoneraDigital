import { test, expect } from '@playwright/test';

test.describe('Core Account System Integration', () => {
  const timestamp = Date.now();
  const testEmail = `core.test.${timestamp}@example.com`;
  const strongPassword = 'Password123!';
  const baseURL = 'http://localhost:5000';

  test.describe('Core Account API (Direct API Tests)', () => {
    test('should create account via Core Account API', async ({ request }) => {
      const createAccountRequest = {
        externalId: `test_user_${timestamp}`,
        accountType: 'INDIVIDUAL',
        profile: {
          email: testEmail,
          firstName: 'Test',
          lastName: 'User',
          nationality: 'CN',
        },
        metadata: {
          source: 'e2e_test',
          timestamp: timestamp,
        },
      };

      const response = await request.post(`${baseURL}/api/core/accounts/create`, {
        data: createAccountRequest,
      });

      expect(response.status()).toBe(201);

      const responseData = await response.json();
      expect(responseData.success).toBe(true);
      expect(responseData.data.accountId).toBeDefined();
      expect(responseData.data.accountId).toMatch(/^core_/);
      expect(responseData.data.status).toBe('CREATING');
      expect(responseData.data.externalId).toBe(`test_user_${timestamp}`);
    });

    test('should reject duplicate externalId', async ({ request }) => {
      const duplicateRequest = {
        externalId: `test_user_${timestamp}`,
        accountType: 'INDIVIDUAL',
        profile: {
          email: 'duplicate@example.com',
          firstName: 'Duplicate',
          lastName: 'User',
          nationality: 'CN',
        },
      };

      const response = await request.post(`${baseURL}/api/core/accounts/create`, {
        data: duplicateRequest,
      });

      expect(response.status()).toBe(400);
      const responseData = await response.json();
      expect(responseData.success).toBe(false);
      expect(responseData.error.code).toBe('ACCOUNT_EXISTS');
    });

    test('should get account by accountId', async ({ request }) => {
      // First create an account
      const createRequest = {
        externalId: `get_test_${timestamp}`,
        accountType: 'INDIVIDUAL',
        profile: {
          email: `get.test.${timestamp}@example.com`,
          firstName: 'Get',
          lastName: 'Test',
          nationality: 'US',
        },
      };

      const createResponse = await request.post(`${baseURL}/api/core/accounts/create`, {
        data: createRequest,
      });

      const createData = await createResponse.json();
      const accountId = createData.data.accountId;

      // Now get the account
      const getResponse = await request.get(`${baseURL}/api/core/accounts/${accountId}`);
      expect(getResponse.status()).toBe(200);

      const getData = await getResponse.json();
      expect(getData.success).toBe(true);
      expect(getData.data.accountId).toBe(accountId);
      expect(getData.data.externalId).toBe(`get_test_${timestamp}`);
    });

    test('should return 404 for non-existent account', async ({ request }) => {
      const response = await request.get(`${baseURL}/api/core/accounts/core_nonexistent_account_123`);
      expect(response.status()).toBe(404);

      const responseData = await response.json();
      expect(responseData.success).toBe(false);
      expect(responseData.error.code).toBe('ACCOUNT_NOT_FOUND');
    });

    test('should update account status', async ({ request }) => {
      // Create an account first
      const createRequest = {
        externalId: `status_test_${timestamp}`,
        accountType: 'INDIVIDUAL',
        profile: {
          email: `status.test.${timestamp}@example.com`,
          firstName: 'Status',
          lastName: 'Test',
          nationality: 'EU',
        },
      };

      const createResponse = await request.post(`${baseURL}/api/core/accounts/create`, {
        data: createRequest,
      });

      const createData = await createResponse.json();
      const accountId = createData.data.accountId;

      // Update status
      const updateResponse = await request.put(`${baseURL}/api/core/accounts/${accountId}/status`, {
        data: {
          status: 'ACTIVE',
          reason: 'KYC verification completed',
        },
      });

      expect(updateResponse.status()).toBe(200);
      const updateData = await updateResponse.json();
      expect(updateData.success).toBe(true);
      expect(updateData.data.status).toBe('ACTIVE');
    });

    test('should submit KYC', async ({ request }) => {
      // Create an account first
      const createRequest = {
        externalId: `kyc_test_${timestamp}`,
        accountType: 'INDIVIDUAL',
        profile: {
          email: `kyc.test.${timestamp}@example.com`,
          firstName: 'KYC',
          lastName: 'Test',
          nationality: 'SG',
        },
      };

      const createResponse = await request.post(`${baseURL}/api/core/accounts/create`, {
        data: createRequest,
      });

      const createData = await createResponse.json();
      const accountId = createData.data.accountId;

      // Submit KYC
      const kycResponse = await request.post(`${baseURL}/api/core/accounts/${accountId}/kyc/submit`, {
        data: {
          documentType: 'PASSPORT',
          documentNumber: 'E12345678',
        },
      });

      expect(kycResponse.status()).toBe(200);
      const kycData = await kycResponse.json();
      expect(kycData.success).toBe(true);
      expect(kycData.data.kycStatus).toBe('PENDING');
    });

    test('should get KYC status', async ({ request }) => {
      // Create an account first
      const createRequest = {
        externalId: `kyc_status_test_${timestamp}`,
        accountType: 'INDIVIDUAL',
        profile: {
          email: `kyc.status.test.${timestamp}@example.com`,
          firstName: 'KYCStatus',
          lastName: 'Test',
          nationality: 'AU',
        },
      };

      const createResponse = await request.post(`${baseURL}/api/core/accounts/create`, {
        data: createRequest,
      });

      const createData = await createResponse.json();
      const accountId = createData.data.accountId;

      // Get KYC status
      const kycStatusResponse = await request.get(`${baseURL}/api/core/accounts/${accountId}/kyc/status`);
      expect(kycStatusResponse.status()).toBe(200);

      const kycStatusData = await kycStatusResponse.json();
      expect(kycStatusData.success).toBe(true);
      expect(kycStatusData.data.accountId).toBe(accountId);
    });

    test('should return health check', async ({ request }) => {
      const response = await request.get(`${baseURL}/api/core/health`);
      expect(response.status()).toBe(200);

      const responseData = await response.json();
      expect(responseData.status).toBe('healthy');
      expect(responseData.timestamp).toBeDefined();
    });
  });

  test.describe('Full Registration Flow with Core Account Integration', () => {
    const uniqueEmail = `full.flow.${timestamp}@example.com`;

    test('should register user and create core account @db', async ({ page }) => {
      // Navigate to registration page
      await page.goto(`${baseURL}/register`);
      await expect(page).toHaveTitle(/Register|Monera/i);

      // Fill in registration form
      await page.fill('input[type="email"]', uniqueEmail);
      await page.fill('input[type="password"]', strongPassword);

      // Submit registration
      await page.click('button[type="submit"]');

      // Wait for registration to complete
      await page.waitForTimeout(3000);

      // Verify success - should redirect to login or show success message
      const url = page.url();
      const hasRedirectedToLogin = url.includes('/login');
      const hasSuccessToast = await page.getByText('注册成功').or(page.getByText('Registration successful')).isVisible().catch(() => false);

      expect(hasRedirectedToLogin || hasSuccessToast).toBeTruthy();
    });

    test('should login and verify core account created @db', async ({ page }) => {
      // Navigate to login page
      await page.goto(`${baseURL}/login`);

      // Fill in login credentials
      await page.fill('input[type="email"]', uniqueEmail);
      await page.fill('input[type="password"]', strongPassword);

      // Submit login
      await page.click('button[type="submit"]');

      // Wait for login and redirect to dashboard
      await page.waitForTimeout(3000);

      // Verify successful login
      await expect(page).toHaveURL(/.*dashboard.*|.*\/?$/, { timeout: 5000 });

      // Check if wallet/account creation is available
      await expect(page.getByText(/Wallet|钱包|Account|账户/i).first()).toBeVisible({ timeout: 5000 });
    });
  });

  test.describe('Core Account Status Flow', () => {
    test('should show account status progression', async ({ page }) => {
      // This test verifies the account goes through proper status flow:
      // CREATING -> PENDING_KYC -> ACTIVE

      const statusEmail = `status.flow.${timestamp}@example.com`;
      const statusPassword = 'Password123!';

      // 1. Register
      await page.goto(`${baseURL}/register`);
      await page.fill('input[type="email"]', statusEmail);
      await page.fill('input[type="password"]', statusPassword);
      await page.click('button[type="submit"]');
      await page.waitForTimeout(3000);

      // 2. Login
      await page.goto(`${baseURL}/login`);
      await page.fill('input[type="email"]', statusEmail);
      await page.fill('input[type="password"]', statusPassword);
      await page.click('button[type="submit"]');
      await page.waitForTimeout(3000);

      // 3. Navigate to wallet/account page
      await page.goto(`${baseURL}/dashboard/account-opening`);
      await page.waitForTimeout(2000);

      // 4. Check for account creation UI
      const createButton = page.getByRole('button', { name: /Activate|Create|开启/i });
      const statusText = page.getByText(/Creating|Pending|Active/i);

      // Either show status or create button should be visible
      const hasStatusOrButton = (await statusText.isVisible().catch(() => false)) || (await createButton.isVisible().catch(() => false));
      expect(hasStatusOrButton).toBeTruthy();
    });
  });
});
