package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
)

type PasswordResetRepository interface {
	Create(ctx context.Context, pr *model.PasswordReset) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.PasswordReset, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	InvalidateForUser(ctx context.Context, userID uuid.UUID) error
}

type passwordResetRepo struct {
	db *sqlx.DB
}

func NewPasswordResetRepository(db *sqlx.DB) PasswordResetRepository {
	return &passwordResetRepo{db: db}
}

func (r *passwordResetRepo) Create(ctx context.Context, pr *model.PasswordReset) error {
	query := `INSERT INTO password_resets (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`
	return r.db.QueryRowxContext(ctx, query,
		pr.UserID, pr.TokenHash, pr.ExpiresAt,
	).Scan(&pr.ID, &pr.CreatedAt)
}

func (r *passwordResetRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*model.PasswordReset, error) {
	var pr model.PasswordReset
	err := r.db.GetContext(ctx, &pr,
		"SELECT * FROM password_resets WHERE token_hash = $1", tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &pr, err
}

func (r *passwordResetRepo) MarkUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE password_resets SET used_at = now() WHERE id = $1", id)
	return err
}

func (r *passwordResetRepo) InvalidateForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE password_resets SET used_at = now() WHERE user_id = $1 AND used_at IS NULL", userID)
	return err
}
