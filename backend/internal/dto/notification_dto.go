package dto

import (
	"encoding/json"

	"github.com/sujaykumarsuman/verdox/backend/internal/model"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
)

type NotificationResponse struct {
	ID             string                   `json:"id"`
	Type           model.NotificationType   `json:"type"`
	Subject        string                   `json:"subject"`
	Body           string                   `json:"body"`
	IsRead         bool                     `json:"is_read"`
	ActionType     *string                  `json:"action_type"`
	ActionPayload  *json.RawMessage         `json:"action_payload"`
	SenderID       *string                  `json:"sender_id"`
	SenderUsername *string                  `json:"sender_username"`
	CreatedAt      string                   `json:"created_at"`
}

func NewNotificationResponse(n *repository.NotificationWithSender) NotificationResponse {
	resp := NotificationResponse{
		ID:             n.ID.String(),
		Type:           n.Type,
		Subject:        n.Subject,
		Body:           n.Body,
		IsRead:         n.IsRead,
		ActionType:     n.ActionType,
		ActionPayload:  n.ActionPayload,
		SenderUsername: n.SenderUsername,
		CreatedAt:      n.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if n.SenderID != nil {
		s := n.SenderID.String()
		resp.SenderID = &s
	}
	return resp
}

type NotificationListResponse struct {
	Notifications []NotificationResponse `json:"notifications"`
	Total         int                    `json:"total"`
	Page          int                    `json:"page"`
	PerPage       int                    `json:"per_page"`
}

type UnreadCountResponse struct {
	Count int `json:"count"`
}
