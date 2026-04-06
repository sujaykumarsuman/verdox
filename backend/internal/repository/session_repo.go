package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
)

type SessionRepository interface {
	Create(ctx context.Context, session *model.Session) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Session, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*model.Session, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.Session, error)
	DeleteByID(ctx context.Context, id uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) (int64, error)
}

type sessionRepo struct {
	db *sqlx.DB
}

func NewSessionRepository(db *sqlx.DB) SessionRepository {
	return &sessionRepo{db: db}
}

func (r *sessionRepo) Create(ctx context.Context, session *model.Session) error {
	query := `INSERT INTO sessions (user_id, refresh_token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`
	return r.db.QueryRowxContext(ctx, query,
		session.UserID, session.RefreshTokenHash, session.ExpiresAt,
	).Scan(&session.ID, &session.CreatedAt)
}

func (r *sessionRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Session, error) {
	var session model.Session
	err := r.db.GetContext(ctx, &session, "SELECT * FROM sessions WHERE id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &session, err
}

func (r *sessionRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.Session, error) {
	var session model.Session
	err := r.db.GetContext(ctx, &session,
		"SELECT * FROM sessions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1", userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &session, err
}

func (r *sessionRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*model.Session, error) {
	var session model.Session
	err := r.db.GetContext(ctx, &session,
		"SELECT * FROM sessions WHERE refresh_token_hash = $1 LIMIT 1", tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &session, err
}

func (r *sessionRepo) DeleteByID(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM sessions WHERE id = $1", id)
	return err
}

func (r *sessionRepo) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM sessions WHERE user_id = $1", userID)
	return err
}

func (r *sessionRepo) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < now()")
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
