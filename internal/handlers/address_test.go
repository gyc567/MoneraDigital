package handlers

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"monera-digital/internal/dto"
	"monera-digital/internal/models"
)

// TestConvertAddressToDTO 测试地址模型到 DTO 的转换
func TestConvertAddressToDTO(t *testing.T) {
	now := time.Now()
	verifiedTime := now.Add(-1 * time.Hour)

	tests := []struct {
		name     string
		input    *models.WithdrawalAddress
		expected dto.WithdrawalAddressResponse
	}{
		{
			name: "完整地址数据",
			input: &models.WithdrawalAddress{
				ID:            1,
				UserID:        123,
				WalletAddress: "0x1234567890abcdef",
				ChainType:     "ETH",
				AddressAlias:  "我的以太坊地址",
				Verified:      true,
				VerifiedAt:    sql.NullTime{Time: verifiedTime, Valid: true},
				IsDeleted:     false,
				CreatedAt:     now,
			},
			expected: dto.WithdrawalAddressResponse{
				ID:         1,
				UserID:     123,
				Address:    "0x1234567890abcdef",
				Type:       "ETH",
				Label:      "我的以太坊地址",
				IsVerified: true,
				IsDeleted:  false,
				CreatedAt:  now,
				VerifiedAt: &verifiedTime,
			},
		},
		{
			name: "未验证地址",
			input: &models.WithdrawalAddress{
				ID:            2,
				UserID:        123,
				WalletAddress: "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
				ChainType:     "BTC",
				AddressAlias:  "我的比特币地址",
				Verified:      false,
				VerifiedAt:    sql.NullTime{Valid: false},
				IsDeleted:     false,
				CreatedAt:     now,
			},
			expected: dto.WithdrawalAddressResponse{
				ID:         2,
				UserID:     123,
				Address:    "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
				Type:       "BTC",
				Label:      "我的比特币地址",
				IsVerified: false,
				IsDeleted:  false,
				CreatedAt:  now,
				VerifiedAt: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 执行转换
			result := dto.WithdrawalAddressResponse{
				ID:         tt.input.ID,
				UserID:     tt.input.UserID,
				Address:    tt.input.WalletAddress,
				Type:       tt.input.ChainType,
				Label:      tt.input.AddressAlias,
				IsVerified: tt.input.Verified,
				IsDeleted:  tt.input.IsDeleted,
				CreatedAt:  tt.input.CreatedAt,
			}
			if tt.input.VerifiedAt.Valid {
				result.VerifiedAt = &tt.input.VerifiedAt.Time
			}

			// 验证结果
			assert.Equal(t, tt.expected.ID, result.ID)
			assert.Equal(t, tt.expected.UserID, result.UserID)
			assert.Equal(t, tt.expected.Address, result.Address)
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.Equal(t, tt.expected.Label, result.Label)
			assert.Equal(t, tt.expected.IsVerified, result.IsVerified)
			assert.Equal(t, tt.expected.IsDeleted, result.IsDeleted)
			assert.Equal(t, tt.expected.CreatedAt, result.CreatedAt)
			
			if tt.expected.VerifiedAt != nil {
				assert.NotNil(t, result.VerifiedAt)
				assert.Equal(t, *tt.expected.VerifiedAt, *result.VerifiedAt)
			} else {
				assert.Nil(t, result.VerifiedAt)
			}
		})
	}
}

// TestWithdrawalAddressResponse_JSONFormat 测试 DTO JSON 格式
func TestWithdrawalAddressResponse_JSONFormat(t *testing.T) {
	now := time.Date(2026, 1, 28, 10, 30, 0, 0, time.UTC)
	
	response := dto.WithdrawalAddressResponse{
		ID:         1,
		UserID:     123,
		Address:    "0x1234567890abcdef",
		Type:       "ETH",
		Label:      "测试地址",
		IsVerified: true,
		IsDeleted:  false,
		CreatedAt:  now,
		VerifiedAt: &now,
	}

	// 验证字段名使用 snake_case
	assert.Equal(t, 1, response.ID)
	assert.Equal(t, 123, response.UserID)
	assert.Equal(t, "0x1234567890abcdef", response.Address)
	assert.Equal(t, "ETH", response.Type)
	assert.Equal(t, "测试地址", response.Label)
	assert.Equal(t, true, response.IsVerified)
	assert.Equal(t, false, response.IsDeleted)
	assert.Equal(t, now, response.CreatedAt)
	assert.NotNil(t, response.VerifiedAt)
}

// TestWithdrawalAddressResponse_NullVerifiedAt 测试未验证地址的 DTO
func TestWithdrawalAddressResponse_NullVerifiedAt(t *testing.T) {
	now := time.Date(2026, 1, 28, 10, 30, 0, 0, time.UTC)
	
	response := dto.WithdrawalAddressResponse{
		ID:         2,
		UserID:     123,
		Address:    "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
		Type:       "BTC",
		Label:      "未验证地址",
		IsVerified: false,
		IsDeleted:  false,
		CreatedAt:  now,
		VerifiedAt: nil,
	}

	assert.Nil(t, response.VerifiedAt)
	assert.False(t, response.IsVerified)
}
