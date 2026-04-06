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

type TeamResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
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
