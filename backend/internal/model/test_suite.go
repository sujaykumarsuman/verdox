package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Execution modes for test suites.
const (
	ExecutionModeContainer = "container"
	ExecutionModeGHA       = "gha"
)

// EnvVarsMap is a map[string]string that can be scanned from JSONB.
type EnvVarsMap map[string]string

func (e *EnvVarsMap) Scan(src interface{}) error {
	if src == nil {
		*e = make(map[string]string)
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("unsupported type for EnvVarsMap: %T", src)
	}
	m := make(map[string]string)
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*e = m
	return nil
}

func (e EnvVarsMap) Value() (driver.Value, error) {
	if e == nil {
		return "{}", nil
	}
	data, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

type TestSuite struct {
	ID             uuid.UUID  `db:"id" json:"id"`
	RepositoryID   uuid.UUID  `db:"repository_id" json:"repository_id"`
	Name           string     `db:"name" json:"name"`
	Type           string     `db:"type" json:"type"`
	ExecutionMode  string     `db:"execution_mode" json:"execution_mode"`
	DockerImage    *string    `db:"docker_image" json:"docker_image"`
	TestCommand    *string    `db:"test_command" json:"test_command"`
	GHAWorkflowID  *string    `db:"gha_workflow_id" json:"gha_workflow_id"`
	EnvVars        EnvVarsMap `db:"env_vars" json:"env_vars"`
	ConfigPath     *string    `db:"config_path" json:"config_path"`
	TimeoutSeconds int        `db:"timeout_seconds" json:"timeout_seconds"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at" json:"updated_at"`
}
