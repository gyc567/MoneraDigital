/**
 * 2FA API Endpoint Test
 * Tests the 2FA setup API endpoint after the fix
 */

import { chromium } from 'playwright';

const BASE_URL = process.env.TEST_URL || 'https://www.moneradigital.com';

async function test2FAAPIEndpoint() {
  console.log('🧪 Starting 2FA API Endpoint Test...\n');
  console.log(`🌐 Testing URL: ${BASE_URL}\n`);

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();

  const consoleErrors = [];
  const networkErrors = [];

  page.on('console', msg => {
    if (msg.type() === 'error') {
      consoleErrors.push(msg.text());
    }
  });

  page.on('pageerror', error => {
    consoleErrors.push(error.message);
  });

  page.on('response', response => {
    if (response.status() >= 400) {
      networkErrors.push({
        url: response.url(),
        status: response.status()
      });
    }
  });

  try {
    // Test 1: Check if API returns 401 without auth (expected behavior)
    console.log('📍 Test 1: Checking /api/auth/2fa/setup endpoint without auth');
    
    const response = await page.evaluate(async (url) => {
      try {
        const res = await fetch(`${url}/api/auth/2fa/setup`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
        });
        return {
          status: res.status,
          body: await res.text().catch(() => 'Unable to read')
        };
      } catch (error) {
        return { error: error.message };
      }
    }, BASE_URL);

    console.log(`   Response Status: ${response.status}`);
    console.log(`   Response Body: ${response.body?.substring(0, 150)}...`);
    
    const isCorrect401 = response.status === 401 && response.body.includes('AUTH_REQUIRED');
    console.log(`   ✅ Returns 401 AUTH_REQUIRED: ${isCorrect401}`);

    // Test 2: Navigate to security page
    console.log('\n📍 Test 2: Navigating to security page');
    await page.goto(`${BASE_URL}/dashboard/security`, { waitUntil: 'domcontentloaded', timeout: 30000 });
    await page.waitForTimeout(2000);

    // Check if we get redirected to login
    const url = page.url();
    console.log(`   Current URL: ${url}`);
    
    const isOnLogin = url.includes('/login') || url.includes('/auth');
    const isOnSecurity = url.includes('/dashboard/security');
    
    console.log(`   On Login Page: ${isOnLogin}`);
    console.log(`   On Security Page: ${isOnSecurity}`);

    // Test 3: Check for old error patterns
    console.log('\n📍 Test 3: Checking for old error patterns');
    const hasOldErrors = consoleErrors.some(err => 
      err.includes('508') || 
      err.includes('Infinite') ||
      err.includes('SyntaxError')
    );
    console.log(`   Has old 508/Infinite errors: ${hasOldErrors}`);
    console.log(`   Total console errors: ${consoleErrors.length}`);

    // Test 4: Verify 405 error is gone
    console.log('\n📍 Test 4: Verifying 405 error is fixed');
    const has405Errors = consoleErrors.some(err => 
      err.includes('405') && err.includes('Method Not Allowed')
    );
    console.log(`   Has 405 errors: ${has405Errors}`);

    // Summary
    console.log('\n' + '='.repeat(60));
    console.log('📊 TEST RESULTS SUMMARY');
    console.log('='.repeat(60));
    console.log(`API Endpoint Status: ${isCorrect401 ? '✅ WORKING (returns 401 auth required)' : '❌ ISSUE'}`);
    console.log(`Page Load: ✅ SUCCESS`);
    console.log(`Old 508 Error: ${hasOldErrors ? '❌ STILL PRESENT' : '✅ FIXED'}`);
    console.log(`405 Error: ${has405Errors ? '❌ STILL PRESENT' : '✅ FIXED'}`);
    console.log(`Console Errors: ${consoleErrors.length}`);
    console.log('='.repeat(60));

    const isFixed = isCorrect401 && !hasOldErrors && !has405Errors;
    console.log(`\n🎯 FINAL VERDICT: ${isFixed ? '✅ 2FA SETUP FIXED!' : '❌ Issues remain'}`);

    if (consoleErrors.length > 0) {
      console.log('\n📋 Console Errors Found:');
      consoleErrors.slice(0, 5).forEach((err, i) => console.log(`   ${i + 1}. ${err.substring(0, 100)}...`));
    }

    return { success: isFixed, consoleErrors, isCorrect401 };

  } catch (error) {
    console.error('❌ Test failed:', error.message);
    return { success: false, error: error.message };
  } finally {
    await browser.close();
  }
}

// Run test
test2FAAPIEndpoint()
  .then(result => {
    console.log('\n✅ Test execution completed');
    process.exit(result.success ? 0 : 1);
  })
  .catch(error => {
    console.error('❌ Fatal error:', error);
    process.exit(1);
  });
