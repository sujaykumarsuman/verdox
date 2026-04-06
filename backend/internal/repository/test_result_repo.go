package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
)

type TestResultRepository interface {
	BatchCreate(ctx context.Context, results []model.TestResult) error
	ListByRunID(ctx context.Context, runID uuid.UUID) ([]model.TestResult, error)
	GetByRunIDAndTestName(ctx context.Context, runID uuid.UUID, testName string) (*model.TestResult, error)
}

type testResultRepo struct {
	db *sqlx.DB
}

func NewTestResultRepository(db *sqlx.DB) TestResultRepository {
	return &testResultRepo{db: db}
}

func (r *testResultRepo) BatchCreate(ctx context.Context, results []model.TestResult) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `INSERT INTO test_results (test_run_id, test_name, status, duration_ms, error_message, log_output)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	for i := range results {
		err := tx.QueryRowxContext(ctx, query,
			results[i].TestRunID, results[i].TestName, results[i].Status,
			results[i].DurationMs, results[i].ErrorMessage, results[i].LogOutput,
		).Scan(&results[i].ID, &results[i].CreatedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *testResultRepo) ListByRunID(ctx context.Context, runID uuid.UUID) ([]model.TestResult, error) {
	var results []model.TestResult
	err := r.db.SelectContext(ctx, &results,
		"SELECT * FROM test_results WHERE test_run_id = $1 ORDER BY test_name", runID)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (r *testResultRepo) GetByRunIDAndTestName(ctx context.Context, runID uuid.UUID, testName string) (*model.TestResult, error) {
	var result model.TestResult
	err := r.db.GetContext(ctx, &result,
		"SELECT * FROM test_results WHERE test_run_id = $1 AND test_name = $2", runID, testName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &result, err
}
