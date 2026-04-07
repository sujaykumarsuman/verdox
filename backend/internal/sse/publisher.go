package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Event types published over SSE.
const (
	EventBanned             = "banned"
	EventUnbanned           = "unbanned"
	EventBanReviewRequested = "ban_review_requested"
	EventNotificationNew    = "notification_new"
	EventTestComplete       = "test_complete"
)

// Envelope is the JSON payload sent over SSE.
type Envelope struct {
	Type      string    `json:"type"`
	Data      any       `json:"data"`
	Timestamp time.Time `json:"timestamp"`
}

// ChannelForUser returns the Redis Pub/Sub channel name for a user.
func ChannelForUser(userID uuid.UUID) string {
	return fmt.Sprintf("sse:user:%s", userID.String())
}

// PublishEvent sends an SSE event to a specific user via Redis Pub/Sub.
func PublishEvent(ctx context.Context, rdb *redis.Client, userID uuid.UUID, eventType string, data any) error {
	env := Envelope{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now().UTC(),
	}
	payload, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal sse event: %w", err)
	}
	return rdb.Publish(ctx, ChannelForUser(userID), payload).Err()
}

// PublishToUsers sends an SSE event to multiple users.
func PublishToUsers(ctx context.Context, rdb *redis.Client, userIDs []uuid.UUID, eventType string, data any) {
	for _, uid := range userIDs {
		_ = PublishEvent(ctx, rdb, uid, eventType, data)
	}
}
