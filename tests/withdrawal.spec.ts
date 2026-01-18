import { test, expect } from '@playwright/test';

test.describe('Withdrawal Feature E2E', () => {
  test.beforeEach(async ({ page }) => {
    // Mock Auth
    await page.route('/api/auth/me', async route => {
      await route.fulfill({ json: { id: 1, email: 'test@example.com' } });
    });

    // Mock Addresses
    await page.route('/api/addresses', async route => {
      await route.fulfill({
        json: {
          addresses: [
            { id: 1, wallet_address: '0x123...abc', chain_type: 'ERC20', address_alias: 'My Eth Wallet', verified: true }
          ]
        }
      });
    });

    // Mock Withdrawal History
    await page.route('/api/withdrawals', async route => {
      await route.fulfill({
        json: {
          withdrawals: []
        }
      });
    });

    // Mock Fees
    await page.route('/api/withdrawals/fees*', async route => {
      await route.fulfill({
        json: { fee: '1.5', receivedAmount: '98.5' }
      });
    });

    // Mock Create Withdrawal
    await page.route('/api/withdrawals', async route => {
      if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 201,
          json: { message: 'Withdrawal created', order: { id: 101, status: 'PENDING' } }
        });
      } else {
        await route.continue();
      }
    });

    // Simulate logged in state
    await page.addInitScript(() => {
      localStorage.setItem('token', 'fake-jwt-token');
    });
  });

  test('should load withdrawal page and display addresses', async ({ page }) => {
    await page.goto('/dashboard/withdraw');
    
    // Check Title
    await expect(page.getByText('Withdraw Funds')).toBeVisible();
    
    // Check Address Select exists
    await expect(page.getByText('My Eth Wallet')).toBeVisible();
  });

  test('should calculate fees and submit withdrawal', async ({ page }) => {
    await page.goto('/dashboard/withdraw');

    // Input Amount
    await page.getByLabel('Amount').fill('100');
    
    // Wait for fee calculation (mocked response)
    await expect(page.getByText('1.5 ETH')).toBeVisible(); // Fee
    await expect(page.getByText('98.5 ETH')).toBeVisible(); // Received

    // Click Confirm
    await page.getByRole('button', { name: 'Review & Confirm' }).click();

    // Check Dialog
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByText('Confirm Withdrawal')).toBeVisible();

    // Submit
    await page.getByRole('button', { name: 'Confirm Withdrawal' }).click();

    // Success Toast/Message
    await expect(page.getByText('Withdrawal initiated successfully')).toBeVisible();
  });
});
