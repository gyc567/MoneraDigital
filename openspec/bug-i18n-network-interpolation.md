# OpenSpec Bug Proposal: i18n Interpolation Variable Not Working

## Bug Summary

**Bug ID**: BUG-2024-I18N-003  
**Severity**: Medium  
**Priority**: High  
**Status**: Proposed  
**Created**: 2026-01-20  
**Component**: Frontend / i18n

---

## Problem Statement

### Current State
On the Deposit page (`/dashboard/deposit`), the warning text displays the literal variable placeholder `{network}` instead of the actual network value.

**Displayed Text**:
```
请务必确认您选择的是 {network} 网络。充值到错误的网络将无法找回。
```

### Expected State
The warning text should display the actual network value (e.g., "TRON", "Ethereum"):

**Expected Text**:
```
请务必确认您选择的是 TRON 网络。充值到错误的网络将无法找回。
```

Or in English:
```
Ensure you are sending via TRON network. Sending to the wrong network may result in loss of funds.
```

---

## Root Cause Analysis

### Issue Location
**File**: `src/pages/dashboard/Deposit.tsx`  
**Line**: 152

**Current Code**:
```typescript
<p>{t("deposit.warning", { network })}</p>
```

**Translation File** (`src/i18n/locales/en.json` and `zh.json`):
```json
{
  "deposit": {
    "warning": "Ensure you are sending via {network} network. Sending to the wrong network may result in loss of funds."
  }
}
```

### Root Cause
The shorthand object property syntax `{ network }` may not be properly expanding to `{ network: network }` in the current context. While ES6 shorthand property syntax is standard, i18next interpolation may require explicit variable passing.

---

## Affected Files

| File | Change Type | Description |
|------|-------------|-------------|
| `src/pages/dashboard/Deposit.tsx` | Modify | Fix i18n interpolation syntax |
| `src/i18n/locales/en.json` | No change | Translation already correct |
| `src/i18n/locales/zh.json` | No change | Translation already correct |

---

## Proposed Solution

### Fix: Use Explicit Variable Passing

**Before**:
```typescript
<p>{t("deposit.warning", { network })}</p>
```

**After**:
```typescript
<p>{t("deposit.warning", { network: network })}</p>
```

### Why This Fix Works
1. **Explicit syntax**: Using `{ network: network }` ensures the variable is correctly passed to i18next
2. **No behavior change**: The component already has `const [network, setNetwork] = useState("TRON");` defined
3. **Backward compatible**: No changes needed to translation files

---

## Verification Steps

### Step 1: Verify Variable Exists
```typescript
// In Deposit.tsx line 17
const [network, setNetwork] = useState("TRON");
```
**Status**: ✅ Variable exists

### Step 2: Verify Translation Key
```json
// In en.json and zh.json
"deposit": {
  "warning": "Ensure you are sending via {network} network..."
}
```
**Status**: ✅ Key exists with correct interpolation

### Step 3: Verify Component Usage
```typescript
// Before fix
<p>{t("deposit.warning", { network })}</p>

// After fix
<p>{t("deposit.warning", { network: network })}</p>
```

---

## Testing Strategy

### Manual Testing
1. Navigate to `/dashboard/deposit`
2. Select a network (e.g., TRON, Ethereum)
3. Verify the warning text shows the actual network name instead of `{network}`

### Automated Testing
```bash
# Use Playwright to verify text content
agent-browser open https://www.moneradigital.com/dashboard/deposit
agent-browser snapshot -i
# Check for "TRON" or "Ethereum" instead of "{network}"
```

---

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Incorrect interpolation | Medium | Use explicit variable syntax |
| Variable scope issue | Low | Component state is properly defined |
| Translation file error | Low | Files validated and correct |

---

## Related Documents

- `src/pages/dashboard/Deposit.tsx` - Deposit page component
- `src/i18n/locales/en.json` - English translations
- `src/i18n/locales/zh.json` - Chinese translations
- [react-i18next Interpolation](https://react.i18next.com/latest/trans-component#interpolation)

---

## References

- [ES6 Shorthand Property Names](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/Object_initializer)
- [i18next Interpolation](https://www.i18next.com/translation-function/interpolation)
