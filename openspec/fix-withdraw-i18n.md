# Fix English Text in Chinese Version of Withdraw Page

## Problem Statement
The Chinese version of the Withdraw page (`src/pages/dashboard/Withdraw.tsx`) contains hardcoded English text that should be translated using the i18n system.

## Current Issues Identified

### Hardcoded English Text in Withdraw.tsx:

1. **Line 456**: `placeholder="Select chain"` - Chain selector placeholder
2. **Line 494**: `placeholder="10.50"` - Amount input placeholder  
3. **Line 673**: `Please enter your 2FA code to confirm this withdrawal.` - 2FA dialog description
4. **Line 687-689**: Various 2FA hint text
5. **Lines 749-753**: Network names in Add Address dialog:
   - "Ethereum"
   - "Arbitrum"
   - "Polygon"
   - "Tron"
   - "Bitcoin"

### Existing Translation Keys in zh.json:
- `"dashboard.withdraw.chain.placeholder": "选择网络"` - already exists but not used
- `"auth.2fa.firstWithdrawal": "这是您首次向新地址提现，请输入验证码进行确认。"` - exists but different text

## Proposed Solution

### 1. Add Missing Translation Keys to zh.json

```json
{
  "dashboard": {
    "withdraw": {
      "chain": {
        "placeholder": "选择网络"
      },
      "amount": {
        "placeholder": "请输入金额"
      },
      "2fa": {
        "withdrawalDescription": "请输入验证码以确认此提现",
        "hint": "请输入6位验证码",
        "network": {
          "ethereum": "以太坊",
          "arbitrum": "Arbitrum",
          "polygon": "Polygon",
          "tron": "Tron",
          "bitcoin": "比特币"
        }
      }
    }
  }
}
```

### 2. Modify Withdraw.tsx

Replace hardcoded text with translation calls:

```typescript
// Line 456: Replace placeholder
<SelectValue placeholder={t("dashboard.withdraw.chain.placeholder")} />

// Line 494: Replace placeholder
placeholder={t("dashboard.withdraw.amount.placeholder")}

// Line 673: Replace 2FA description
{t("dashboard.withdraw.2fa.withdrawalDescription")}

// Lines 687-689: Replace 2FA hint
{t("dashboard.withdraw.2fa.hint")}

// Lines 749-753: Replace network names with translation
<SelectItem value="Ethereum">{t("dashboard.withdraw.2fa.network.ethereum")}</SelectItem>
<SelectItem value="Arbitrum">{t("dashboard.withdraw.2fa.network.arbitrum")}</SelectItem>
<SelectItem value="Polygon">{t("dashboard.withdraw.2fa.network.polygon")}</SelectItem>
<SelectItem value="Tron">{t("dashboard.withdraw.2fa.network.tron")}</SelectItem>
<SelectItem value="Bitcoin">{t("dashboard.withdraw.2fa.network.bitcoin")}</SelectItem>
```

### 3. Add Translation Keys to en.json

```json
{
  "dashboard": {
    "withdraw": {
      "amount": {
        "placeholder": "Enter amount"
      },
      "2fa": {
        "withdrawalDescription": "Please enter your 2FA code to confirm this withdrawal.",
        "hint": "Enter the 6-digit code",
        "network": {
          "ethereum": "Ethereum",
          "arbitrum": "Arbitrum",
          "polygon": "Polygon",
          "tron": "Tron",
          "bitcoin": "Bitcoin"
        }
      }
    }
  }
}
```

## Implementation Plan

### Phase 1: Add Translation Keys
1. Add missing keys to `src/i18n/locales/zh.json`
2. Add corresponding keys to `src/i18n/locales/en.json`

### Phase 2: Update Component
1. Replace hardcoded English text with `t()` calls in `Withdraw.tsx`
2. Ensure all placeholders and static text are translated

### Phase 3: Testing
1. Verify Chinese text displays correctly
2. Verify English text still works
3. Run existing tests

## Files to Modify
1. `src/i18n/locales/zh.json` - Add translation keys
2. `src/i18n/locales/en.json` - Add translation keys  
3. `src/pages/dashboard/Withdraw.tsx` - Replace hardcoded text with translations

## Constraints
- KISS: Keep changes minimal and focused
- High cohesion: Group related translations together
- Low coupling: Don't affect other functionality
- 100% test coverage for new code

## Verification Steps
1. Switch language to Chinese
2. Visit /dashboard/withdraw
3. Verify all text is in Chinese:
   - Chain selector shows "选择网络"
   - Amount input shows "请输入金额"
   - 2FA dialog shows Chinese text
   - Network dropdown shows Chinese network names
4. Switch back to English and verify text is English
5. Run tests: `npm test -- src/pages/dashboard/Withdraw.test.tsx`
