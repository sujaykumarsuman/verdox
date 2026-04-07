package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
)

// NotificationWithSender is a projection joining notifications with the sender's username.
type NotificationWithSender struct {
	model.Notification
	SenderUsername *string `db:"sender_username" json:"sender_username"`
}

type NotificationRepository interface {
	Create(ctx context.Context, n *model.Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error)
	ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]NotificationWithSender, int, error)
	CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
	MarkRead(ctx context.Context, id, userID uuid.UUID) error
	MarkAllRead(ctx context.Context, userID uuid.UUID) error
}

type notificationRepo struct {
	db *sqlx.DB
}

func NewNotificationRepository(db *sqlx.DB) NotificationRepository {
	return &notificationRepo{db: db}
}

func (r *notificationRepo) Create(ctx context.Context, n *model.Notification) error {
	query := `INSERT INTO notifications (user_id, type, subject, body, action_type, action_payload, sender_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`
	return r.db.QueryRowxContext(ctx, query,
		n.UserID, n.Type, n.Subject, n.Body, n.ActionType, n.ActionPayload, n.SenderID,
	).Scan(&n.ID, &n.CreatedAt)
}

func (r *notificationRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	var n model.Notification
	err := r.db.GetContext(ctx, &n, "SELECT * FROM notifications WHERE id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &n, err
}

func (r *notificationRepo) ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]NotificationWithSender, int, error) {
	var total int
	err := r.db.GetContext(ctx, &total,
		"SELECT COUNT(*) FROM notifications WHERE user_id = $1", userID)
	if err != nil {
		return nil, 0, err
	}

	var notifications []NotificationWithSender
	err = r.db.SelectContext(ctx, &notifications,
		`SELECT n.id, n.user_id, n.type, n.subject, n.body, n.is_read,
			n.action_type, n.action_payload, n.sender_id, n.created_at,
			u.username AS sender_username
		FROM notifications n
		LEFT JOIN users u ON u.id = n.sender_id
		WHERE n.user_id = $1
		ORDER BY n.created_at DESC
		LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}

func (r *notificationRepo) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		"SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false", userID)
	return count, err
}

func (r *notificationRepo) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		"UPDATE notifications SET is_read = true WHERE id = $1 AND user_id = $2", id, userID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *notificationRepo) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE notifications SET is_read = true WHERE user_id = $1 AND is_read = false", userID)
	return err
}
