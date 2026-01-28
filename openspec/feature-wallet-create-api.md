# Wallet Create API Integration Proposal

## 1. Overview

Integrate the wallet creation API endpoint `POST /api/v1/wallet/create` into the backend Go code. The API should accept userId, productCode, and currency parameters and return wallet creation status.

## 2. Architecture

### 2.1 Backend (Go)
- **Handlers:** `internal/handlers/wallet_handler.go`
- **Services:** `internal/services/wallet.go`
- **Routes:** `internal/routes/routes.go`
- **DTOs:** Create new DTO for wallet creation request/response

### 2.2 Frontend API Proxy
- **File:** `api/[...route].ts`
- **Route:** `POST /v1/wallet/create` → backend `/api/wallet/create`

## 3. API Specification

### Request
```json
{
  "userId": "test00001",
  "productCode": "C_SPOT",
  "currency": "USDT_ERC20"
}
```

### Response
```json
{
  "code": 200,
  "message": "成功",
  "data": {
    "userId": "test00001",
    "productCode": "C_SPOT",
    "currency": "USDT_ERC20",
    "status": "NORMAL",
    "createdAt": "2026-01-27 06:49:21"
  },
  "success": true,
  "timestamp": 1769496561055
}
```

## 4. Implementation Plan (TDD)

### Phase 1: DTOs
1. Create wallet DTOs in `internal/dto/wallet.go`

### Phase 2: Service Layer
1. Update `WalletService.CreateWallet` to accept productCode and currency
2. Handle wallet creation with product/currency parameters
3. Return proper response format

### Phase 3: Handler Layer
1. Update `CreateWallet` handler to parse request body
2. Call service with correct parameters
3. Return standardized API response

### Phase 4: Route Configuration
1. Add route to Go backend `internal/routes/routes.go`
2. Add route to API proxy `api/[...route].ts`

### Phase 5: Testing
1. Write unit tests for service layer
2. Write integration tests for handler layer
3. Ensure 100% test coverage

## 5. Risk Assessment

- **Low Risk**: Existing wallet infrastructure is in place
- **Isolation**: Changes limited to wallet-related files
- **Rollback**: Git revert if issues arise

## 6. Files to Modify

1. `internal/dto/wallet.go` - New file
2. `internal/services/wallet.go` - Update CreateWallet signature
3. `internal/handlers/wallet_handler.go` - Update handler
4. `internal/routes/routes.go` - Add route
5. `api/[...route].ts` - Add route config
6. `internal/services/wallet_test.go` - Add tests
7. `internal/handlers/handlers_test.go` - Add tests
