# OpenSpec Bug Proposal: Missing Login Page i18n Translation Keys

## Bug Summary

**Bug ID**: BUG-2024-I18N-004  
**Severity**: Critical  
**Priority**: P0 (Immediate)  
**Status**: Proposed  
**Created**: 2026-01-20  
**Component**: Frontend / i18n / Translation Files

---

## Problem Statement

### Current State
On the Login page (`/login`), the page displays translation keys instead of actual translated text:

- **Displayed**: `auth.login.title`
- **Displayed**: `auth.login.description`

### Expected State
The page should display:
- **Expected**: "Login" or "登录"
- **Expected**: "Enter your email and password to access your account." or Chinese equivalent

### Screenshots
```
Current (Wrong):
┌─────────────────────────────────────┐
│ auth.login.title                    │  ← Should be "Login"
│ auth.login.description              │  ← Should be description text
└─────────────────────────────────────┘

Expected (Correct):
┌─────────────────────────────────────┐
│ Login                               │
│ Enter your email and password...    │
└─────────────────────────────────────┘
```

---

## Root Cause Analysis

### Issue Location

**Translation File**: `src/i18n/locales/en.json` and `zh.json`

**Current State**:
```json
{
  "header": {
    "auth": {
      "signIn": "Sign In",
      "getStarted": "Get Started"
    }
  },
  "auth": {
    "2fa": { ... }
  }
}
```

**Missing Keys**:
```json
{
  "auth": {
    "login": {
      "title": "Login",
      "description": "Enter your email and password to access your account.",
      "email": "Email",
      "emailPlaceholder": "m@example.com",
      "password": "Password",
      "button": "Login",
      "loggingIn": "Logging in...",
      "noAccount": "Don't have an account?",
      "register": "Register",
      "successMessage": "Logged in successfully",
      "logout": "Logout"
    }
  }
}
```

### Code Using Missing Keys

**File**: `src/pages/Login.tsx`

```typescript
// Line 130
<CardTitle>{requires2FA ? t("dashboard.security.twoFactor") : t("auth.login.title")}</CardTitle>

// Line 132
<CardDescription>
  {requires2FA ? t("dashboard.security.enterCode") : t("auth.login.description")}
</CardDescription>

// Line 141
<Label htmlFor="email">{t("auth.login.email")}</Label>

// Line 146
placeholder={t("auth.login.emailPlaceholder")}

// Line 168
<Label htmlFor="password">{t("auth.login.password")}</Label>
```

---

## Affected Files

| File | Change Type | Description |
|------|-------------|-------------|
| `src/i18n/locales/en.json` | Add | Add missing `auth.login.*` translation keys |
| `src/i18n/locales/zh.json` | Add | Add missing `auth.login.*` translation keys |
| `src/pages/Login.tsx` | No change | Already correctly using i18n keys |

---

## Proposed Solution

### Add Missing Translation Keys

#### File: `src/i18n/locales/en.json`

Add after line 151 (`"auth": {`):

```json
    "login": {
      "title": "Login",
      "description": "Enter your email and password to access your account.",
      "email": "Email",
      "emailPlaceholder": "m@example.com",
      "password": "Password",
      "button": "Login",
      "loggingIn": "Logging in...",
      "noAccount": "Don't have an account?",
      "register": "Register",
      "successMessage": "Logged in successfully",
      "logout": "Logout"
    },
```

#### File: `src/i18n/locales/zh.json`

Add after the corresponding line:

```json
    "login": {
      "title": "登录",
      "description": "请输入您的邮箱和密码以访问您的账户。",
      "email": "邮箱",
      "emailPlaceholder": "m@example.com",
      "password": "密码",
      "button": "登录",
      "loggingIn": "登录中...",
      "noAccount": "还没有账户？",
      "register": "注册",
      "successMessage": "登录成功",
      "logout": "退出"
    },
```

---

## Verification Steps

### Step 1: Verify Translation Files
```bash
# Check if keys exist in translation files
grep -n "auth.login.title" src/i18n/locales/en.json
grep -n "auth.login.title" src/i18n/locales/zh.json
```

### Step 2: Verify Login Page
1. Navigate to `/login`
2. Verify "Login" title displays
3. Verify description text displays
4. Verify all form labels display correctly

### Step 3: Test Language Switching
1. Switch to Chinese language
2. Verify all text displays in Chinese
3. Switch back to English
4. Verify all text displays in English

---

## Testing Strategy

### Automated Testing
```bash
# Use agent-browser to verify page content
agent-browser open https://www.moneradigital.com/login
agent-browser snapshot -i
# Check for "Login" text instead of "auth.login.title"
```

### Manual Testing
1. Open browser DevTools
2. Navigate to `/login`
3. Check console for i18n missing key warnings
4. Verify all text elements display correctly

---

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Missing keys after deployment | High | Add comprehensive key verification |
| Incorrect translations | Medium | Native speaker review |
| Duplicate keys | Low | JSON validation before commit |

---

## Related Documents

- `src/pages/Login.tsx` - Login page component
- `src/i18n/locales/en.json` - English translations
- `src/i18n/locales/zh.json` - Chinese translations
- `src/i18n/config.ts` - i18n configuration

---

## References

- [react-i18next](https://react.i18next.com/)
- [i18next JSON Format](https://www.i18next.com/misc/json-format)
