package dto

import (
	"time"

	"github.com/sujaykumarsuman/verdox/backend/internal/model"
)

type AddRepositoryRequest struct {
	GithubURL string `json:"github_url" validate:"required,github_url"`
	TeamID    string `json:"team_id" validate:"required,uuid"`
}

type UpdateRepositoryRequest struct {
	Description *string `json:"description"`
}

type RepositoryResponse struct {
	ID             string  `json:"id"`
	GithubRepoID   int64   `json:"github_repo_id"`
	GithubFullName string  `json:"github_full_name"`
	Name           string  `json:"name"`
	Description    *string `json:"description"`
	DefaultBranch  string  `json:"default_branch"`
	CloneStatus    string  `json:"clone_status"`
	IsActive       bool    `json:"is_active"`
	TeamID         string  `json:"team_id"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type RepositoryListResponse struct {
	Repositories []RepositoryResponse `json:"repositories"`
	Meta         PaginationMeta       `json:"meta"`
}

type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type BranchResponse struct {
	Name      string `json:"name"`
	CommitSHA string `json:"commit_sha"`
}

type CommitResponse struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
	Author  string `json:"author"`
	Date    string `json:"date"`
}

type ResyncResponse struct {
	Message       string `json:"message"`
	DefaultBranch string `json:"default_branch"`
	BranchCount   int    `json:"branch_count"`
}

func NewRepositoryResponse(repo *model.Repository, teamID string) RepositoryResponse {
	return RepositoryResponse{
		ID:             repo.ID.String(),
		GithubRepoID:   repo.GithubRepoID,
		GithubFullName: repo.GithubFullName,
		Name:           repo.Name,
		Description:    repo.Description,
		DefaultBranch:  repo.DefaultBranch,
		CloneStatus:    repo.CloneStatus,
		IsActive:       repo.IsActive,
		TeamID:         teamID,
		CreatedAt:      repo.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      repo.UpdatedAt.Format(time.RFC3339),
	}
}
