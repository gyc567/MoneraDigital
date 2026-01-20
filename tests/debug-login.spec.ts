import { test, expect } from '@playwright/test';

test.describe('Debug Login Error Handling', () => {
  test('should debug login error response', async ({ page }) => {
    // Capture console logs
    const consoleLogs: string[] = [];
    page.on('console', msg => {
      consoleLogs.push(`${msg.type()}: ${msg.text()}`);
    });

    // Navigate to login
    await page.goto('/login');
    
    // Fill in form
    await page.fill('input[type="email"]', 'nonexistent@example.com');
    await page.fill('input[type="password"]', 'Password123!');
    
    // Click submit and wait
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000);
    
    // Print console logs
    console.log('=== Console Logs ===');
    consoleLogs.forEach(log => console.log(log));
    
    // Check page content
    const pageContent = await page.content();
    console.log('=== Page Content Contains Error ===');
    console.log('Has "邮箱输入错误":', pageContent.includes('邮箱输入错误'));
    console.log('Has "Email input error":', pageContent.includes('Email input error'));
    
    // Take screenshot
    await page.screenshot({ path: '/tmp/debug-login.png' });
    console.log('Screenshot saved to /tmp/debug-login.png');
  });
});
