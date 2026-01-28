import { test, expect } from '@playwright/test';

test('diagnose 2FA skip flow', async ({ page }) => {
  // Listen to console logs
  page.on('console', msg => {
    console.log(`[Console ${msg.type()}] ${msg.text()}`);
  });

  // Listen to network requests
  page.on('request', request => {
    if (request.url().includes('/api/')) {
      console.log(`[Request] ${request.method()} ${request.url()}`);
      const postData = request.postData();
      if (postData) {
        console.log(`[Request Body] ${postData}`);
      }
    }
  });

  page.on('response', async response => {
    if (response.url().includes('/api/')) {
      console.log(`[Response] ${response.status()} ${response.url()}`);
      try {
        const body = await response.text();
        console.log(`[Response Body] ${body.substring(0, 500)}`);
      } catch (e) {
        console.log('[Response Body] [could not read]');
      }
    }
  });

  // Navigate to login page
  console.log('[Test] Navigating to login page...');
  await page.goto('https://www.moneradigital.com/login');
  
  // Wait for page to load
  await page.waitForSelector('input[type="email"]');
  
  // Take screenshot
  await page.screenshot({ path: '/tmp/01-login-page.png' });
  console.log('[Test] Screenshot saved: /tmp/01-login-page.png');
  
  // Fill in login form with test user (no 2FA)
  console.log('[Test] Filling login form...');
  await page.fill('input[type="email"]', 'test-1767941919811@example.com');
  await page.fill('input[type="password"]', 'password123');
  
  // Click login button
  console.log('[Test] Clicking login button...');
  await page.click('button[type="submit"]');
  
  // Wait for navigation or response
  await page.waitForTimeout(3000);
  
  // Take screenshot after login attempt
  await page.screenshot({ path: '/tmp/02-after-login.png' });
  console.log('[Test] Screenshot saved: /tmp/02-after-login.png');
  
  // Check if we're on 2FA page
  const url = page.url();
  console.log(`[Test] Current URL: ${url}`);
  
  if (url.includes('/login')) {
    // Check for 2FA input
    const has2FAInput = await page.locator('input[inputmode="numeric"]').count() > 0;
    console.log(`[Test] Has 2FA input: ${has2FAInput}`);
    
    if (has2FAInput) {
      // Look for Skip button
      const skipButton = await page.locator('button:has-text("Skip")');
      const skipButtonCount = await skipButton.count();
      console.log(`[Test] Skip button count: ${skipButtonCount}`);
      
      if (skipButtonCount > 0) {
        console.log('[Test] Clicking Skip button...');
        await skipButton.click();
        await page.waitForTimeout(3000);
        
        // Take screenshot after clicking skip
        await page.screenshot({ path: '/tmp/03-after-skip.png' });
        console.log('[Test] Screenshot saved: /tmp/03-after-skip.png');
      }
    }
  }
  
  console.log('[Test] Test completed');
});
