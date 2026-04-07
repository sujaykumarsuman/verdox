package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type NotificationType string

const (
	NotificationSystem          NotificationType = "system"
	NotificationAdminMessage    NotificationType = "admin_message"
	NotificationBanReview       NotificationType = "ban_review"
	NotificationTestComplete    NotificationType = "test_complete"
	NotificationTeamInvite      NotificationType = "team_invite"
	NotificationTeamJoinRequest NotificationType = "team_join_request"
)

type Notification struct {
	ID            uuid.UUID        `db:"id" json:"id"`
	UserID        uuid.UUID        `db:"user_id" json:"user_id"`
	Type          NotificationType `db:"type" json:"type"`
	Subject       string           `db:"subject" json:"subject"`
	Body          string           `db:"body" json:"body"`
	IsRead        bool             `db:"is_read" json:"is_read"`
	ActionType    *string          `db:"action_type" json:"action_type"`
	ActionPayload *json.RawMessage `db:"action_payload" json:"action_payload"`
	SenderID      *uuid.UUID       `db:"sender_id" json:"sender_id"`
	CreatedAt     time.Time        `db:"created_at" json:"created_at"`
}
