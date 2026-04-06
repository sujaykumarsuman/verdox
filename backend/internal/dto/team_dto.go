package dto

import (
	"time"

	"github.com/sujaykumarsuman/verdox/backend/internal/model"
)

// --- Team CRUD ---

type CreateTeamRequest struct {
	Name string `json:"name" validate:"required,min=2,max=128"`
	Slug string `json:"slug" validate:"required,min=2,max=128"`
}

type UpdateTeamRequest struct {
	Name string `json:"name" validate:"required,min=2,max=128"`
}

type TeamResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	MyRole      string `json:"my_role,omitempty"`
	MemberCount int    `json:"member_count"`
	RepoCount   int    `json:"repo_count"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func NewTeamResponse(t *model.Team) TeamResponse {
	return TeamResponse{
		ID:        t.ID.String(),
		Name:      t.Name,
		Slug:      t.Slug,
		CreatedAt: t.CreatedAt.Format(time.RFC3339),
		UpdatedAt: t.UpdatedAt.Format(time.RFC3339),
	}
}

// TeamDetailResponse includes members and repositories.
type TeamDetailResponse struct {
	TeamResponse
	Members      []MemberResponse   `json:"members"`
	Repositories []TeamRepoResponse `json:"repositories"`
}

// --- Members ---

type InviteMemberRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
	Role   string `json:"role" validate:"required,oneof=admin maintainer viewer"`
}

type UpdateMemberRequest struct {
	Role   string `json:"role,omitempty" validate:"omitempty,oneof=admin maintainer viewer"`
	Status string `json:"status,omitempty" validate:"omitempty,oneof=approved rejected"`
}

type MemberResponse struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	Username  string  `json:"username"`
	Email     string  `json:"email"`
	AvatarURL *string `json:"avatar_url"`
	Role      string  `json:"role"`
	Status    string  `json:"status"`
	InvitedBy *string `json:"invited_by"`
	CreatedAt string  `json:"created_at"`
}

// --- Join Requests ---

type SubmitJoinRequestDTO struct {
	Message string `json:"message" validate:"max=500"`
}

type ReviewJoinRequestDTO struct {
	Status string `json:"status" validate:"required,oneof=approved rejected"`
	Role   string `json:"role,omitempty" validate:"omitempty,oneof=admin maintainer viewer"`
}

type JoinRequestUserResponse struct {
	ID        string  `json:"id"`
	Username  string  `json:"username"`
	Email     string  `json:"email"`
	AvatarURL *string `json:"avatar_url"`
}

type JoinRequestResponse struct {
	ID           string                  `json:"id"`
	User         JoinRequestUserResponse `json:"user"`
	Message      *string                 `json:"message"`
	Status       string                  `json:"status"`
	RoleAssigned *string                 `json:"role_assigned"`
	ReviewedBy   *string                 `json:"reviewed_by"`
	CreatedAt    string                  `json:"created_at"`
	UpdatedAt    string                  `json:"updated_at"`
}

// --- Repository Assignment ---

type AssignRepoRequest struct {
	RepositoryID string `json:"repository_id" validate:"required,uuid"`
}

type TeamRepoResponse struct {
	ID             string `json:"id"`
	TeamID         string `json:"team_id"`
	RepositoryID   string `json:"repository_id"`
	RepositoryName string `json:"repository_name"`
	GithubFullName string `json:"github_full_name"`
	IsActive       bool   `json:"is_active"`
	AddedBy        *string `json:"added_by"`
	CreatedAt      string `json:"created_at"`
}

// --- Discovery ---

type DiscoverableTeamResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	MemberCount int     `json:"member_count"`
	RepoCount   int     `json:"repo_count"`
	CreatedAt   string  `json:"created_at"`
	UserStatus  *string `json:"user_status"`
}

// --- PAT ---

type SetPATRequest struct {
	Token string `json:"token" validate:"required"`
}

type PATInfoResponse struct {
	IsConfigured   bool   `json:"is_configured"`
	GithubUsername string `json:"github_username,omitempty"`
	SetAt          string `json:"set_at,omitempty"`
}

type PATValidationResponse struct {
	Valid          bool   `json:"valid"`
	GithubUsername string `json:"github_username,omitempty"`
	Error          string `json:"error,omitempty"`
}
