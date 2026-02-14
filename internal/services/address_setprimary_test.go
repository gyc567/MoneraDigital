package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"monera-digital/internal/repository"
)

func TestAddressService_SetPrimary(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully sets primary address", func(t *testing.T) {
		mockRepo := new(MockAddressRepository)
		service := NewAddressService(mockRepo)

		mockRepo.On("SetPrimary", ctx, 1, 10).Return(nil)

		err := service.SetPrimary(ctx, 1, 10)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns error when address not found", func(t *testing.T) {
		mockRepo := new(MockAddressRepository)
		service := NewAddressService(mockRepo)

		mockRepo.On("SetPrimary", ctx, 1, 99).Return(repository.ErrNotFound)

		err := service.SetPrimary(ctx, 1, 99)

		assert.Error(t, err)
		assert.Equal(t, repository.ErrNotFound, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns error when database fails", func(t *testing.T) {
		mockRepo := new(MockAddressRepository)
		service := NewAddressService(mockRepo)

		dbError := errors.New("database connection failed")
		mockRepo.On("SetPrimary", ctx, 1, 10).Return(dbError)

		err := service.SetPrimary(ctx, 1, 10)

		assert.Error(t, err)
		assert.Equal(t, dbError, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestAddressService_SetPrimary_Validation(t *testing.T) {
	ctx := context.Background()

	t.Run("handles zero userID gracefully", func(t *testing.T) {
		mockRepo := new(MockAddressRepository)
		service := NewAddressService(mockRepo)

		mockRepo.On("SetPrimary", ctx, 0, 10).Return(repository.ErrNotFound)

		err := service.SetPrimary(ctx, 0, 10)

		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("handles zero addressID gracefully", func(t *testing.T) {
		mockRepo := new(MockAddressRepository)
		service := NewAddressService(mockRepo)

		mockRepo.On("SetPrimary", ctx, 1, 0).Return(repository.ErrNotFound)

		err := service.SetPrimary(ctx, 1, 0)

		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}
