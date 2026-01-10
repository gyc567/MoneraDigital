// internal/validator/validator.go
package validator

import (
	"fmt"
	"regexp"
)

// Validator interface for validation operations
type Validator interface {
	ValidateEmail(email string) error
	ValidatePassword(password string) error
	ValidateAmount(amount float64) error
	ValidateAddress(address string) error
	ValidateAsset(asset string) error
	ValidateDuration(days int) error
}

// DefaultValidator implements Validator interface
type DefaultValidator struct{}

// NewValidator creates a new validator instance
func NewValidator() Validator {
	return &DefaultValidator{}
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

// ValidateEmail validates email format
func (v *DefaultValidator) ValidateEmail(email string) error {
	if email == "" {
		return &ValidationError{Field: "email", Message: "email is required"}
	}

	// RFC 5322 simplified email regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return &ValidationError{Field: "email", Message: "invalid email format"}
	}

	if len(email) > 255 {
		return &ValidationError{Field: "email", Message: "email is too long (max 255 characters)"}
	}

	return nil
}

// ValidatePassword validates password strength
func (v *DefaultValidator) ValidatePassword(password string) error {
	if password == "" {
		return &ValidationError{Field: "password", Message: "password is required"}
	}

	if len(password) < 8 {
		return &ValidationError{Field: "password", Message: "password must be at least 8 characters"}
	}

	if len(password) > 128 {
		return &ValidationError{Field: "password", Message: "password is too long (max 128 characters)"}
	}

	// Check for at least one uppercase letter
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	// Check for at least one lowercase letter
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	// Check for at least one digit
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)

	if !hasUpper || !hasLower || !hasDigit {
		return &ValidationError{
			Field:   "password",
			Message: "password must contain uppercase, lowercase, and digit characters",
		}
	}

	return nil
}

// ValidateAmount validates numeric amount
func (v *DefaultValidator) ValidateAmount(amount float64) error {
	if amount <= 0 {
		return &ValidationError{Field: "amount", Message: "amount must be greater than 0"}
	}

	if amount > 1e15 { // Prevent overflow
		return &ValidationError{Field: "amount", Message: "amount is too large"}
	}

	return nil
}

// ValidateAddress validates blockchain address format
func (v *DefaultValidator) ValidateAddress(address string) error {
	if address == "" {
		return &ValidationError{Field: "address", Message: "address is required"}
	}

	if len(address) < 20 || len(address) > 100 {
		return &ValidationError{Field: "address", Message: "address length must be between 20 and 100 characters"}
	}

	// Basic alphanumeric check (supports most blockchain addresses)
	addressRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !addressRegex.MatchString(address) {
		return &ValidationError{Field: "address", Message: "address contains invalid characters"}
	}

	return nil
}

// ValidateAsset validates asset type
func (v *DefaultValidator) ValidateAsset(asset string) error {
	validAssets := map[string]bool{
		"BTC":  true,
		"ETH":  true,
		"USDC": true,
		"USDT": true,
	}

	if !validAssets[asset] {
		return &ValidationError{
			Field:   "asset",
			Message: "asset must be one of: BTC, ETH, USDC, USDT",
		}
	}

	return nil
}

// ValidateDuration validates lending duration
func (v *DefaultValidator) ValidateDuration(days int) error {
	if days <= 0 {
		return &ValidationError{Field: "duration_days", Message: "duration must be greater than 0"}
	}

	if days > 365 {
		return &ValidationError{Field: "duration_days", Message: "duration must not exceed 365 days"}
	}

	return nil
}
