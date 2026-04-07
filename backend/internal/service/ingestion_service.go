package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
)

type IngestionService struct {
	suiteRepo repository.TestSuiteRepository
	runRepo   repository.TestRunRepository
	groupRepo repository.TestGroupRepository
	caseRepo  repository.TestCaseRepository
	repoRepo  repository.RepositoryRepository
	log       zerolog.Logger
}

func NewIngestionService(
	suiteRepo repository.TestSuiteRepository,
	runRepo repository.TestRunRepository,
	groupRepo repository.TestGroupRepository,
	caseRepo repository.TestCaseRepository,
	repoRepo repository.RepositoryRepository,
	log zerolog.Logger,
) *IngestionService {
	return &IngestionService{
		suiteRepo: suiteRepo,
		runRepo:   runRepo,
		groupRepo: groupRepo,
		caseRepo:  caseRepo,
		repoRepo:  repoRepo,
		log:       log,
	}
}

type IngestResult struct {
	ReportID string
	RunIDs   []uuid.UUID
}

// IngestHierarchical processes a full multi-suite payload.
// It matches each payload suite to an existing test_suite (or creates one),
// creates a test_run per suite with a shared report_id,
// and inserts test_groups and test_cases.
func (s *IngestionService) IngestHierarchical(ctx context.Context, repoID uuid.UUID, payload *dto.HierarchicalPayload) (*IngestResult, error) {
	reportID := uuid.New().String()

	existingSuites, err := s.suiteRepo.ListByRepositoryID(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("list suites: %w", err)
	}
	suiteByName := make(map[string]*model.TestSuite, len(existingSuites))
	for i := range existingSuites {
		suiteByName[existingSuites[i].Name] = &existingSuites[i]
	}

	result := &IngestResult{ReportID: reportID}

	for i, ps := range payload.Jobs {
		suite := suiteByName[ps.Name]
		if suite == nil {
			// Auto-create suite for this ingested data
			suite = &model.TestSuite{
				RepositoryID:   repoID,
				Name:           ps.Name,
				Type:           ps.Type,
				ExecutionMode:  "fork_gha",
				TimeoutSeconds: 300,
			}
			if err := s.suiteRepo.Create(ctx, suite); err != nil {
				return nil, fmt.Errorf("create suite %q: %w", ps.Name, err)
			}
			suiteByName[ps.Name] = suite
		}

		runNumber, err := s.runRepo.NextRunNumber(ctx, suite.ID)
		if err != nil {
			return nil, fmt.Errorf("next run number for suite %q: %w", ps.Name, err)
		}

		run := &model.TestRun{
			TestSuiteID: suite.ID,
			RunNumber:   runNumber,
			Branch:      payload.Branch,
			CommitHash:  payload.CommitSHA,
			Status:      model.TestRunStatusRunning,
			ReportID:    &reportID,
		}
		if err := s.runRepo.Create(ctx, run); err != nil {
			return nil, fmt.Errorf("create run for suite %q: %w", ps.Name, err)
		}
		result.RunIDs = append(result.RunIDs, run.ID)

		if err := s.ingestJobResults(ctx, run.ID, ps, i); err != nil {
			return nil, fmt.Errorf("ingest results for suite %q: %w", ps.Name, err)
		}
	}

	return result, nil
}

// IngestForRun processes a hierarchical payload for a single existing run.
// Used when the poller downloads the artifact and finds a hierarchical payload.
// All jobs in the payload become groups within this single run.
func (s *IngestionService) IngestForRun(ctx context.Context, runID uuid.UUID, payload *dto.HierarchicalPayload) error {
	if len(payload.Jobs) == 0 {
		return fmt.Errorf("no jobs in payload")
	}

	// Build groups from all jobs' tests
	var groups []model.TestGroup
	// Track how many cases each group has, so we can assign IDs after batch create
	type groupCases struct {
		cases []model.TestCase
	}
	var perGroup []groupCases
	sortOrder := 0

	for _, job := range payload.Jobs {
		for _, pt := range job.Tests {
			group, cases := s.buildGroupAndCases(runID, pt, sortOrder)
			groups = append(groups, group)
			perGroup = append(perGroup, groupCases{cases: cases})
			sortOrder++
		}
	}

	// Batch create groups (populates group IDs)
	if err := s.groupRepo.BatchCreate(ctx, groups); err != nil {
		return fmt.Errorf("batch create groups: %w", err)
	}

	// Now assign the real group IDs to cases and collect all cases
	var allCases []model.TestCase
	for gi, g := range groups {
		for ci := range perGroup[gi].cases {
			perGroup[gi].cases[ci].TestGroupID = g.ID
		}
		allCases = append(allCases, perGroup[gi].cases...)
	}

	// Batch create cases
	if err := s.caseRepo.BatchCreate(ctx, allCases); err != nil {
		return fmt.Errorf("batch create cases: %w", err)
	}

	// Compute and store summary
	summary := computeRunSummary(groups, allCases)
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return fmt.Errorf("marshal summary: %w", err)
	}

	finalStatus := model.TestRunStatusPassed
	if summary.Failed > 0 {
		finalStatus = model.TestRunStatusFailed
	}

	sj := model.SummaryJSON(summaryJSON)
	return s.runRepo.UpdateSummaryAndFinish(ctx, runID, sj, finalStatus)
}

func (s *IngestionService) ingestJobResults(ctx context.Context, runID uuid.UUID, ps dto.PayloadJob, suiteSortOffset int) error {
	groups := make([]model.TestGroup, 0, len(ps.Tests))

	for i, pt := range ps.Tests {
		group := model.TestGroup{
			TestRunID: runID,
			GroupID:   pt.TestID,
			Name:      pt.Name,
			Status:    mapStatus(pt.Status),
			SortOrder: suiteSortOffset*1000 + i,
		}
		if pt.Package != "" {
			group.Package = &pt.Package
		}

		// Compute stats from cases
		var totalDuration int
		for _, pc := range pt.Cases {
			switch mapStatus(pc.Status) {
			case model.TestResultStatusPass:
				group.Passed++
			case model.TestResultStatusFail:
				group.Failed++
			case model.TestResultStatusSkip:
				group.Skipped++
			}
			durationMs := int(pc.DurationSeconds * 1000)
			totalDuration += durationMs
		}
		group.Total = len(pt.Cases)
		if group.Total > 0 {
			rate := float64(group.Passed) / float64(group.Total) * 100
			rate = math.Round(rate*100) / 100
			group.PassRate = &rate
		}
		group.DurationMs = &totalDuration

		// Derive status from cases if not provided
		if group.Status == model.TestResultStatusUnknown {
			if group.Failed > 0 {
				group.Status = model.TestResultStatusFail
			} else if group.Total == group.Passed+group.Skipped {
				group.Status = model.TestResultStatusPass
			}
		}

		groups = append(groups, group)
	}

	if err := s.groupRepo.BatchCreate(ctx, groups); err != nil {
		return fmt.Errorf("batch create groups: %w", err)
	}

	// Now insert cases, linking to the created groups
	var allCases []model.TestCase
	for gi, pt := range ps.Tests {
		groupID := groups[gi].ID
		for _, pc := range pt.Cases {
			tc := model.TestCase{
				TestGroupID: groupID,
				TestRunID:   runID,
				CaseID:      pc.CaseID,
				Name:        pc.Name,
				Status:      mapStatus(pc.Status),
				RetryCount:  pc.RetryCount,
			}
			if pc.DurationSeconds > 0 {
				ms := int(pc.DurationSeconds * 1000)
				tc.DurationMs = &ms
			}
			if pc.ErrorMessage != "" {
				tc.ErrorMessage = &pc.ErrorMessage
			}
			if pc.StackTrace != "" {
				tc.StackTrace = &pc.StackTrace
			}
			if pc.LogsURL != "" {
				tc.LogsURL = &pc.LogsURL
			}
			allCases = append(allCases, tc)
		}
	}

	if err := s.caseRepo.BatchCreate(ctx, allCases); err != nil {
		return fmt.Errorf("batch create cases: %w", err)
	}

	// Compute run summary and update
	summary := computeRunSummary(groups, allCases)
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return fmt.Errorf("marshal summary: %w", err)
	}

	finalStatus := model.TestRunStatusPassed
	if summary.Failed > 0 {
		finalStatus = model.TestRunStatusFailed
	}

	sj := model.SummaryJSON(summaryJSON)
	if err := s.runRepo.UpdateSummaryAndFinish(ctx, runID, sj, finalStatus); err != nil {
		return fmt.Errorf("update run summary: %w", err)
	}

	return nil
}

func (s *IngestionService) buildGroupAndCases(runID uuid.UUID, pt dto.PayloadTest, sortOrder int) (model.TestGroup, []model.TestCase) {
	group := model.TestGroup{
		TestRunID: runID,
		GroupID:   pt.TestID,
		Name:      pt.Name,
		Status:    mapStatus(pt.Status),
		SortOrder: sortOrder,
	}
	if pt.Package != "" {
		group.Package = &pt.Package
	}

	cases := make([]model.TestCase, 0, len(pt.Cases))
	var totalDuration int
	for _, pc := range pt.Cases {
		tc := model.TestCase{
			TestRunID:  runID,
			CaseID:     pc.CaseID,
			Name:       pc.Name,
			Status:     mapStatus(pc.Status),
			RetryCount: pc.RetryCount,
		}
		ms := int(pc.DurationSeconds * 1000)
		tc.DurationMs = &ms
		totalDuration += ms
		if pc.ErrorMessage != "" {
			tc.ErrorMessage = &pc.ErrorMessage
		}
		if pc.StackTrace != "" {
			tc.StackTrace = &pc.StackTrace
		}
		if pc.LogsURL != "" {
			tc.LogsURL = &pc.LogsURL
		}
		cases = append(cases, tc)

		// Compute group stats
		switch tc.Status {
		case model.TestResultStatusPass:
			group.Passed++
		case model.TestResultStatusFail:
			group.Failed++
		case model.TestResultStatusSkip:
			group.Skipped++
		}
	}
	group.Total = len(cases)
	// Prefer payload stats duration (from package-level go test elapsed) over sum of individual tests
	if pt.Stats != nil && pt.Stats.DurationSeconds > 0 {
		ms := int(pt.Stats.DurationSeconds * 1000)
		group.DurationMs = &ms
		totalDuration = ms
	} else {
		group.DurationMs = &totalDuration
	}
	if group.Total > 0 {
		rate := math.Round(float64(group.Passed)/float64(group.Total)*10000) / 100
		group.PassRate = &rate
	}
	if group.Status == model.TestResultStatusUnknown {
		if group.Failed > 0 {
			group.Status = model.TestResultStatusFail
		} else if group.Total == group.Passed+group.Skipped {
			group.Status = model.TestResultStatusPass
		}
	}

	return group, cases
}

func computeRunSummary(groups []model.TestGroup, cases []model.TestCase) dto.RunSummaryV2 {
	var totalCases, passed, failed, skipped int
	var totalDurationMs int64

	for _, c := range cases {
		totalCases++
		switch c.Status {
		case model.TestResultStatusPass:
			passed++
		case model.TestResultStatusFail:
			failed++
		case model.TestResultStatusSkip:
			skipped++
		}
	}

	// Use group durations (package-level) which are more accurate than sum of individual test durations
	for _, g := range groups {
		if g.DurationMs != nil {
			totalDurationMs += int64(*g.DurationMs)
		}
	}

	var passRate float64
	if totalCases > 0 {
		passRate = math.Round(float64(passed)/float64(totalCases)*10000) / 100
	}

	return dto.RunSummaryV2{
		TotalJobs:   len(groups),
		TotalCases:  totalCases,
		Passed:      passed,
		Failed:      failed,
		Skipped:     skipped,
		DurationMs:  totalDurationMs,
		PassRate:    passRate,
	}
}

func mapStatus(s string) model.TestResultStatus {
	switch s {
	case "passed", "pass":
		return model.TestResultStatusPass
	case "failed", "fail":
		return model.TestResultStatusFail
	case "skipped", "skip":
		return model.TestResultStatusSkip
	case "error":
		return model.TestResultStatusError
	case "running":
		return model.TestResultStatusRunning
	default:
		return model.TestResultStatusUnknown
	}
}
