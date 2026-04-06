package model

import (
	"time"

	"github.com/google/uuid"
)

type TeamRepository struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	TeamID       uuid.UUID  `db:"team_id" json:"team_id"`
	RepositoryID uuid.UUID  `db:"repository_id" json:"repository_id"`
	AddedBy      *uuid.UUID `db:"added_by" json:"added_by"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
}
