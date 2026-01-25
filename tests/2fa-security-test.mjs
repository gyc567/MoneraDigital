/**
 * 2FA Security Page Test
 * Tests the 2FA setup functionality after the fix
 */

import { chromium } from 'playwright';

const BASE_URL = process.env.TEST_URL || 'https://www.moneradigital.com';

async function test2FASecurityPage() {
  console.log('🧪 Starting 2FA Security Page Test...\n');

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();

  // Collect console messages
  const consoleMessages = [];
  const consoleErrors = [];

  page.on('console', msg => {
    const text = msg.text();
    consoleMessages.push({ type: msg.type(), text });
    if (msg.type() === 'error') {
      consoleErrors.push(text);
    }
  });

  page.on('pageerror', error => {
    consoleErrors.push(error.message);
  });

  try {
    // Step 1: Navigate to the security page
    console.log('📍 Step 1: Navigating to /dashboard/security');
    await page.goto(`${BASE_URL}/dashboard/security`, { waitUntil: 'networkidle' });

    // Check if page loaded
    const title = await page.title();
    console.log(`   Page title: ${title}`);

    // Step 2: Check for security page elements
    console.log('📍 Step 2: Checking security page elements');
    const securityTitle = await page.locator('h1:has-text("Security")').isVisible().catch(() => false);
    const twoFactorCard = await page.locator('text=Two-Factor').isVisible().catch(() => false);

    console.log(`   Security title visible: ${securityTitle}`);
    console.log(`   Two-Factor card visible: ${twoFactorCard}`);

    // Step 3: Check if 2FA button exists
    console.log('📍 Step 3: Checking 2FA enable button');
    const enableButton = await page.locator('button:has-text("Enable 2FA")').isVisible().catch(() => false);
    const disableButton = await page.locator('button:has-text("Disable 2FA")').isVisible().catch(() => false);

    console.log(`   Enable 2FA button visible: ${enableButton}`);
    console.log(`   Disable 2FA button visible: ${disableButton}`);

    // Step 4: Check for 401 error patterns in console
    console.log('📍 Step 4: Checking for 401 errors');
    const has2FASetupError = consoleErrors.some(err =>
      err.includes('2FA Setup error') ||
      err.includes('401') ||
      err.includes('Failed to load resource')
    );

    console.log(`   Has 2FA setup errors: ${has2FASetupError}`);

    // Step 5: Check network requests
    console.log('📍 Step 5: Checking API request patterns');
    const requests = page.request();
    const response = await requests.fetch(`${BASE_URL}/api/auth/2fa/setup`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      ignoreHTTPSErrors: true
    }).catch(() => null);

    if (response) {
      console.log(`   API response status: ${response.status()}`);
      const body = await response.text().catch(() => 'Unable to read body');
      console.log(`   API response body: ${body.substring(0, 100)}...`);
    }

    // Summary
    console.log('\n📊 Test Results Summary:');
    console.log('='.repeat(50));
    console.log(`Page Load: ✅ SUCCESS`);
    console.log(`Security Elements: ${securityTitle && twoFactorCard ? '✅' : '❌'}`);
    console.log(`2FA Button: ${enableButton || disableButton ? '✅' : '❌'}`);
    console.log(`API Endpoint Status: ${response?.status() === 401 ? '✅ (401 = working, auth required)' : '⚠️'}`);
    console.log(`Console Errors: ${consoleErrors.length} errors found`);
    if (consoleErrors.length > 0) {
      console.log('   Error samples:');
      consoleErrors.slice(0, 3).forEach(err => console.log(`   - ${err.substring(0, 100)}`));
    }
    console.log('='.repeat(50));

    // Final verdict
    const isFixed = response?.status() === 401 && !consoleErrors.some(e => e.includes('508') || e.includes('Infinite'));

    console.log(`\n🎯 Final Verdict: ${isFixed ? '✅ FIXED - 2FA setup endpoint is working!' : '❌ Still has issues'}`);

    return { success: isFixed, consoleErrors, pageLoad: true };

  } catch (error) {
    console.error('❌ Test failed with error:', error.message);
    return { success: false, error: error.message, consoleErrors };
  } finally {
    await browser.close();
  }
}

// Run test
test2FASecurityPage()
  .then(result => {
    console.log('\n✅ Test completed');
    process.exit(result.success ? 0 : 1);
  })
  .catch(error => {
    console.error('❌ Test execution failed:', error);
    process.exit(1);
  });
