package services

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"monera-digital/internal/models"
	"monera-digital/internal/repository"
)

type AddressService struct {
	repo repository.Address
}

func NewAddressService(repo repository.Address) *AddressService {
	return &AddressService{repo: repo}
}

func (s *AddressService) GetAddresses(ctx context.Context, userID int) ([]*models.WithdrawalAddress, error) {
	return s.repo.GetAddressesByUserID(ctx, userID)
}

func (s *AddressService) AddAddress(ctx context.Context, userID int, req models.AddAddressRequest) (*models.WithdrawalAddress, error) {
	// Check if already exists (optional, unique constraint handles it but maybe check alias?)
	// DB Unique Constraint: (user_id, wallet_address)

	addr := &models.WithdrawalAddress{
		UserID:        userID,
		AddressAlias:  req.AddressAlias,
		ChainType:     req.ChainType,
		WalletAddress: req.WalletAddress,
		Verified:      false, // New addresses need verification
		// VerifiedAt: nil
		// VerificationMethod: nil
	}

	// PRD 4.1: If first time, needs verification. (Default verified=false)
	// If existing whitelist, maybe skip?
	// But AddAddress implies adding new.

	createdAddr, err := s.repo.CreateAddress(ctx, addr)
	if err != nil {
		if err == repository.ErrAlreadyExists {
			return nil, errors.New("address already exists")
		}
		return nil, err
	}

	return createdAddr, nil
}

func (s *AddressService) VerifyAddress(ctx context.Context, userID int, addressID int, method string) error {
	addr, err := s.repo.GetAddressByID(ctx, addressID)
	if err != nil {
		return err
	}
	if addr.UserID != userID {
		return errors.New("address not found")
	}

	addr.Verified = true
	now := time.Now()
	addr.VerifiedAt = sql.NullTime{Time: now, Valid: true}
	addr.VerificationMethod = sql.NullString{String: method, Valid: true}

	return s.repo.UpdateAddress(ctx, addr)
}

func (s *AddressService) DeleteAddress(ctx context.Context, userID int, addressID int) error {
	addr, err := s.repo.GetAddressByID(ctx, addressID)
	if err != nil {
		return err
	}
	if addr.UserID != userID {
		return errors.New("address not found")
	}

	return s.repo.DeleteAddress(ctx, addressID)
}

func (s *AddressService) SetPrimary(ctx context.Context, userID int, addressID int) error {
	return s.repo.SetPrimary(ctx, userID, addressID)
}
