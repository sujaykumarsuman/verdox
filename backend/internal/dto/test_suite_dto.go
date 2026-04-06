package dto

import (
	"time"

	"github.com/sujaykumarsuman/verdox/backend/internal/model"
)

// --- Requests ---

type CreateTestSuiteRequest struct {
	Name           string  `json:"name" validate:"required,min=1,max=255"`
	Type           string  `json:"type" validate:"required,oneof=unit integration"`
	ConfigPath     *string `json:"config_path"`
	TimeoutSeconds *int    `json:"timeout_seconds" validate:"omitempty,min=30,max=3600"`
}

type UpdateTestSuiteRequest struct {
	Name           *string `json:"name" validate:"omitempty,min=1,max=255"`
	Type           *string `json:"type" validate:"omitempty,oneof=unit integration"`
	ConfigPath     *string `json:"config_path"`
	TimeoutSeconds *int    `json:"timeout_seconds" validate:"omitempty,min=30,max=3600"`
}

// --- Responses ---

type TestSuiteResponse struct {
	ID             string  `json:"id"`
	RepositoryID   string  `json:"repository_id"`
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	ConfigPath     *string `json:"config_path"`
	TimeoutSeconds int     `json:"timeout_seconds"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

func NewTestSuiteResponse(s *model.TestSuite) TestSuiteResponse {
	return TestSuiteResponse{
		ID:             s.ID.String(),
		RepositoryID:   s.RepositoryID.String(),
		Name:           s.Name,
		Type:           string(s.Type),
		ConfigPath:     s.ConfigPath,
		TimeoutSeconds: s.TimeoutSeconds,
		CreatedAt:      s.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      s.UpdatedAt.Format(time.RFC3339),
	}
}
