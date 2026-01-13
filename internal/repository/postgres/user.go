// internal/repository/postgres/user.go
package postgres

import (
	"context"
	"database/sql"
	"time"

	"monera-digital/internal/repository"
)

// UserRepository PostgreSQL 用户仓储实现
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository 创建用户仓储
func NewUserRepository(db *sql.DB) repository.User {
	return &UserRepository{db: db}
}

// GetByEmail 根据邮箱获取用户
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*repository.UserModel, error) {
	var user repository.UserModel

	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, email, password, two_factor_enabled, two_factor_secret,
		        two_factor_backup_codes, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.TwoFactorEnabled,
		&user.TwoFactorSecret,
		&user.TwoFactorBackupCodes,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByID 根据ID获取用户
func (r *UserRepository) GetByID(ctx context.Context, id int) (*repository.UserModel, error) {
	var user repository.UserModel

	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, email, password, two_factor_enabled, two_factor_secret,
		        two_factor_backup_codes, created_at, updated_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.TwoFactorEnabled,
		&user.TwoFactorSecret,
		&user.TwoFactorBackupCodes,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// Create 创建用户
func (r *UserRepository) Create(ctx context.Context, email, passwordHash string) (*repository.UserModel, error) {
	var user repository.UserModel

	err := r.db.QueryRowContext(
		ctx,
		`INSERT INTO users (email, password, created_at, updated_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, email, password, two_factor_enabled, two_factor_secret,
		           two_factor_backup_codes, created_at, updated_at`,
		email,
		passwordHash,
		time.Now(),
		time.Now(),
	).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.TwoFactorEnabled,
		&user.TwoFactorSecret,
		&user.TwoFactorBackupCodes,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		// 检查唯一性约束
		if err.Error() == "pq: duplicate key value violates unique constraint \"users_email_key\"" {
			return nil, repository.ErrAlreadyExists
		}
		return nil, err
	}

	return &user, nil
}

// Update 更新用户
func (r *UserRepository) Update(ctx context.Context, user *repository.UserModel) error {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE users
		 SET email = $1, password = $2, two_factor_enabled = $3,
		     two_factor_secret = $4, two_factor_backup_codes = $5, updated_at = $6
		 WHERE id = $7`,
		user.Email,
		user.Password,
		user.TwoFactorEnabled,
		user.TwoFactorSecret,
		user.TwoFactorBackupCodes,
		time.Now(),
		user.ID,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// Delete 删除用户
func (r *UserRepository) Delete(ctx context.Context, id int) error {
	result, err := r.db.ExecContext(
		ctx,
		`DELETE FROM users WHERE id = $1`,
		id,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}
