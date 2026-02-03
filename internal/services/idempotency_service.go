package services

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"monera-digital/internal/cache"
	"monera-digital/internal/repository/postgres"
)

var (
	ErrIdempotencyNotFound   = errors.New("idempotency record not found")
	ErrIdempotencyProcessing = errors.New("request is being processed")
	ErrIdempotencyCompleted  = errors.New("request has already been completed")
	ErrIdempotencyFailed     = errors.New("request has failed")
)

// IdempotencyRecord 幂等性记录
type IdempotencyRecord struct {
	Key       string          `json:"key"`
	Request   string          `json:"request"`
	Response  json.RawMessage `json:"response,omitempty"`
	Result    string          `json:"result,omitempty"`
	Error     string          `json:"error,omitempty"`
	Status    string          `json:"status"` // PROCESSING, COMPLETED, FAILED
	CreatedAt time.Time       `json:"created_at"`
	ExpiresAt time.Time       `json:"expires_at"`
}

const (
	IdempotencyStatusProcessing = "PROCESSING"
	IdempotencyStatusCompleted  = "COMPLETED"
	IdempotencyStatusFailed     = "FAILED"
)

// 错误变量
var (
	IdempotencyProcessing = errors.New("request is being processed")
	IdempotencyCompleted  = errors.New("request has already been completed")
	IdempotencyFailed     = errors.New("request has failed")
)

// IdempotencyService 幂等服务
type IdempotencyService struct {
	redisCache *cache.RedisCache
	dbRepo     *postgres.IdempotencyRepository
}

func NewIdempotencyService(redisCache *cache.RedisCache, dbRepo *postgres.IdempotencyRepository) *IdempotencyService {
	return &IdempotencyService{
		redisCache: redisCache,
		dbRepo:     dbRepo,
	}
}

// CheckOrCreate 检查或创建幂等性记录
// 优先使用 Redis，如果 Redis 不可用则回退到数据库
func (s *IdempotencyService) CheckOrCreate(ctx context.Context, key string) (*IdempotencyRecord, bool, error) {
	// 优先使用 Redis
	if s.redisCache != nil {
		return s.checkOrCreateFromRedis(ctx, key)
	}

	// 回退到数据库
	if s.dbRepo != nil {
		return s.checkOrCreateFromDB(ctx, key)
	}

	// 都没有，返回 nil, true, nil（允许请求继续）
	return nil, true, nil
}

// checkOrCreateFromRedis 从 Redis 检查或创建幂等性记录
func (s *IdempotencyService) checkOrCreateFromRedis(ctx context.Context, key string) (*IdempotencyRecord, bool, error) {
	data, err := s.redisCache.Get(ctx, "idempotency:"+key)
	if err != nil {
		// 如果键不存在，创建新记录
		record := &IdempotencyRecord{
			Key:       key,
			Status:    IdempotencyStatusProcessing,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		jsonData, _ := json.Marshal(record)
		err := s.redisCache.Set(ctx, "idempotency:"+key, string(jsonData), 24*time.Hour)
		if err != nil {
			// Redis 写入失败，回退到数据库
			return s.checkOrCreateFromDB(ctx, key)
		}

		return record, true, nil
	}

	// 解析现有记录
	var record IdempotencyRecord
	if err := json.Unmarshal([]byte(data), &record); err != nil {
		return nil, true, nil
	}

	// 检查是否过期
	if time.Now().After(record.ExpiresAt) {
		record.Status = IdempotencyStatusProcessing
		record.CreatedAt = time.Now()
		record.ExpiresAt = time.Now().Add(24 * time.Hour)

		jsonData, _ := json.Marshal(record)
		s.redisCache.Set(ctx, "idempotency:"+key, string(jsonData), 24*time.Hour)

		return &record, true, nil
	}

	// 如果正在处理，返回错误
	if record.Status == IdempotencyStatusProcessing {
		return &record, false, ErrIdempotencyProcessing
	}

	// 如果已完成，返回记录
	return &record, false, nil
}

// checkOrCreateFromDB 从数据库检查或创建幂等性记录
func (s *IdempotencyService) checkOrCreateFromDB(ctx context.Context, key string) (*IdempotencyRecord, bool, error) {
	// 从数据库查找
	dbRecord, err := s.dbRepo.FindByRequestID(ctx, key)
	if err != nil {
		return nil, true, nil
	}

	if dbRecord != nil {
		// 转换为服务记录
		record := &IdempotencyRecord{
			Key:       dbRecord.RequestID,
			Status:    dbRecord.Status,
			CreatedAt: dbRecord.CreatedAt,
			ExpiresAt: dbRecord.TTLExpireAt,
		}

		if dbRecord.Status == IdempotencyStatusCompleted {
			return record, false, nil
		}

		if dbRecord.Status == IdempotencyStatusProcessing {
			// 检查是否超时（超过 5 分钟视为超时）
			if time.Since(dbRecord.CreatedAt) > 5*time.Minute {
				// 允许重新处理
				return record, true, nil
			}
			return record, false, ErrIdempotencyProcessing
		}

		return record, false, nil
	}

	// 创建新记录
	now := time.Now()
	newRecord := &postgres.IdempotencyRecordModel{
		RequestID:   key,
		BizType:     "WEALTH_SUBSCRIBE",
		Status:      IdempotencyStatusProcessing,
		CreatedAt:   now,
		TTLExpireAt: now.Add(24 * time.Hour),
	}

	err = s.dbRepo.Create(ctx, newRecord)
	if err != nil {
		// 如果创建失败（可能是并发），尝试重新查找
		dbRecord, _ := s.dbRepo.FindByRequestID(ctx, key)
		if dbRecord != nil {
			record := &IdempotencyRecord{
				Key:       dbRecord.RequestID,
				Status:    dbRecord.Status,
				CreatedAt: dbRecord.CreatedAt,
				ExpiresAt: dbRecord.TTLExpireAt,
			}
			return record, false, nil
		}
	}

	return &IdempotencyRecord{
		Key:       key,
		Status:    IdempotencyStatusProcessing,
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}, true, nil
}

// Complete 标记请求完成
func (s *IdempotencyService) Complete(ctx context.Context, key string, result string) error {
	// 优先更新 Redis
	if s.redisCache != nil {
		data, err := s.redisCache.Get(ctx, "idempotency:"+key)
		if err == nil {
			var record IdempotencyRecord
			if err := json.Unmarshal([]byte(data), &record); err == nil {
				record.Result = result
				record.Status = IdempotencyStatusCompleted
				jsonData, _ := json.Marshal(record)
				s.redisCache.Set(ctx, "idempotency:"+key, string(jsonData), 24*time.Hour)
			}
		}
	}

	// 同时更新数据库（如果可用）
	if s.dbRepo != nil {
		resultJSON, _ := json.Marshal(result)
		s.dbRepo.UpdateStatus(ctx, key, IdempotencyStatusCompleted, resultJSON, "")
	}

	return nil
}

// Fail 标记请求失败
func (s *IdempotencyService) Fail(ctx context.Context, key string, errMsg string) error {
	// 优先更新 Redis
	if s.redisCache != nil {
		data, err := s.redisCache.Get(ctx, "idempotency:"+key)
		if err == nil {
			var record IdempotencyRecord
			if err := json.Unmarshal([]byte(data), &record); err == nil {
				record.Error = errMsg
				record.Status = IdempotencyStatusFailed
				jsonData, _ := json.Marshal(record)
				s.redisCache.Set(ctx, "idempotency:"+key, string(jsonData), 24*time.Hour)
			}
		}
	}

	// 同时更新数据库（如果可用）
	if s.dbRepo != nil {
		s.dbRepo.UpdateStatus(ctx, key, IdempotencyStatusFailed, nil, errMsg)
	}

	return nil
}

// GetResult 获取已完成的结果
func (s *IdempotencyService) GetResult(ctx context.Context, key string) (string, error) {
	// 优先从 Redis 获取
	if s.redisCache != nil {
		data, err := s.redisCache.Get(ctx, "idempotency:"+key)
		if err == nil {
			var record IdempotencyRecord
			if err := json.Unmarshal([]byte(data), &record); err == nil {
				if record.Status == IdempotencyStatusCompleted {
					return record.Result, nil
				}
			}
		}
	}

	// 回退到数据库
	if s.dbRepo != nil {
		dbRecord, err := s.dbRepo.FindByRequestID(ctx, key)
		if err == nil && dbRecord != nil && dbRecord.Status == IdempotencyStatusCompleted {
			var result string
			if len(dbRecord.ResultData) > 0 {
				json.Unmarshal(dbRecord.ResultData, &result)
			}
			return result, nil
		}
	}

	return "", ErrIdempotencyNotFound
}

// EnsureTableExists 确保幂等性表存在
func (s *IdempotencyService) EnsureTableExists(ctx context.Context) error {
	if s.dbRepo != nil {
		return s.dbRepo.EnsureTableExists(ctx)
	}
	return nil
}
