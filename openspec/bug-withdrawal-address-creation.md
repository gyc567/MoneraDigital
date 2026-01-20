# OpenSpec Bug Proposal: Withdrawal Address Creation Flow

## Bug Summary

**Bug ID**: BUG-2024-ADDR-001  
**Severity**: Critical  
**Priority**: P0 (Immediate)  
**Status**: Proposed  
**Created**: 2026-01-20  
**Component**: Backend / Database / API

---

## Problem Statement

### Current State
When a user on the Withdraw page fills in the address form (Address Label, Blockchain Network, Wallet Address) and clicks "Add Withdrawal Address", the frontend sends a request to the backend API `POST /api/addresses`. The backend should save this address to the database table `withdrawal_address_whitelist`.

### Expected Behavior
1. User fills in address form with:
   - Address Label (e.g., "My MetaMask Wallet")
   - Blockchain Network (e.g., "Ethereum")
   - Wallet Address (e.g., "0x742d35Cc6634C0532925a3b844Bc9e7595f2bD51")
2. User clicks "Add Withdrawal Address" button
3. Frontend sends `POST /api/addresses` with JSON body
4. Backend validates the request
5. Backend saves address to `withdrawal_address_whitelist` table
6. Backend returns created address with ID
7. Frontend displays success message and refreshes address list

### Current Issues
1. Need to verify backend API is fully implemented
2. Need to verify database table exists and has correct schema
3. Need to test end-to-end flow

---

## Affected Files

### Backend (Go)
| File | Description |
|------|-------------|
| `internal/handlers/handlers.go` | `AddAddress` handler |
| `internal/services/address.go` | `AddressService.AddAddress` |
| `internal/repository/postgres/address.go` | `AddressRepository.CreateAddress` |
| `internal/models/models.go` | `AddAddressRequest` struct |
| `internal/routes/routes.go` | Route definition `POST /api/addresses` |

### Database
| Table | Description |
|-------|-------------|
| `withdrawal_address_whitelist` | Stores user's withdrawal addresses |

### Frontend
| File | Description |
|------|-------------|
| `src/pages/dashboard/Withdraw.tsx` | Address creation dialog and `handleCreateAddress` function |

---

## Database Schema

### Table: withdrawal_address_whitelist

```sql
CREATE TABLE withdrawal_address_whitelist (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    address_alias VARCHAR(255) NOT NULL,
    chain_type VARCHAR(50) NOT NULL,
    wallet_address VARCHAR(255) NOT NULL,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at TIMESTAMP,
    verification_method VARCHAR(50),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, wallet_address)
);

-- Index for faster lookups
CREATE INDEX idx_withdrawal_address_whitelist_user_id ON withdrawal_address_whitelist(user_id);
CREATE INDEX idx_withdrawal_address_whitelist_wallet_address ON withdrawal_address_whitelist(wallet_address);
```

---

## API Specification

### POST /api/addresses

**Request**:
```json
{
  "wallet_address": "0x742d35Cc6634C0532925a3b844Bc9e7595f2bD51",
  "chain_type": "Ethereum",
  "address_alias": "My MetaMask Wallet"
}
```

**Headers**:
```
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

**Response (Success - 201 Created)**:
```json
{
  "id": 4,
  "user_id": 127,
  "address_alias": "My MetaMask Wallet",
  "chain_type": "Ethereum",
  "wallet_address": "0x742d35Cc6634C0532925a3b844Bc9e7595f2bD51",
  "verified": false,
  "verified_at": null,
  "verification_method": null,
  "is_deleted": false,
  "created_at": "2026-01-20T06:35:15Z",
  "updated_at": "2026-01-20T06:35:15Z"
}
```

**Response (Error - 400 Bad Request)**:
```json
{
  "error": "Invalid request: wallet_address is required"
}
```

**Response (Error - 409 Conflict)**:
```json
{
  "error": "Address already exists"
}
```

**Response (Error - 500 Internal Server Error)**:
```json
{
  "error": "Failed to add address"
}
```

---

## Implementation Verification

### Backend Implementation Status

#### 1. Handler (`internal/handlers/handlers.go`)
```go
func (h *Handler) AddAddress(c *gin.Context) {
    userID, err := h.getUserID(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
        return
    }

    var req models.AddAddressRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    addr, err := h.AddressService.AddAddress(c.Request.Context(), userID, req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, addr)
}
```
**Status**: ✅ Implemented

#### 2. Service (`internal/services/address.go`)
```go
func (s *AddressService) AddAddress(ctx context.Context, userID int, req models.AddAddressRequest) (*models.WithdrawalAddress, error) {
    addr := &models.WithdrawalAddress{
        UserID:        userID,
        AddressAlias:  req.AddressAlias,
        ChainType:     req.ChainType,
        WalletAddress: req.WalletAddress,
        Verified:      false,
    }

    createdAddr, err := s.repo.CreateAddress(ctx, addr)
    if err != nil {
        if err == repository.ErrAlreadyExists {
            return nil, errors.New("address already exists")
        }
        return nil, err
    }

    return createdAddr, nil
}
```
**Status**: ✅ Implemented

#### 3. Repository (`internal/repository/postgres/address.go`)
```go
func (r *AddressRepository) CreateAddress(ctx context.Context, address *models.WithdrawalAddress) (*models.WithdrawalAddress, error) {
    err := r.db.QueryRowContext(ctx,
        `INSERT INTO withdrawal_address_whitelist (
            user_id, address_alias, chain_type, wallet_address, verified,
            verified_at, verification_method, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id`,
        address.UserID, address.AddressAlias, address.ChainType, address.WalletAddress,
        address.Verified, address.VerifiedAt, address.VerificationMethod,
        time.Now(), time.Now(),
    ).Scan(&address.ID)
    if err != nil {
        if err.Error() == "pq: duplicate key value violates unique constraint \"withdrawal_address_whitelist_user_id_wallet_address_key\"" {
            return nil, repository.ErrAlreadyExists
        }
        return nil, err
    }
    return address, nil
}
```
**Status**: ✅ Implemented

#### 4. Route (`internal/routes/routes.go`)
```go
addresses := protected.Group("/addresses")
{
    addresses.GET("", h.GetAddresses)
    addresses.POST("", h.AddAddress)
    addresses.POST("/:id/verify", h.VerifyAddress)
    addresses.POST("/:id/set-primary", h.SetPrimaryAddress)
    addresses.POST("/:id/deactivate", h.DeactivateAddress)
}
```
**Status**: ✅ Implemented

---

## Testing Strategy

### Manual Testing (agent-browser)

1. **Login** to the application
2. **Navigate** to `/dashboard/withdraw`
3. **Click** "Add Withdrawal Address" button
4. **Fill** the form:
   - Address Label: "Test Wallet"
   - Blockchain Network: "Ethereum"
   - Wallet Address: "0x742d35Cc6634C0532925a3b844Bc9e7595f2bD51"
5. **Click** "Add Address" button
6. **Verify** success message appears
7. **Verify** address appears in the dropdown

### API Testing (curl)

```bash
# Register and login to get token
TOKEN=$(curl -s -X POST http://localhost:8081/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}' | jq -r '.token')

# Create address
curl -X POST http://localhost:8081/api/addresses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"wallet_address":"0x742d35Cc6634C0532925a3b844Bc9e7595f2bD51","chain_type":"Ethereum","address_alias":"Test Wallet"}'
```

---

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Duplicate address submission | Medium | Unique constraint on (user_id, wallet_address) |
| Invalid address format | Medium | Frontend validation exists |
| Database connection failure | High | Connection pooling, error handling |
| SQL injection | High | Parameterized queries used |

---

## Verification Checklist

- [ ] Backend API returns 201 on successful address creation
- [ ] Address is saved to database table `withdrawal_address_whitelist`
- [ ] Frontend displays success message
- [ ] Address appears in dropdown after creation
- [ ] Duplicate addresses are rejected with 409 error
- [ ] Invalid requests return 400 error
- [ ] Unauthenticated requests return 401 error

---

## Related Documents

- `internal/handlers/handlers.go` - HTTP handlers
- `internal/services/address.go` - Business logic
- `internal/repository/postgres/address.go` - Database operations
- `src/pages/dashboard/Withdraw.tsx` - Frontend implementation
