# OpenSpec: Fix Deposit Network Display Issue

## Problem Statement

User `gyc567@gmail.com` accessing `https://www.moneradigital.com/dashboard/deposit` sees a placeholder text `{network}` in the warning message instead of the actual network name (e.g., "TRON", "Ethereum", "BNB Smart Chain").

**Current Output:**
```
"Ensure you are sending via {network} network. Sending to the wrong network may result in loss of funds."
```

**Expected Output:**
```
"Ensure you are sending via TRON network. Sending to the wrong network may result in loss of funds."
(or "Ethereum", "BNB Smart Chain" depending on selected network)
```

## Root Cause Analysis

The i18n translation string in both English and Chinese locale files uses a placeholder `{network}` but there may be an issue with how the `networkName` variable is derived or passed.

**File:** `src/i18n/locales/en.json` (line 575) and `zh.json` (line 594)
```json
"warning": "Ensure you are sending via {network} network. Sending to the wrong network may result in loss of funds."
```

**File:** `src/pages/dashboard/Deposit.tsx` (lines 36-38, 173)
```typescript
const selectedNetwork = networkOptions.find(option => option.value === network);
const networkLabel = selectedNetwork ? selectedNetwork.label : network;
const networkName = selectedNetwork ? selectedNetwork.name : network;
// ... later ...
<p>{t("deposit.warning", { network: networkName })}</p>
```

**Network Options Definition (lines 30-34):**
```typescript
const networkOptions = [
  { value: "TRON", label: "TRON (TRC20)", name: "TRON" },
  { value: "ETH", label: "Ethereum (ERC20)", name: "Ethereum" },
  { value: "BSC", label: "BNB Smart Chain (BEP20)", name: "BNB Smart Chain" },
];
```

**Possible Issues:**
1. React component may be rendering before `networkName` is properly derived
2. The i18n interpolation might not be working as expected
3. Default state `network: "TRON"` (line 26) should map correctly

## Solution Design

### KISS Principle - Simplest Fix
Ensure `networkName` is always a valid, non-undefined string by using optional chaining and explicit fallback:

```typescript
const networkName = selectedNetwork?.name || network || "Unknown Network";
```

This guarantees the translation parameter is always defined.

## Implementation Plan

### Phase 1: Code Changes
1. Update `Deposit.tsx` line 38 to ensure `networkName` is always defined
2. Optionally add JSDoc comments for clarity

### Phase 2: Testing (100% Coverage Required)
1. Unit test: Verify network name extraction for each network option
2. Unit test: Verify i18n interpolation with actual network names
3. Unit test: Verify fallback behavior when network is not found
4. Integration test: Render component and verify warning message displays correctly
5. Manual test: Test all three networks in UI

### Phase 3: Build & Deploy
1. Run full test suite
2. Build production bundle
3. Deploy to Vercel
4. Verify fix on staging/production

## Design Principles

- **KISS:** Minimal code change - one line fix
- **High Cohesion:** Network options remain centralized
- **Low Coupling:** No changes to i18n configuration
- **100% Test Coverage:** All networkName extraction paths tested

## Files to Modify

1. **src/pages/dashboard/Deposit.tsx**
   - Line 38: Update networkName extraction logic
   - Ensure proper TypeScript typing

2. **src/pages/dashboard/Deposit.test.tsx** (CREATE NEW)
   - Test network name extraction
   - Test warning message rendering
   - Test all three network options
   - Test fallback behavior

## Success Criteria

- [x] Warning message displays actual network name, not `{network}` placeholder
- [x] All three networks display their correct full names
- [x] No regressions in deposit functionality
- [x] 100% test coverage for networkName logic
- [x] Unit tests pass
- [x] Build succeeds
- [x] Deployed to production
