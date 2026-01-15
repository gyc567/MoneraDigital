# Bug Report: Deposit Page i18n Keys Not Resolving

## 1. Issue Description
Users navigating to the Deposit page see raw translation keys (e.g., `deposit.activate`, `deposit.activateDesc`) instead of the localized text.

## 2. Environment
- **URL:** `/dashboard/deposit`
- **State:** New user (Wallet status `NONE`).

## 3. Root Cause Analysis
- **Potential:** JSON syntax error in `en.json` / `zh.json` prevented the translation resource from loading fully.
- **Potential:** Key nesting depth issue (unlikely, i18next supports deep nesting).
- **Verification:** Need to ensure the `deposit` object is correctly placed at the root level of the translation JSON, parallel to `auth`, `dashboard`, etc.

## 4. Proposed Fix
1.  Validate and re-write `src/i18n/locales/en.json` and `zh.json` to ensure strict JSON compliance.
2.  Verify `Deposit.tsx` uses the keys exactly as defined.

## 5. Verification
- Reload page.
- Check if text "Activate Account" appears.
