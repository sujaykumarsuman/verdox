package model

import (
	"time"

	"github.com/google/uuid"
)

type TeamMemberRole string

const (
	TeamMemberRoleAdmin      TeamMemberRole = "admin"
	TeamMemberRoleMaintainer TeamMemberRole = "maintainer"
	TeamMemberRoleViewer     TeamMemberRole = "viewer"
)

type TeamMemberStatus string

const (
	TeamMemberStatusPending  TeamMemberStatus = "pending"
	TeamMemberStatusApproved TeamMemberStatus = "approved"
	TeamMemberStatusRejected TeamMemberStatus = "rejected"
)

type Team struct {
	ID                      uuid.UUID  `db:"id" json:"id"`
	Name                    string     `db:"name" json:"name"`
	Slug                    string     `db:"slug" json:"slug"`
	IsDiscoverable          bool       `db:"is_discoverable" json:"is_discoverable"`
	CreatedBy               *uuid.UUID `db:"created_by" json:"created_by"`
	GithubPATEncrypted      *string    `db:"github_pat_encrypted" json:"-"`
	GithubPATNonce          []byte     `db:"github_pat_nonce" json:"-"`
	GithubPATSetAt          *time.Time `db:"github_pat_set_at" json:"-"`
	GithubPATSetBy          *uuid.UUID `db:"github_pat_set_by" json:"-"`
	GithubPATGithubUsername *string    `db:"github_pat_github_username" json:"-"`
	CreatedAt               time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt               time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt               *time.Time `db:"deleted_at" json:"-"`
}

func (t *Team) HasPAT() bool {
	return t.GithubPATEncrypted != nil && *t.GithubPATEncrypted != ""
}

type TeamMember struct {
	ID        uuid.UUID        `db:"id" json:"id"`
	TeamID    uuid.UUID        `db:"team_id" json:"team_id"`
	UserID    uuid.UUID        `db:"user_id" json:"user_id"`
	Role      TeamMemberRole   `db:"role" json:"role"`
	Status    TeamMemberStatus `db:"status" json:"status"`
	InvitedBy *uuid.UUID       `db:"invited_by" json:"invited_by"`
	CreatedAt time.Time        `db:"created_at" json:"created_at"`
}

type TeamJoinRequest struct {
	ID           uuid.UUID        `db:"id" json:"id"`
	TeamID       uuid.UUID        `db:"team_id" json:"team_id"`
	UserID       uuid.UUID        `db:"user_id" json:"user_id"`
	Message      *string          `db:"message" json:"message"`
	Status       TeamMemberStatus `db:"status" json:"status"`
	ReviewedBy   *uuid.UUID       `db:"reviewed_by" json:"reviewed_by"`
	RoleAssigned *TeamMemberRole  `db:"role_assigned" json:"role_assigned"`
	CreatedAt    time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time        `db:"updated_at" json:"updated_at"`
}

