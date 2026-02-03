package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type IdempotencyRecordModel struct {
	ID           int64           `json:"id"`
	UserID       int64           `json:"user_id"`
	RequestID    string          `json:"request_id"`
	BizType      string          `json:"biz_type"`
	Status       string          `json:"status"`
	ResultData   json.RawMessage `json:"result_data"`
	ErrorMessage string          `json:"error_message"`
	CreatedAt    time.Time       `json:"created_at"`
	CompletedAt  sql.NullTime    `json:"completed_at"`
	TTLExpireAt  time.Time       `json:"ttl_expire_at"`
}

type IdempotencyRepository struct {
	db *sql.DB
}

func NewIdempotencyRepository(db *sql.DB) *IdempotencyRepository {
	return &IdempotencyRepository{db: db}
}

// FindByRequestID 根据请求ID查找幂等性记录
func (r *IdempotencyRepository) FindByRequestID(ctx context.Context, requestID string) (*IdempotencyRecordModel, error) {
	query := `
		SELECT id, user_id, request_id, biz_type, status, result_data, error_message,
		       created_at, completed_at, ttl_expire_at
		FROM idempotency_record
		WHERE request_id = $1 AND ttl_expire_at > NOW()
	`
	var record IdempotencyRecordModel
	err := r.db.QueryRowContext(ctx, query, requestID).Scan(
		&record.ID, &record.UserID, &record.RequestID, &record.BizType,
		&record.Status, &record.ResultData, &record.ErrorMessage,
		&record.CreatedAt, &record.CompletedAt, &record.TTLExpireAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// FindByUserAndRequest 根据用户ID、请求ID和业务类型查找
func (r *IdempotencyRepository) FindByUserAndRequest(ctx context.Context, userID int64, requestID, bizType string) (*IdempotencyRecordModel, error) {
	query := `
		SELECT id, user_id, request_id, biz_type, status, result_data, error_message,
		       created_at, completed_at, ttl_expire_at
		FROM idempotency_record
		WHERE user_id = $1 AND request_id = $2 AND biz_type = $3 AND ttl_expire_at > NOW()
	`
	var record IdempotencyRecordModel
	err := r.db.QueryRowContext(ctx, query, userID, requestID, bizType).Scan(
		&record.ID, &record.UserID, &record.RequestID, &record.BizType,
		&record.Status, &record.ResultData, &record.ErrorMessage,
		&record.CreatedAt, &record.CompletedAt, &record.TTLExpireAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// Create 创建幂等性记录
func (r *IdempotencyRepository) Create(ctx context.Context, record *IdempotencyRecordModel) error {
	query := `
		INSERT INTO idempotency_record (user_id, request_id, biz_type, status, result_data, error_message, created_at, completed_at, ttl_expire_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id, request_id, biz_type) DO NOTHING
		RETURNING id
	`
	now := time.Now()
	record.CreatedAt = now
	record.TTLExpireAt = now.Add(24 * time.Hour)

	err := r.db.QueryRowContext(ctx, query,
		record.UserID, record.RequestID, record.BizType, record.Status,
		record.ResultData, record.ErrorMessage, record.CreatedAt,
		record.CompletedAt, record.TTLExpireAt,
	).Scan(&record.ID)

	if err == sql.ErrNoRows {
		return nil
	}
	return err
}

// UpdateStatus 更新状态
func (r *IdempotencyRepository) UpdateStatus(ctx context.Context, requestID, status string, resultData json.RawMessage, errorMessage string) error {
	query := `
		UPDATE idempotency_record
		SET status = $1, result_data = $2, error_message = $3, completed_at = NOW()
		WHERE request_id = $4
	`
	_, err := r.db.ExecContext(ctx, query, status, resultData, errorMessage, requestID)
	return err
}

// DeleteExpired 删除过期记录
func (r *IdempotencyRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM idempotency_record WHERE ttl_expire_at < NOW()`
	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// CreateTable 创建幂等性表（如果不存在）
func (r *IdempotencyRepository) CreateTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS idempotency_record (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			request_id TEXT NOT NULL,
			biz_type TEXT NOT NULL,
			status TEXT DEFAULT 'PROCESSING' NOT NULL,
			result_data JSONB,
			error_message TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
			completed_at TIMESTAMP WITH TIME ZONE,
			ttl_expire_at TIMESTAMP WITH TIME ZONE NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_idempotency_record_request_id ON idempotency_record(request_id);
		CREATE INDEX IF NOT EXISTS idx_idempotency_record_ttl ON idempotency_record(ttl_expire_at);
		CREATE UNIQUE INDEX IF NOT EXISTS uk_idempotency ON idempotency_record(user_id, request_id, biz_type);
	`
	_, err := r.db.ExecContext(ctx, query)
	return err
}

// EnsureTableExists 确保表存在
func (r *IdempotencyRepository) EnsureTableExists(ctx context.Context) error {
	return r.CreateTable(ctx)
}
