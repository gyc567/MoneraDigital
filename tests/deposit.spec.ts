import { test, expect } from '@playwright/test';

test.describe('User Deposit Flow', () => {
  const timestamp = Date.now();
  const email = `deposit.test.${timestamp}@example.com`;
  const password = 'Password123!';
  const baseURL = 'http://localhost:5001';

  test('should activate wallet and view deposit address', async ({ page }) => {
    // 1. Register and Login
    await page.goto(`${baseURL}/register`);
    await page.fill('input[type="email"]', email);
    await page.fill('input[type="password"]', password);
    await page.click('button[type="submit"]');
    
    // Wait for login redirect
    await page.waitForURL(`${baseURL}/login`);
    
    await page.fill('input[type="email"]', email);
    await page.fill('input[type="password"]', password);
    await page.click('button[type="submit"]');
    
    // Wait for dashboard
    await expect(page).toHaveURL(`${baseURL}/dashboard`);

    // 2. Navigate to Deposit Page
    // Click "Deposit" in sidebar
    await page.getByRole('link', { name: 'Deposit' }).click();
    await expect(page).toHaveURL(`${baseURL}/dashboard/deposit`);

    // 3. Verify "Activate Account" Prompt
    // Since wallet is not created, we expect the activate prompt
    await expect(page.getByText('Activate Account').first()).toBeVisible();
    
    // Click "Activate Account" button (redirects to account-opening)
    await page.getByRole('button', { name: 'Activate Account' }).click();
    await expect(page).toHaveURL(`${baseURL}/dashboard/account-opening`);

    // 4. Activate Wallet
    // Click "Activate Now"
    // Note: The button text might be "Activate Now" or translated key.
    // In en.json: "activateButton": "Activate Now"
    await page.getByRole('button', { name: 'Activate Now' }).click();

    // Wait for mock creation (2s delay in backend mock, frontend might show loading)
    // The frontend mock in AccountOpening.tsx uses setTimeout(2000).
    // The BACKEND API is also mocked with 2s delay. 
    // AccountOpening.tsx calls the API?
    // Let's check AccountOpening.tsx implementation again.
    // It currently uses `handleCreateWallet` which has `setTimeout` AND sets local state.
    // Wait, did I update AccountOpening.tsx to call the API?
    // The previous turns updated `docs` and `Deposit.tsx`, but I don't recall overwriting `AccountOpening.tsx` with API calls.
    // I wrote the PLAN to update it, but did I execute?
    // Let's check `src/pages/dashboard/AccountOpening.tsx`.
  });
});
