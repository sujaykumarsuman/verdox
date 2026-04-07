package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
)

type TestGroupRepository interface {
	BatchCreate(ctx context.Context, groups []model.TestGroup) error
	ListByRunID(ctx context.Context, runID uuid.UUID) ([]model.TestGroup, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.TestGroup, error)
	GetByRunIDAndGroupID(ctx context.Context, runID uuid.UUID, groupID string) (*model.TestGroup, error)
}

type testGroupRepo struct {
	db *sqlx.DB
}

func NewTestGroupRepository(db *sqlx.DB) TestGroupRepository {
	return &testGroupRepo{db: db}
}

func (r *testGroupRepo) BatchCreate(ctx context.Context, groups []model.TestGroup) error {
	if len(groups) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const batchSize = 100
	for i := 0; i < len(groups); i += batchSize {
		end := i + batchSize
		if end > len(groups) {
			end = len(groups)
		}
		batch := groups[i:end]

		valueStrings := make([]string, len(batch))
		args := make([]interface{}, 0, len(batch)*12)
		for j, g := range batch {
			base := j * 12
			valueStrings[j] = fmt.Sprintf(
				"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				base+1, base+2, base+3, base+4, base+5, base+6,
				base+7, base+8, base+9, base+10, base+11, base+12,
			)
			args = append(args, g.TestRunID, g.GroupID, g.Name, g.Package,
				g.Status, g.Total, g.Passed, g.Failed, g.Skipped,
				g.DurationMs, g.PassRate, g.SortOrder)
		}

		query := fmt.Sprintf(
			`INSERT INTO test_groups (test_run_id, group_id, name, package, status, total, passed, failed, skipped, duration_ms, pass_rate, sort_order)
			VALUES %s
			RETURNING id, created_at`,
			strings.Join(valueStrings, ", "),
		)

		rows, err := tx.QueryxContext(ctx, query, args...)
		if err != nil {
			return err
		}
		idx := i
		for rows.Next() {
			if err := rows.Scan(&groups[idx].ID, &groups[idx].CreatedAt); err != nil {
				rows.Close()
				return err
			}
			idx++
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *testGroupRepo) ListByRunID(ctx context.Context, runID uuid.UUID) ([]model.TestGroup, error) {
	var groups []model.TestGroup
	err := r.db.SelectContext(ctx, &groups,
		"SELECT * FROM test_groups WHERE test_run_id = $1 ORDER BY sort_order, name", runID)
	if err != nil {
		return nil, err
	}
	return groups, nil
}

func (r *testGroupRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.TestGroup, error) {
	var group model.TestGroup
	err := r.db.GetContext(ctx, &group, "SELECT * FROM test_groups WHERE id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &group, err
}

func (r *testGroupRepo) GetByRunIDAndGroupID(ctx context.Context, runID uuid.UUID, groupID string) (*model.TestGroup, error) {
	var group model.TestGroup
	err := r.db.GetContext(ctx, &group,
		"SELECT * FROM test_groups WHERE test_run_id = $1 AND group_id = $2", runID, groupID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &group, err
}
