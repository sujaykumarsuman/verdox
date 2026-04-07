package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/sujaykumarsuman/verdox/backend/internal/model"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/internal/sse"
)

type NotificationService struct {
	notifRepo repository.NotificationRepository
	userRepo  repository.UserRepository
	rdb       *redis.Client
	log       zerolog.Logger
}

func NewNotificationService(
	notifRepo repository.NotificationRepository,
	userRepo repository.UserRepository,
	rdb *redis.Client,
	log zerolog.Logger,
) *NotificationService {
	return &NotificationService{
		notifRepo: notifRepo,
		userRepo:  userRepo,
		rdb:       rdb,
		log:       log,
	}
}

// CreateAndPublish inserts a notification into the DB and pushes an SSE event to the user.
func (s *NotificationService) CreateAndPublish(ctx context.Context, n *model.Notification) error {
	if err := s.notifRepo.Create(ctx, n); err != nil {
		return fmt.Errorf("create notification: %w", err)
	}

	// Publish SSE event
	eventData := map[string]any{
		"id":      n.ID.String(),
		"type":    string(n.Type),
		"subject": n.Subject,
	}
	if err := sse.PublishEvent(ctx, s.rdb, n.UserID, sse.EventNotificationNew, eventData); err != nil {
		s.log.Error().Err(err).Str("user_id", n.UserID.String()).Msg("failed to publish notification SSE event")
	}

	return nil
}

// CreateForAdmins creates a notification for all admin and root users and publishes SSE events.
func (s *NotificationService) CreateForAdmins(ctx context.Context, notifTemplate *model.Notification) error {
	admins, _, err := s.userRepo.ListFiltered(ctx, "", "", "active", "created_at", "asc", 0, 1000)
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}

	for _, u := range admins {
		if u.Role != model.RoleRoot && u.Role != model.RoleAdmin {
			continue
		}
		n := &model.Notification{
			UserID:        u.ID,
			Type:          notifTemplate.Type,
			Subject:       notifTemplate.Subject,
			Body:          notifTemplate.Body,
			ActionType:    notifTemplate.ActionType,
			ActionPayload: notifTemplate.ActionPayload,
			SenderID:      notifTemplate.SenderID,
		}
		if err := s.CreateAndPublish(ctx, n); err != nil {
			s.log.Error().Err(err).Str("admin_id", u.ID.String()).Msg("failed to create admin notification")
		}
	}

	return nil
}

func (s *NotificationService) List(ctx context.Context, userID uuid.UUID, page, perPage int) ([]repository.NotificationWithSender, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage
	return s.notifRepo.ListByUser(ctx, userID, offset, perPage)
}

func (s *NotificationService) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.notifRepo.CountUnread(ctx, userID)
}

func (s *NotificationService) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	return s.notifRepo.MarkRead(ctx, id, userID)
}

func (s *NotificationService) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return s.notifRepo.MarkAllRead(ctx, userID)
}
