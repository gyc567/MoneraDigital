# OpenSpec Bug Proposal: i18n Translation Keys Not Working

## Bug Summary

**Bug ID**: BUG-2024-I18N-002  
**Severity**: Critical  
**Priority**: P0 (Immediate)  
**Status**: Proposed  
**Created**: 2026-01-20  
**Component**: Frontend / i18n

---

## Problem Statement

### Current State
The Withdraw page displays translation keys (e.g., `withdraw.title`, `withdraw.tab.new`, `withdraw.tab.history`, `withdraw.address.empty`, `withdraw.address.addButton`) instead of their corresponding translated values.

### Expected State
The page should display:
- "Withdraw Funds" instead of `withdraw.title`
- "New Withdrawal" instead of `withdraw.tab.new`
- "History" instead of `withdraw.tab.history`
- "No verified addresses available..." instead of `withdraw.address.empty`
- "Add Withdrawal Address" instead of `withdraw.address.addButton`

### Screenshots
```
Current:  withdraw.title
Expected: Withdraw Funds

Current:  withdraw.tab.new
Expected: New Withdrawal

Current:  withdraw.address.empty
Expected: No verified addresses available. Please add a withdrawal address first.
```

---

## Root Cause Analysis

### Potential Causes

#### 1. Translation Files Not Properly Imported
- **File**: `src/i18n/config.ts`
- **Issue**: JSON files might not be importing correctly

#### 2. i18n Initialization Order
- **File**: `src/App.tsx`
- **Issue**: i18n might be initialized after components render

#### 3. Build Cache Issue
- **Issue**: Vercel build cache might be serving stale translation files

#### 4. Missing Translation Keys
- **Issue**: Some keys might not exist in the translation files

---

## Affected Files

| File | Change Type | Description |
|------|-------------|-------------|
| `src/i18n/config.ts` | Investigate | Check import mechanism |
| `src/i18n/locales/en.json` | Verify | Verify all keys exist |
| `src/i18n/locales/zh.json` | Verify | Verify all keys exist |
| `src/App.tsx` | Verify | Check i18n initialization |
| `src/pages/dashboard/Withdraw.tsx` | Verify | Verify t() usage |

---

## Verification Steps

### Step 1: Verify Translation Files
```bash
# Check JSON syntax
python3 -c "import json; json.load(open('src/i18n/locales/en.json'))"
python3 -c "import json; json.load(open('src/i18n/locales/zh.json'))"

# Check specific keys exist
python3 -c "
import json
with open('src/i18n/locales/en.json') as f:
    data = json.load(f)
    keys = [
        'dashboard.withdraw.title',
        'dashboard.withdraw.tab.new',
        'dashboard.withdraw.tab.history',
        'dashboard.withdraw.address.empty',
        'dashboard.withdraw.address.addButton',
    ]
    for key in keys:
        keys = key.split('.')
        value = data
        for k in keys:
            value = value.get(k, 'NOT FOUND')
        print(f'{key}: {value}')
"
```

### Step 2: Check i18n Configuration
- Verify `src/i18n/config.ts` properly imports JSON files
- Verify `src/App.tsx` wraps app with `I18nextProvider`

### Step 3: Force Clean Build
```bash
# Clear Vercel cache by redeploying
bash scripts/deploy.sh
```

---

## Proposed Solution

### Option 1: Clear Build Cache (Quick Fix)
Delete the deployed version and redeploy with fresh build cache.

### Option 2: Verify i18n Configuration (Recommended)
1. Add console.log to verify translations are loaded
2. Add Suspense wrapper for async loading
3. Verify fallbackLng is set correctly

### Option 3: Use Vite Environment (Development)
```bash
npm run dev
# Test locally to see if i18n works
```

---

## Testing Strategy

### Manual Testing
1. Navigate to `/dashboard/withdraw`
2. Verify page title displays "Withdraw Funds" (English) or "提现" (Chinese)
3. Verify tab labels display correctly
4. Verify all dialog text displays correctly

### Automated Testing
```bash
# Use Playwright to verify text content
agent-browser open https://www.moneradigital.com/dashboard/withdraw
agent-browser snapshot -i
# Check for key strings instead of key names
```

---

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Stale build cache | High | Force clean deployment |
| JSON syntax error | High | Validate JSON files |
| i18n init timing | Medium | Add Suspense wrapper |
| Missing keys | High | Comprehensive key verification |

---

## Related Documents

- `src/i18n/config.ts` - i18n configuration
- `src/i18n/locales/en.json` - English translations
- `src/i18n/locales/zh.json` - Chinese translations
- `src/pages/dashboard/Withdraw.tsx` - Withdraw page component
- `src/App.tsx` - Main app component with i18n provider

---

## References

- [react-i18next Documentation](https://react.i18next.com/)
- [i18next Configuration](https://www.i18next.com/overview/configuration-options)
