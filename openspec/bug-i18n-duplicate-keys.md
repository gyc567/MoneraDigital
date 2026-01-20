# OpenSpec Bug Proposal: i18n Duplicate Keys in Translation Files

## Bug Summary

**Bug ID**: BUG-2024-I18N-005  
**Severity**: High  
**Priority**: P1 (High)  
**Status**: Proposed  
**Created**: 2026-01-20  
**Component**: Frontend / i18n / Translation Files

---

## Problem Statement

### Current State

During a comprehensive audit of the i18n translation files (`src/i18n/locales/en.json` and `zh.json`), duplicate top-level keys were discovered. When Python parses JSON files (used by the backend loader), **only the first occurrence of a duplicate key is kept**, silently discarding subsequent definitions.

This causes translation keys to be missing at runtime, displaying raw keys instead of translated text.

### Affected Translation Files

| File | Issue |
|------|-------|
| `src/i18n/locales/en.json` | 4 duplicate top-level keys |
| `src/i18n/locales/zh.json` | 4 duplicate top-level keys |

---

## Root Cause Analysis

### Duplicate Keys Found

| Key | Occurrences | First Definition | Second Definition (Lost) |
|-----|-------------|------------------|--------------------------|
| `auth` | 2 | `header.nav.auth` (line 155) | Top-level `auth` with `login`, `register`, `errors`, `2fa` (line 5528) |
| `buttons` | 2 | Empty `buttons: {}` (somewhere) | `cta.buttons` with `consultation`, `documentation` |
| `nav` | 2 | `header.nav` with `products`, `solutions`, etc. | `dashboard.nav` with `overview`, `assets`, etc. |
| `addresses` | 2 | Top-level `addresses` | `dashboard.addresses` with different content |

### Why This Is a Problem

When Python's `json.load()` parses a JSON file with duplicate keys:
```python
# Python only keeps the FIRST value
>>> import json
>>> json.loads('{"a": 1, "a": 2}')
{'a': 1}  # Second "a" is silently discarded!
```

### Impact

1. **Login Page**: `auth.login.*` keys were missing → displayed raw keys like `auth.login.title`
2. **Dashboard Navigation**: Could potentially miss `dashboard.nav.*` translations
3. **Code Maintainability**: Silent failures are hard to debug

---

## Affected Files

| File | Change Type | Description |
|------|-------------|-------------|
| `src/i18n/locales/en.json` | Fix | Consolidate duplicate keys into single definitions |
| `src/i18n/locales/zh.json` | Fix | Consolidate duplicate keys into single definitions |

---

## Proposed Solution

### Fix Strategy

For each duplicate key, consolidate all content into a single definition:

#### 1. Fix `auth` key

**Current (broken)**:
```json
{
  "header": {
    "nav": {
      "auth": { "signIn": "Sign In", "getStarted": "Get Started" }
    }
  },
  "auth": { "2fa": { ... } }  // LOST: login, register, errors
}
```

**Fixed**:
```json
{
  "header": {
    "nav": {
      "auth": { "signIn": "Sign In", "getStarted": "Get Started" }
    }
  },
  "auth": {
    "login": { ... },
    "register": { ... },
    "errors": { ... },
    "2fa": { ... }
  }
}
```

#### 2. Fix `buttons` key

Remove duplicate `buttons: {}` definitions and keep the ones under `hero` and `cta`.

#### 3. Fix `nav` key

Remove duplicate top-level `nav: {}` and keep the nested ones under `header.nav` and `dashboard.nav`.

#### 4. Fix `addresses` key

Verify both `addresses` and `dashboard.addresses` are preserved as they contain different content for different contexts.

---

## Verification Steps

### Step 1: Validate JSON Structure
```bash
# Check for duplicate top-level keys
grep -n '"auth":' src/i18n/locales/en.json
grep -n '"buttons":' src/i18n/locales/en.json
grep -n '"nav":' src/i18n/locales/en.json
grep -n '"addresses":' src/i18n/locales/en.json
```

### Step 2: Verify All Translation Keys Exist
```bash
python3 << 'EOF'
import json
from pathlib import Path

# Collect all t() calls from pages
import re
used_keys = set()
for page in Path('src/pages').rglob('*.tsx'):
    if 'test' in page.name:
        continue
    content = page.read_text()
    matches = re.findall(r't\(\s*["\']([^"\']+)["\']\s*\)', content)
    for m in matches:
        if not m.startswith('/') and m != '0':
            used_keys.add(m)

# Load and check
with open('src/i18n/locales/en.json') as f:
    data = json.load(f)

def exists(data, key):
    keys = key.split('.')
    cur = data
    for k in keys:
        if k in cur:
            cur = cur[k]
        else:
            return False
    return True

missing = [k for k in used_keys if not exists(data, k)]
if missing:
    print(f"Missing: {missing}")
else:
    print("All keys exist!")
EOF
```

### Step 3: Browser Verification
1. Navigate to `/login` - verify "Login" displays, not `auth.login.title`
2. Navigate to `/register` - verify all form labels display correctly
3. Navigate to `/dashboard` - verify navigation items display correctly
4. Test language switcher - both EN and ZH should work

---

## Testing Strategy

### Automated Testing
```bash
# Use agent-browser to verify all pages
agent-browser open https://www.moneradigital.com/login
agent-browser open https://www.moneradigital.com/register
agent-browser open https://www.moneradigital.com/dashboard
```

### Manual Testing Checklist
- [ ] Login page shows translated text
- [ ] Register page shows translated text
- [ ] Dashboard navigation shows translated text
- [ ] Language switcher works correctly
- [ ] No console warnings about missing i18n keys

---

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| JSON syntax error during edit | High | Validate with `python -m json.tool` after changes |
| Accidentally removing valid keys | Medium | Review each change carefully |
| Missing keys in production | High | Run verification script before deployment |

---

## Related Documents

- `src/i18n/locales/en.json` - English translations
- `src/i18n/locales/zh.json` - Chinese translations
- `src/pages/Login.tsx` - Login page (affected)
- `src/pages/Register.tsx` - Register page (affected)
- `openspec/bug-missing-login-i18n-keys.md` - Previous related bug

---

## References

- [Python json module behavior with duplicate keys](https://docs.python.org/3/library/json.html)
- [react-i18next](https://react.i18next.com/)
- [i18next JSON Format](https://www.i18next.com/misc/json-format)
