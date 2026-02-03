// internal/cache/cache_keys.go
package cache

import "fmt"

// Cache key constants and builders for consistent key naming

const (
	// User cache keys
	UserKeyPrefix = "user:"
	UserEmailKey  = "user:email:"

	// Lending cache keys
	LendingKeyPrefix     = "lending:"
	LendingPositionKey   = "lending:position:"
	LendingUserPositions = "lending:user_positions:"

	// Address cache keys
	AddressKeyPrefix = "address:"
	UserAddresses    = "address:user:"

	// Withdrawal cache keys
	WithdrawalKeyPrefix = "withdrawal:"
	UserWithdrawals     = "withdrawal:user:"

	// Rate limit cache keys
	RateLimitKeyPrefix = "ratelimit:"

	// Session cache keys
	SessionKeyPrefix = "session:"

	// Token blacklist cache keys
	TokenBlacklistPrefix = "token:blacklist:"

	// Idempotency cache keys
	IdempotencyKeyPrefix = "idempotency:"
)

// User cache key builders
func UserCacheKey(userID int) string {
	return fmt.Sprintf("%s%d", UserKeyPrefix, userID)
}

func UserEmailCacheKey(email string) string {
	return fmt.Sprintf("%s%s", UserEmailKey, email)
}

// Lending cache key builders
func LendingPositionCacheKey(positionID int) string {
	return fmt.Sprintf("%s%d", LendingPositionKey, positionID)
}

func UserLendingPositionsCacheKey(userID int) string {
	return fmt.Sprintf("%s%d", LendingUserPositions, userID)
}

// Address cache key builders
func UserAddressesCacheKey(userID int) string {
	return fmt.Sprintf("%s%d", UserAddresses, userID)
}

func AddressCacheKey(addressID int) string {
	return fmt.Sprintf("%s%d", AddressKeyPrefix, addressID)
}

// Withdrawal cache key builders
func UserWithdrawalsCacheKey(userID int) string {
	return fmt.Sprintf("%s%d", UserWithdrawals, userID)
}

func WithdrawalCacheKey(withdrawalID int) string {
	return fmt.Sprintf("%s%d", WithdrawalKeyPrefix, withdrawalID)
}

// Rate limit cache key builder
func RateLimitCacheKey(identifier string) string {
	return fmt.Sprintf("%s%s", RateLimitKeyPrefix, identifier)
}

// Session cache key builder
func SessionCacheKey(sessionID string) string {
	return fmt.Sprintf("%s%s", SessionKeyPrefix, sessionID)
}

// Token blacklist cache key builder
func TokenBlacklistCacheKey(token string) string {
	return fmt.Sprintf("%s%s", TokenBlacklistPrefix, token)
}

// Idempotency cache key builder
func IdempotencyCacheKey(key string) string {
	return fmt.Sprintf("%s%s", IdempotencyKeyPrefix, key)
}
