package model

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID               uuid.UUID `db:"id"`
	UserID           uuid.UUID `db:"user_id"`
	RefreshTokenHash string    `db:"refresh_token_hash"`
	ExpiresAt        time.Time `db:"expires_at"`
	CreatedAt        time.Time `db:"created_at"`
}
