package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
)

// BanReviewWithUser is a projection joining ban_reviews with users for display.
type BanReviewWithUser struct {
	model.BanReview
	Username string `db:"username"`
	Email    string `db:"email"`
}

type BanReviewRepository interface {
	Create(ctx context.Context, review *model.BanReview) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.BanReview, error)
	ListPending(ctx context.Context) ([]BanReviewWithUser, error)
	HasPendingForUser(ctx context.Context, userID uuid.UUID) (bool, error)
	CountByUser(ctx context.Context, userID uuid.UUID) (int, error)
	DeleteByUser(ctx context.Context, userID uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, reviewedBy uuid.UUID) error
	CountPending(ctx context.Context) (int, error)
}

type banReviewRepo struct {
	db *sqlx.DB
}

func NewBanReviewRepository(db *sqlx.DB) BanReviewRepository {
	return &banReviewRepo{db: db}
}

func (r *banReviewRepo) Create(ctx context.Context, review *model.BanReview) error {
	query := `INSERT INTO ban_reviews (user_id, ban_reason, clarification)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`
	return r.db.QueryRowxContext(ctx, query,
		review.UserID, review.BanReason, review.Clarification,
	).Scan(&review.ID, &review.CreatedAt)
}

func (r *banReviewRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.BanReview, error) {
	var review model.BanReview
	err := r.db.GetContext(ctx, &review, "SELECT * FROM ban_reviews WHERE id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &review, err
}

func (r *banReviewRepo) ListPending(ctx context.Context) ([]BanReviewWithUser, error) {
	var reviews []BanReviewWithUser
	err := r.db.SelectContext(ctx, &reviews,
		`SELECT br.*, u.username, u.email
		FROM ban_reviews br
		JOIN users u ON u.id = br.user_id
		WHERE br.status = 'pending'
		ORDER BY br.created_at DESC`)
	return reviews, err
}

func (r *banReviewRepo) HasPendingForUser(ctx context.Context, userID uuid.UUID) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		"SELECT COUNT(*) FROM ban_reviews WHERE user_id = $1 AND status = 'pending'", userID)
	return count > 0, err
}

func (r *banReviewRepo) CountByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		"SELECT COUNT(*) FROM ban_reviews WHERE user_id = $1", userID)
	return count, err
}

func (r *banReviewRepo) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM ban_reviews WHERE user_id = $1", userID)
	return err
}

func (r *banReviewRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string, reviewedBy uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ban_reviews SET status = $1, reviewed_by = $2, reviewed_at = now() WHERE id = $3`,
		status, reviewedBy, id)
	return err
}

func (r *banReviewRepo) CountPending(ctx context.Context) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		"SELECT COUNT(*) FROM ban_reviews WHERE status = 'pending'")
	return count, err
}
