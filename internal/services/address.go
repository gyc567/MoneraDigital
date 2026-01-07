package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"monera-digital/internal/db"
	"monera-digital/internal/models"
)

type AddressService struct{}

func (s *AddressService) GetAddresses(userID int) ([]models.WithdrawalAddress, error) {
	query := `
		SELECT id, user_id, address, address_type, label, is_verified, is_primary, created_at, verified_at, deactivated_at
		FROM withdrawal_addresses
		WHERE user_id = $1 AND deactivated_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := db.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var addresses []models.WithdrawalAddress
	for rows.Next() {
		var addr models.WithdrawalAddress
		err := rows.Scan(
			&addr.ID, &addr.UserID, &addr.Address, &addr.AddressType, &addr.Label,
			&addr.IsVerified, &addr.IsPrimary, &addr.CreatedAt, &addr.VerifiedAt, &addr.DeactivatedAt,
		)
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, addr)
	}

	return addresses, nil
}

func (s *AddressService) AddAddress(userID int, req models.AddAddressRequest) (*models.WithdrawalAddress, error) {
	query := `
		INSERT INTO withdrawal_addresses (user_id, address, address_type, label)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, address, address_type, label, is_verified, is_primary, created_at
	`

	var addr models.WithdrawalAddress
	err := db.DB.QueryRow(query, userID, req.Address, req.AddressType, req.Label).Scan(
		&addr.ID, &addr.UserID, &addr.Address, &addr.AddressType, &addr.Label,
		&addr.IsVerified, &addr.IsPrimary, &addr.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// TODO: Send verification email
	return &addr, nil
}

func (s *AddressService) VerifyAddress(userID int, addressID int, token string) error {
	// Check verification token
	query := `
		SELECT expires_at
		FROM address_verifications
		WHERE address_id = $1 AND token = $2
	`

	var expiresAt time.Time
	err := db.DB.QueryRow(query, addressID, token).Scan(&expiresAt)
	if err != nil {
		return err
	}

	if time.Now().After(expiresAt) {
		return errors.New("token expired")
	}

	// Update address as verified
	query = `UPDATE withdrawal_addresses SET is_verified = true, verified_at = NOW() WHERE id = $1 AND user_id = $2`
	_, err = db.DB.Exec(query, addressID, userID)
	return err
}

func (s *AddressService) SetPrimaryAddress(userID int, addressID int) error {
	// First, unset all primary addresses for this user
	query := `UPDATE withdrawal_addresses SET is_primary = false WHERE user_id = $1`
	_, err := db.DB.Exec(query, userID)
	if err != nil {
		return err
	}

	// Set this address as primary
	query = `UPDATE withdrawal_addresses SET is_primary = true WHERE id = $1 AND user_id = $2`
	_, err = db.DB.Exec(query, addressID, userID)
	return err
}

func (s *AddressService) DeactivateAddress(userID int, addressID int) error {
	query := `UPDATE withdrawal_addresses SET deactivated_at = NOW() WHERE id = $1 AND user_id = $2`
	_, err := db.DB.Exec(query, addressID, userID)
	return err
}

// Generate verification token
func generateVerificationToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
