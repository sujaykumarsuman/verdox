package model

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleRoot      UserRole = "root"
	RoleAdmin     UserRole = "admin"
	RoleModerator UserRole = "moderator"
	RoleUser      UserRole = "user"
)

type User struct {
	ID           uuid.UUID `db:"id" json:"id"`
	Username     string    `db:"username" json:"username"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Role         UserRole  `db:"role" json:"role"`
	AvatarURL    *string   `db:"avatar_url" json:"avatar_url"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	IsBanned     bool      `db:"is_banned" json:"is_banned"`
	BanReason    *string   `db:"ban_reason" json:"ban_reason"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}
