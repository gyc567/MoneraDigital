import { test } from '@playwright/test';

test.describe('Account Opening Debug Logging', () => {
  test('should capture debug logs when creating wallet', async ({ page }) => {
    const consoleLogs: string[] = [];

    page.on('console', msg => {
      const text = msg.text();
      consoleLogs.push(`[${msg.type()}] ${text}`);
    });

    await page.goto('https://www.moneradigital.com/dashboard/account-opening');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    console.log('=== ALL CONSOLE LOGS ===');
    consoleLogs.forEach(log => console.log(log));

    const debugLogs = consoleLogs.filter(log => log.includes('[DEBUG-ACCOUNT-OPENING]'));
    console.log('\n=== DEBUG LOGS ===');
    if (debugLogs.length > 0) {
      console.log('Found', debugLogs.length, 'debug logs:');
      debugLogs.forEach(log => console.log(log));
    } else {
      console.log('No [DEBUG-ACCOUNT-OPENING] logs (expected - no wallet creation triggered)');
    }

    await page.screenshot({ path: '/tmp/account-opening-page.png' });
    console.log('\nScreenshot saved to /tmp/account-opening-page.png');
    console.log('Page title:', await page.title());
  });
});
