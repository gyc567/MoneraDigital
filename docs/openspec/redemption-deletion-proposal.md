# Redemption Feature Deletion Proposal

**Scope**: Complete removal of redemption feature (frontend + backend) to reduce Vercel Serverless Functions and simplify codebase.

**Design Principles**: KISS - Keep It Simple, Stupid. High cohesion, low coupling.

---

## 1. Overview

### 1.1 Background
- Redemption feature was never fully integrated into the product
- 2 API endpoints consuming Vercel Serverless Function slots
- Feature is isolated with no dependencies on other modules

### 1.2 Goal
- Reduce API count from 8 to 6 (eliminate `/api/redemption`)
- Simplify codebase by removing unused feature
- Zero impact on existing functionality

---

## 2. Files to Delete

### 2.1 Frontend TypeScript (6 files)

| File Path | Type | Lines | Purpose |
|-----------|------|-------|---------|
| `api/redemption.ts` | API | 38 | Vercel serverless endpoint |
| `src/lib/redemption/redemption-service.ts` | Service | ~50 | Business logic |
| `src/lib/redemption/redemption-repository.ts` | Repository | ~30 | Data access |
| `src/lib/redemption/redemption-model.ts` | Model | ~30 | Type definitions |
| `src/lib/redemption/products.ts` | Config | ~30 | Product catalog |
| `src/__tests__/redemption-service.test.ts` | Tests | ~60 | Unit tests |

### 2.2 Backend Go (6 files)

| File Path | Type | Lines | Purpose |
|-----------|------|-------|---------|
| `internal/api/redemption.go` | API | ~50 | HTTP handlers |
| `internal/redemption/service.go` | Service | ~120 | Business logic |
| `internal/redemption/repository.go` | Repository | ~50 | Data access |
| `internal/redemption/model.go` | Model | ~40 | Type definitions |
| `internal/redemption/product.go` | Config | ~40 | Product catalog |
| `internal/redemption/service_test.go` | Tests | ~80 | Unit tests |

### 2.3 Documentation (1 file)

| File Path | Purpose |
|-----------|---------|
| `docs/openspec/redemption-open-spec.md` | OpenSpec documentation |

### 2.4 Total Deletion Count

| Category | Count |
|----------|-------|
| Frontend TypeScript | 6 files |
| Backend Go | 6 files |
| Documentation | 1 file |
| **Total** | **13 files** |

---

## 3. Dependency Analysis

### 3.1 Cross-Module Dependencies

**Imports Check** (verified via grep):
- ❌ No other files import `redemption` package
- ❌ No other files import `RedemptionService`
- ❌ No routes reference redemption endpoints

**Safe to Delete** ✓

### 3.2 Isolation Verification

```
src/lib/redemption/          → Standalone module
  ├── redemption-service.ts  → No external deps
  ├── redemption-repository.ts → No external deps
  ├── redemption-model.ts    → No external deps
  └── products.ts            → No external deps

internal/redemption/         → Standalone package
  ├── service.go             → No external deps (except time)
  ├── repository.go          → No external deps
  ├── model.go               → No external deps
  └── product.go             → No external deps
```

---

## 4. KISS Compliance Checklist

| Principle | Compliance | Notes |
|-----------|------------|-------|
| **Single Responsibility** | ✅ | Each file has one purpose |
| **No Unused Code** | ✅ | 100% unused, safe to delete |
| **No Coupling** | ✅ | Isolated module |
| **Minimal Changes** | ✅ | Pure deletion, no refactoring needed |
| **Reversible** | ✅ | Git can restore if needed |

---

## 5. Implementation Steps

### Phase 1: Backup (Safety First)

```bash
# Create git branch for this operation
git checkout -b feature/delete-redemption

# Tag current state
git tag redemption-backup-$(date +%Y%m%d)
```

### Phase 2: Delete Frontend

```bash
# Remove TypeScript files
rm -rf src/lib/redemption/
rm -f api/redemption.ts
rm -f src/__tests__/redemption-service.test.ts
```

### Phase 3: Delete Backend

```bash
# Remove Go files
rm -rf internal/redemption/
rm -f internal/api/redemption.go
```

### Phase 4: Delete Documentation

```bash
# Remove OpenSpec
rm -f docs/openspec/redemption-open-spec.md
```

### Phase 5: Verification

```bash
# Ensure no orphaned references
grep -r "redemption" src/ --include="*.ts" --include="*.tsx" | grep -v "__tests__" || echo "No references found"

grep -r "redemption" internal/ --include="*.go" | grep -v "redemption/" || echo "No references found"

# Run tests
npm test

# Run build
npm run build
```

---

## 6. Vercel Serverless Function Reduction

### Before Deletion

| Count | Endpoints |
|-------|-----------|
| 8 | `/api/auth/*` (7) + `/api/redemption` (1) |

### After Deletion

| Count | Endpoints |
|-------|-----------|
| 6 | `/api/auth/*` (6) |

**Reduction**: 2 endpoints (25% reduction)

---

## 7. Risk Assessment

| Risk | Level | Mitigation |
|------|-------|------------|
| Accidental deletion of dependent code | Low | Grep verification before deletion |
| Build failure after deletion | Low | Run `npm run build` after deletion |
| Test failure after deletion | Low | Run `npm test` after deletion |
| Rollback complexity | Low | Git tag enables instant rollback |

---

## 8. Rollback Plan

If issues arise:

```bash
# Instant rollback
git checkout HEAD -- src/lib/redemption/
git checkout HEAD -- api/redemption.ts
git checkout HEAD -- internal/redemption/
git checkout HEAD -- internal/api/redemption.go
git checkout HEAD -- docs/openspec/redemption-open-spec.md

# Or full branch revert
git checkout main
git branch -D feature/delete-redemption
```

---

## 9. Acceptance Criteria

- [ ] All 13 redemption files deleted
- [ ] `npm run build` passes without errors
- [ ] `npm test` passes without errors
- [ ] No redemption references in other files
- [ ] Vercel deploy succeeds with ≤12 Serverless Functions

---

## 10. Effort Estimation

| Task | Time |
|------|------|
| Verification and grep analysis | 10 min |
| File deletion | 5 min |
| Build and test verification | 5 min |
| **Total** | **~20 minutes** |

---

**Generated**: 2026-01-22
**Author**: Sisyphus AI Agent
**Status**: Ready for Implementation
