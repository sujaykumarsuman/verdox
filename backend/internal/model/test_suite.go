package model

import (
	"time"

	"github.com/google/uuid"
)

type TestType string

const (
	TestTypeUnit        TestType = "unit"
	TestTypeIntegration TestType = "integration"
)

type TestSuite struct {
	ID             uuid.UUID `db:"id" json:"id"`
	RepositoryID   uuid.UUID `db:"repository_id" json:"repository_id"`
	Name           string    `db:"name" json:"name"`
	Type           TestType  `db:"type" json:"type"`
	ConfigPath     *string   `db:"config_path" json:"config_path"`
	TimeoutSeconds int       `db:"timeout_seconds" json:"timeout_seconds"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}
