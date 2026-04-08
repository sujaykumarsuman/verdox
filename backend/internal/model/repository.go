package model

import (
	"time"

	"github.com/google/uuid"
)

// Fork status values.
const (
	ForkStatusNone    = "none"
	ForkStatusForking = "forking"
	ForkStatusReady   = "ready"
	ForkStatusFailed  = "failed"
)

type Repository struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	GithubRepoID    int64      `db:"github_repo_id" json:"github_repo_id"`
	GithubFullName  string     `db:"github_full_name" json:"github_full_name"`
	Name            string     `db:"name" json:"name"`
	Description     *string    `db:"description" json:"description"`
	DefaultBranch   string     `db:"default_branch" json:"default_branch"`
	IsActive        bool       `db:"is_active" json:"is_active"`
	ForkFullName    *string    `db:"fork_full_name" json:"fork_full_name"`
	ForkStatus      string     `db:"fork_status" json:"fork_status"`
	ForkSyncedAt    *time.Time `db:"fork_synced_at" json:"fork_synced_at"`
	ForkWorkflowID  *string    `db:"fork_workflow_id" json:"fork_workflow_id"`
	ForkHeadSHA     *string    `db:"fork_head_sha" json:"fork_head_sha"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
}
