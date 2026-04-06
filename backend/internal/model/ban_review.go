package model

import (
	"time"

	"github.com/google/uuid"
)

type BanReview struct {
	ID            uuid.UUID  `db:"id" json:"id"`
	UserID        uuid.UUID  `db:"user_id" json:"user_id"`
	BanReason     string     `db:"ban_reason" json:"ban_reason"`
	Clarification string     `db:"clarification" json:"clarification"`
	Status        string     `db:"status" json:"status"` // pending, approved, denied
	ReviewedBy    *uuid.UUID `db:"reviewed_by" json:"reviewed_by"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	ReviewedAt    *time.Time `db:"reviewed_at" json:"reviewed_at"`
}
