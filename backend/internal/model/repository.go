package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	CloneStatusPending = "pending"
	CloneStatusCloning = "cloning"
	CloneStatusReady   = "ready"
	CloneStatusFailed  = "failed"
	CloneStatusEvicted = "evicted"
)

type Repository struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	GithubRepoID    int64      `db:"github_repo_id" json:"github_repo_id"`
	GithubFullName  string     `db:"github_full_name" json:"github_full_name"`
	Name            string     `db:"name" json:"name"`
	Description     *string    `db:"description" json:"description"`
	DefaultBranch   string     `db:"default_branch" json:"default_branch"`
	LocalPath       *string    `db:"local_path" json:"local_path,omitempty"`
	CloneStatus     string     `db:"clone_status" json:"clone_status"`
	IsActive        bool       `db:"is_active" json:"is_active"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
}
