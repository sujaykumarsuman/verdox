package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
)

type TestCaseRepository interface {
	BatchCreate(ctx context.Context, cases []model.TestCase) error
	ListByGroupID(ctx context.Context, groupID uuid.UUID, page, perPage int) ([]model.TestCase, int, error)
	ListByRunID(ctx context.Context, runID uuid.UUID, page, perPage int) ([]model.TestCase, int, error)
	ListFailedByRunID(ctx context.Context, runID uuid.UUID) ([]model.TestCase, error)
	CountByRunIDAndStatus(ctx context.Context, runID uuid.UUID) (map[model.TestResultStatus]int, error)
}

type testCaseRepo struct {
	db *sqlx.DB
}

func NewTestCaseRepository(db *sqlx.DB) TestCaseRepository {
	return &testCaseRepo{db: db}
}

func (r *testCaseRepo) BatchCreate(ctx context.Context, cases []model.TestCase) error {
	if len(cases) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const batchSize = 500
	for i := 0; i < len(cases); i += batchSize {
		end := i + batchSize
		if end > len(cases) {
			end = len(cases)
		}
		batch := cases[i:end]

		valueStrings := make([]string, len(batch))
		args := make([]interface{}, 0, len(batch)*10)
		for j, c := range batch {
			base := j * 10
			valueStrings[j] = fmt.Sprintf(
				"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				base+1, base+2, base+3, base+4, base+5,
				base+6, base+7, base+8, base+9, base+10,
			)
			args = append(args, c.TestGroupID, c.TestRunID, c.CaseID, c.Name,
				c.Status, c.DurationMs, c.ErrorMessage, c.StackTrace,
				c.RetryCount, c.LogsURL)
		}

		query := fmt.Sprintf(
			`INSERT INTO test_cases (test_group_id, test_run_id, case_id, name, status, duration_ms, error_message, stack_trace, retry_count, logs_url)
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
			if err := rows.Scan(&cases[idx].ID, &cases[idx].CreatedAt); err != nil {
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

func (r *testCaseRepo) ListByGroupID(ctx context.Context, groupID uuid.UUID, page, perPage int) ([]model.TestCase, int, error) {
	offset := (page - 1) * perPage

	var total int
	if err := r.db.GetContext(ctx, &total,
		"SELECT COUNT(*) FROM test_cases WHERE test_group_id = $1", groupID); err != nil {
		return nil, 0, err
	}

	var cases []model.TestCase
	err := r.db.SelectContext(ctx, &cases,
		"SELECT * FROM test_cases WHERE test_group_id = $1 ORDER BY name LIMIT $2 OFFSET $3",
		groupID, perPage, offset)
	if err != nil {
		return nil, 0, err
	}

	return cases, total, nil
}

func (r *testCaseRepo) ListByRunID(ctx context.Context, runID uuid.UUID, page, perPage int) ([]model.TestCase, int, error) {
	offset := (page - 1) * perPage

	var total int
	if err := r.db.GetContext(ctx, &total,
		"SELECT COUNT(*) FROM test_cases WHERE test_run_id = $1", runID); err != nil {
		return nil, 0, err
	}

	var cases []model.TestCase
	err := r.db.SelectContext(ctx, &cases,
		"SELECT * FROM test_cases WHERE test_run_id = $1 ORDER BY name LIMIT $2 OFFSET $3",
		runID, perPage, offset)
	if err != nil {
		return nil, 0, err
	}

	return cases, total, nil
}

func (r *testCaseRepo) ListFailedByRunID(ctx context.Context, runID uuid.UUID) ([]model.TestCase, error) {
	var cases []model.TestCase
	err := r.db.SelectContext(ctx, &cases,
		"SELECT * FROM test_cases WHERE test_run_id = $1 AND status = 'fail' ORDER BY name", runID)
	if err != nil {
		return nil, err
	}
	return cases, nil
}

type statusCount struct {
	Status model.TestResultStatus `db:"status"`
	Count  int                    `db:"count"`
}

func (r *testCaseRepo) CountByRunIDAndStatus(ctx context.Context, runID uuid.UUID) (map[model.TestResultStatus]int, error) {
	var counts []statusCount
	err := r.db.SelectContext(ctx, &counts,
		"SELECT status, COUNT(*) as count FROM test_cases WHERE test_run_id = $1 GROUP BY status", runID)
	if err != nil {
		return nil, err
	}

	result := make(map[model.TestResultStatus]int)
	for _, c := range counts {
		result[c.Status] = c.Count
	}
	return result, nil
}
