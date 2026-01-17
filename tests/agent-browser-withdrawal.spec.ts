import { test, expect } from '@playwright/test';

test.describe('Agent Browser: User Login and Withdrawal Flow', () => {
  const timestamp = Date.now();
  const testEmail = `agent.withdraw.${timestamp}@example.com`;
  const strongPassword = 'Password123!';

  test.beforeAll(async ({ browser }) => {
    // Optional: Setup global state if needed, but we'll do per-test user creation for isolation
  });

  test('should complete login and verify withdrawal restrictions for new user', async ({ page }) => {
    // --- Step 1: User Registration & Login ---
    console.log('Step 1: Registering and Logging in...');
    await page.goto('/register');
    await page.fill('input[type="email"]', testEmail);
    await page.fill('input[type="password"]', strongPassword);
    await page.click('button[type="submit"]');
    
    // Wait for navigation
    await page.waitForLoadState('networkidle');

    // Check where we are
    const url = page.url();
    console.log('Current URL after register:', url);

    if (url.includes('/login')) {
        console.log('Redirected to login. Logging in manually...');
        await page.fill('input[type="email"]', testEmail);
        await page.fill('input[type="password"]', strongPassword);
        await page.click('button[type="submit"]');
    } else if (url.includes('/dashboard')) {
        console.log('Auto-logged in to dashboard.');
    } else {
        // Fallback: Force goto login if stuck on register or elsewhere
        console.log('Unknown state, forcing navigation to login...');
        await page.goto('/login');
        await page.fill('input[type="email"]', testEmail);
        await page.fill('input[type="password"]', strongPassword);
        await page.click('button[type="submit"]');
    }
    
    // Wait for dashboard
    await page.waitForURL(/\/dashboard/, { timeout: 15000 });
    console.log('Login successful, on dashboard.');

    // --- Step 2: Verify Withdrawal Page (Initial State) ---
    console.log('Step 2: Checking Withdrawal page for empty state...');
    await page.goto('/dashboard/withdraw');
    
    // We expect a message about no verified addresses
    // Matches text from Withdraw.tsx: "No verified addresses available. Please add and verify an address first."
    await expect(page.getByText(/No verified addresses available/i)).toBeVisible();
    console.log('Verified: Withdrawal blocked due to no addresses.');

    // --- Step 3: Add an Address ---
    console.log('Step 3: Adding a new withdrawal address...');
    await page.goto('/dashboard/addresses');
    
    // Click "Add Address"
    await page.getByRole('button', { name: /Add Address/i }).click();
    
    // Fill Dialog
    await expect(page.getByRole('dialog')).toBeVisible();
    await page.fill('input#label', 'My Test Wallet');
    // Select ETH (default)
    // Fill Address
    await page.fill('input#address', '0x71C7656EC7ab88b098defB751B7401B5f6d8976F'); // Valid format ETH address
    
    // Submit (target button in dialog footer)
    await page.locator('div[role="dialog"] button').filter({ hasText: 'Add Address' }).click();

    // Check for success or error toast
    const successToast = page.getByText(/Address added successfully/i);
    const errorToast = page.getByText(/Failed to add address/i);
    
    await expect(successToast.or(errorToast)).toBeVisible();
    
    if (await errorToast.isVisible()) {
        throw new Error('Failed to add address - API returned error');
    }
    
    // Verify it appears in the list (Pending)
    await expect(page.getByText(/No addresses added yet/i)).not.toBeVisible();
    await expect(page.getByText('My Test Wallet')).toBeVisible();
    await expect(page.getByText('Pending')).toBeVisible();
    console.log('Address added and visible as Pending.');

    // --- Step 4: Verify Withdrawal Page (Unverified Address) ---
    console.log('Step 4: Checking Withdrawal page with unverified address...');
    await page.goto('/dashboard/withdraw');
    
    // Should STILL show "No verified addresses available" because the address is not verified
    // This confirms the security requirement that only verified addresses can be used.
    await expect(page.getByText(/No verified addresses available/i)).toBeVisible();
    console.log('Verified: Unverified address does not enable withdrawal.');
  });
});
