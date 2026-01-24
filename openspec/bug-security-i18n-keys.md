# Bug Fix: Missing Security Dashboard i18n Keys

## Problem
Users reported missing translation keys on the `/dashboard/security` page when using the Chinese locale. The error messages indicate missing keys like `dashboard.security.passwordDescription`, `dashboard.security.currentPassword`, etc.

## Solution
Add the missing keys to `src/i18n/locales/zh.json` and `src/i18n/locales/en.json`.

### Missing Keys
- `dashboard.security.title`
- `dashboard.security.password`
- `dashboard.security.passwordDescription`
- `dashboard.security.currentPassword`
- `dashboard.security.newPassword`
- `dashboard.security.twoFactor`
- `dashboard.security.twoFactorDescription`
- `dashboard.security.status`
- `dashboard.security.enabled`
- `dashboard.security.disabled`
- `dashboard.security.enable2FA`
- `dashboard.security.disable2FA`
- `dashboard.security.disable2FAConfirm`
- `dashboard.security.enterCode`
- `dashboard.security.verifying`
- `dashboard.security.setup2FA`
- `dashboard.security.scanQR`
- `dashboard.security.saveBackupCodes`
- `dashboard.security.secretKeyManual`
- `dashboard.security.openInApp`
- `dashboard.security.nextBackupCodes`
- `dashboard.security.backupCodeInfo`
- `dashboard.security.verify`
- `dashboard.security.whitelist`
- `dashboard.security.whitelistDescription`
- `dashboard.security.manageAddresses`

## Design Principles
- **KISS:** Directly add the missing keys to the localization files.
- **High Cohesion:** Keep translation keys grouped under `dashboard.security`.

## Testing Plan
1.  **Automated Check:** Create a script to verify that all keys used in `Security.tsx` exist in `en.json` and `zh.json`.
2.  **Manual Verification:** (Implicit) The fix is purely data-entry in JSON files.

## Implementation Steps
1.  Read `src/i18n/locales/zh.json` and `src/i18n/locales/en.json`.
2.  Add the missing keys with appropriate translations.
3.  Create a test script `scripts/verify-security-i18n.js` to ensure keys are present.
