package dto

import (
	"time"

	"github.com/sujaykumarsuman/verdox/backend/internal/model"
)

// --- Requests ---

type CreateTestSuiteRequest struct {
	Name           string             `json:"name" validate:"required,min=1,max=255"`
	Type           string             `json:"type" validate:"required,min=1,max=50"`
	ExecutionMode  string             `json:"execution_mode" validate:"omitempty,oneof=container gha"`
	DockerImage    *string            `json:"docker_image" validate:"omitempty,max=255"`
	TestCommand    *string            `json:"test_command"`
	GHAWorkflowID  *string            `json:"gha_workflow_id" validate:"omitempty,max=255"`
	EnvVars        map[string]string  `json:"env_vars"`
	ConfigPath     *string            `json:"config_path"`
	TimeoutSeconds *int               `json:"timeout_seconds" validate:"omitempty,min=30,max=3600"`
}

type UpdateTestSuiteRequest struct {
	Name           *string            `json:"name" validate:"omitempty,min=1,max=255"`
	Type           *string            `json:"type" validate:"omitempty,min=1,max=50"`
	ExecutionMode  *string            `json:"execution_mode" validate:"omitempty,oneof=container gha"`
	DockerImage    *string            `json:"docker_image" validate:"omitempty,max=255"`
	TestCommand    *string            `json:"test_command"`
	GHAWorkflowID  *string            `json:"gha_workflow_id" validate:"omitempty,max=255"`
	EnvVars        map[string]string  `json:"env_vars"`
	ConfigPath     *string            `json:"config_path"`
	TimeoutSeconds *int               `json:"timeout_seconds" validate:"omitempty,min=30,max=3600"`
}

// --- Responses ---

type TestSuiteResponse struct {
	ID             string            `json:"id"`
	RepositoryID   string            `json:"repository_id"`
	Name           string            `json:"name"`
	Type           string            `json:"type"`
	ExecutionMode  string            `json:"execution_mode"`
	DockerImage    *string           `json:"docker_image"`
	TestCommand    *string           `json:"test_command"`
	GHAWorkflowID  *string           `json:"gha_workflow_id"`
	EnvVars        map[string]string `json:"env_vars"`
	ConfigPath     *string           `json:"config_path"`
	TimeoutSeconds int               `json:"timeout_seconds"`
	CreatedAt      string            `json:"created_at"`
	UpdatedAt      string            `json:"updated_at"`
}

func NewTestSuiteResponse(s *model.TestSuite) TestSuiteResponse {
	envVars := map[string]string(s.EnvVars)
	if envVars == nil {
		envVars = make(map[string]string)
	}
	return TestSuiteResponse{
		ID:             s.ID.String(),
		RepositoryID:   s.RepositoryID.String(),
		Name:           s.Name,
		Type:           s.Type,
		ExecutionMode:  s.ExecutionMode,
		DockerImage:    s.DockerImage,
		TestCommand:    s.TestCommand,
		GHAWorkflowID:  s.GHAWorkflowID,
		EnvVars:        envVars,
		ConfigPath:     s.ConfigPath,
		TimeoutSeconds: s.TimeoutSeconds,
		CreatedAt:      s.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      s.UpdatedAt.Format(time.RFC3339),
	}
}
