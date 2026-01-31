# PRD: Fix AddAddress Wallet Lookup

## Problem Statement

Users with existing wallets are getting "wallet not found" errors when trying to add new addresses.

### Root Cause

`WalletService.AddAddress()` only queries `wallet_creation_requests` table for active wallets:

```go
wallet, err := s.repo.GetActiveWalletByUserID(ctx, userID)
// Only checks wallet_creation_requests WHERE status='SUCCESS'
```

However, wallet data is also stored in `user_wallets` table. If a user has a wallet in `user_wallets` but no matching record in `wallet_creation_requests` with status='SUCCESS', the function returns an error.

### Data Flow Issue

1. `CreateWallet` creates record in both tables:
   - `wallet_creation_requests` (tracks creation flow)
   - `user_wallets` (persistent wallet storage)

2. `AddAddress` only checks `wallet_creation_requests`

3. If there's a mismatch between the two tables, users get errors

## Solution

Modify `AddAddress` to check both tables:
1. First try `wallet_creation_requests` (primary source)
2. Fall back to `user_wallets` (secondary source)

### Implementation

Add a new repository method to get wallet from `user_wallets`:

```go
// GetActiveWalletFromUserWallets retrieves the first active wallet from user_wallets
func (r *WalletRepository) GetActiveWalletFromUserWallets(ctx context.Context, userID int) (*models.UserWallet, error)
```

Modify `AddAddress` to use both sources:

```go
func (s *WalletService) AddAddress(ctx context.Context, userID int, req AddAddressRequest) (*models.WalletCreationRequest, error) {
    // Try wallet_creation_requests first
    wallet, err := s.repo.GetActiveWalletByUserID(ctx, userID)
    if err != nil {
        return nil, err
    }
    
    // Fall back to user_wallets
    if wallet == nil {
        userWallet, err := s.repo.GetUserWalletsByUserID(ctx, userID)
        if err != nil {
            return nil, err
        }
        if len(userWallet) > 0 {
            // Convert UserWallet to WalletCreationRequest format
            wallet = convertUserWalletToRequest(userWallet[0])
        }
    }
    
    if wallet == nil {
        return nil, errors.New("wallet not found")
    }
    // ... rest of function
}
```

## Scope

### In Scope
- Fix `AddAddress` wallet lookup logic
- Add repository method for `user_wallets` lookup
- Add unit tests for new logic

### Out of Scope
- Database schema changes
- Changes to `CreateWallet` flow
- Other wallet-related functions

## Acceptance Criteria

- [ ] Users with wallets in `user_wallets` can add addresses
- [ ] Users without any wallet still get proper error
- [ ] All existing tests pass
- [ ] New tests added with 100% coverage for new code

## Technical Notes

### KISS Principles
- Minimal changes to existing code
- No new interfaces or major refactoring
- Simple fallback logic

### Data Conversion
When falling back to `user_wallets`, need to convert `UserWallet` to `WalletCreationRequest` format for the rest of the function.

```go
func convertUserWalletToRequest(uw *models.UserWallet) *models.WalletCreationRequest {
    return &models.WalletCreationRequest{
        UserID:      uw.UserID,
        ProductCode: "X_FINANCE", // Default, may need adjustment
        Currency:    uw.Currency,
        Address:     uw.Address,
        Addresses:   sql.NullString{String: "{}"}, // Empty for now
    }
}
```
