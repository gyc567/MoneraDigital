# OpenSpec Bug Proposal: Withdraw Page i18n Support

## Bug Summary

**Bug ID**: BUG-2024-I18N-001  
**Severity**: Medium  
**Priority**: High  
**Status**: Proposed  
**Created**: 2026-01-20  
**Component**: Frontend / i18n

---

## Problem Statement

### Current State
1. The sidebar menu shows "提币" (literal translation) instead of "提现" (standard terminology)
2. The Withdraw page does not support internationalization (i18n)
3. All UI text is hardcoded in English, not translated for Chinese users

### Expected State
1. Sidebar menu should show "提现" (withdraw) as the standard terminology
2. All text on the Withdraw page should be internationalized
3. Chinese version should display in Simplified Chinese
4. English version should display in English

---

## Root Cause Analysis

### Issue 1: Incorrect Menu Label
- **File**: `src/components/DashboardSidebar.tsx`
- **Problem**: Hardcoded Chinese text "提币" instead of using i18n translation key
- **Impact**: Inconsistent terminology, not following standard financial terminology

### Issue 2: Missing i18n Keys
- **File**: `src/pages/dashboard/Withdraw.tsx`
- **Problem**: All UI text is hardcoded in English
  - "Withdraw Funds"
  - "New Withdrawal"
  - "History"
  - "Select Withdrawal Address"
  - etc.
- **Impact**: Non-English users cannot understand the interface

### Issue 3: Missing Translation Files
- **File**: `src/i18n/locales/zh.json`
- **Problem**: Missing translation keys for Withdraw page
- **Impact**: Even if i18n is implemented, Chinese translation is unavailable

---

## Affected Files

| File | Change Type | Description |
|------|-------------|-------------|
| `src/components/DashboardSidebar.tsx` | Modify | Update menu label to use i18n |
| `src/pages/dashboard/Withdraw.tsx` | Modify | Replace hardcoded text with t() calls |
| `src/i18n/locales/en.json` | Add | Add English translation keys |
| `src/i18n/locales/zh.json` | Add | Add Chinese translation keys |

---

## Proposed Solution

### 1. Update DashboardSidebar.tsx

**Current Code**:
```typescript
{ icon: ArrowUpCircle, label: t("dashboard.nav.withdraw"), href: "/dashboard/withdraw" }
```

**Translation Key** (currently in en.json):
```json
"dashboard": {
  "nav": {
    "withdraw": "提币"
  }
}
```

**Fix**: Change translation key value from "提币" to "提现"

### 2. Update Withdraw.tsx - Replace Hardcoded Text

**List of text to replace**:

| Current Text | Translation Key | Description |
|--------------|-----------------|-------------|
| "Withdraw Funds" | `withdraw.title` | Page title |
| "New Withdrawal" | `withdraw.tab.new` | Tab label |
| "History" | `withdraw.tab.history` | Tab label |
| "Select Withdrawal Address" | `withdraw.address.title` | Card title |
| "Choose a verified address to withdraw funds to" | `withdraw.address.description` | Card description |
| "Withdrawal Address" | `withdraw.address.label` | Input label |
| "Withdrawal Chain" | `withdraw.chain.label` | Input label |
| "Withdrawal Amount" | `withdraw.amount.title` | Card title |
| "Enter the amount you wish to withdraw" | `withdraw.amount.description` | Card description |
| "Amount" | `withdraw.amount.label` | Input label |
| "Network Fee (estimated)" | `withdraw.fee.label` | Fee display |
| "You will receive" | `withdraw.receive.label` | Receive display |
| "Review & Confirm" | `withdraw.review.button` | Button text |
| "Confirm Withdrawal" | `withdraw.confirm.button` | Button text |
| "Cancel" | `common.cancel` | Button text |
| "Loading addresses..." | `withdraw.address.loading` | Loading state |
| "No verified addresses available" | `withdraw.address.empty` | Empty state |
| "Loading history..." | `withdraw.history.loading` | Loading state |
| "No withdrawal history yet" | `withdraw.history.empty` | Empty state |

### 3. Add Translation Keys

**src/i18n/locales/en.json**:
```json
{
  "withdraw": {
    "title": "Withdraw Funds",
    "tab": {
      "new": "New Withdrawal",
      "history": "History"
    },
    "address": {
      "title": "Select Withdrawal Address",
      "description": "Choose a verified address to withdraw funds to",
      "label": "Withdrawal Address",
      "loading": "Loading addresses...",
      "empty": "No verified addresses available. Please add a withdrawal address first.",
      "addButton": "Add Withdrawal Address"
    },
    "chain": {
      "label": "Withdrawal Chain"
    },
    "amount": {
      "title": "Withdrawal Amount",
      "description": "Enter the amount you wish to withdraw",
      "label": "Amount"
    },
    "fee": {
      "label": "Network Fee (estimated)"
    },
    "receive": {
      "label": "You will receive"
    },
    "review": {
      "button": "Review & Confirm"
    },
    "confirm": {
      "button": "Confirm Withdrawal",
      "title": "Confirm Withdrawal",
      "description": "Please review the details below before confirming this withdrawal."
    },
    "history": {
      "loading": "Loading history...",
      "empty": "No withdrawal history yet"
    },
    "success": {
      "initiated": "Withdrawal initiated successfully",
      "verified": "Withdrawal verified successfully"
    },
    "error": {
      "failed": "Failed to initiate withdrawal",
      "invalidAddress": "Invalid address format",
      "insufficientBalance": "Insufficient balance"
    }
  }
}
```

**src/i18n/locales/zh.json**:
```json
{
  "withdraw": {
    "title": "提现",
    "tab": {
      "new": "发起提现",
      "history": "历史记录"
    },
    "address": {
      "title": "选择提现地址",
      "description": "选择一个已验证的钱包地址用于提现",
      "label": "提现地址",
      "loading": "正在加载地址...",
      "empty": "暂无可用的已验证地址，请先添加提现地址。",
      "addButton": "添加提现地址"
    },
    "chain": {
      "label": "提现网络"
    },
    "amount": {
      "title": "提现金额",
      "description": "请输入您希望提现的金额",
      "label": "金额"
    },
    "fee": {
      "label": "网络手续费（预估）"
    },
    "receive": {
      "label": "您将收到"
    },
    "review": {
      "button": "确认信息"
    },
    "confirm": {
      "button": "确认提现",
      "title": "确认提现",
      "description": "请在确认前仔细核对以下信息。"
    },
    "history": {
      "loading": "正在加载历史记录...",
      "empty": "暂无提现记录"
    },
    "success": {
      "initiated": "提现申请已提交",
      "verified": "提现已验证成功"
    },
    "error": {
      "failed": "提现申请提交失败",
      "invalidAddress": "地址格式无效",
      "insufficientBalance": "余额不足"
    }
  }
}
```

---

## Implementation Plan

### Phase 1: Translation Files
1. Add all translation keys to `src/i18n/locales/en.json`
2. Add all translation keys to `src/i18n/locales/zh.json`
3. Verify JSON syntax is valid

### Phase 2: Sidebar Menu
1. Update `dashboard.nav.withdraw` value in zh.json from "提币" to "提现"

### Phase 3: Withdraw Page
1. Replace all hardcoded strings with `t()` function calls
2. Verify all text is properly translated
3. Test in both English and Chinese locales

### Phase 4: Testing
1. Verify sidebar menu displays correctly in both languages
2. Verify all text on Withdraw page is translated
3. Test functionality remains intact

---

## Verification Checklist

- [ ] Sidebar shows "提现" instead of "提币" in Chinese
- [ ] All text on Withdraw page is translated to Chinese
- [ ] All text on Withdraw page is in English for English locale
- [ ] No hardcoded text remains on the page
- [ ] Translation keys are properly namespaced
- [ ] Page functionality works correctly
- [ ] No console errors related to missing translations

---

## Testing Strategy

### Manual Testing
1. Switch browser language to Chinese
2. Navigate to Dashboard
3. Verify "提现" appears in sidebar
4. Navigate to Withdraw page
5. Verify all text is in Chinese
6. Switch browser language to English
7. Verify all text is in English

### Automated Testing
- Use Playwright to verify i18n functionality
- Test language switching
- Verify all translation keys are present

---

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Missing translation keys | Medium | Comprehensive testing checklist |
| Incorrect translations | Low | Native speaker review |
| Regression in functionality | Medium | Test all existing functionality |

---

## Related Documents

- `src/i18n/i18n.ts` - i18n configuration
- `src/components/DashboardSidebar.tsx` - Sidebar component
- `src/pages/dashboard/Withdraw.tsx` - Withdraw page

---

## References

- [i18n Best Practices](https://react.i18next.com/)
- [ISO 639-1 Language Codes](https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes)
