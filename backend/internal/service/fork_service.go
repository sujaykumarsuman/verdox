package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"

	"github.com/sujaykumarsuman/verdox/backend/internal/config"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
)

type ForkService struct {
	client *http.Client
	cfg    *config.Config
	db     *sqlx.DB
	log    zerolog.Logger
}

func NewForkService(cfg *config.Config, db *sqlx.DB, log zerolog.Logger) *ForkService {
	return &ForkService{
		client: &http.Client{Timeout: 30 * time.Second},
		cfg:    cfg,
		db:     db,
		log:    log,
	}
}

// IsConfigured returns true if the service account PAT and username are set.
func (s *ForkService) IsConfigured() bool {
	return s.cfg.ServiceAccountPAT != "" && s.cfg.ServiceAccountUsername != ""
}

// SuiteWorkflowFilename returns the workflow filename for a suite (e.g. "verdox-<uuid>.yml").
func SuiteWorkflowFilename(suiteID uuid.UUID) string {
	return fmt.Sprintf("verdox-%s.yml", suiteID.String())
}

// SuiteWorkflowPath returns the full repo path for a suite's workflow file.
func SuiteWorkflowPath(suiteID uuid.UUID) string {
	return fmt.Sprintf(".github/workflows/%s", SuiteWorkflowFilename(suiteID))
}

// SuiteWorkflowSpec holds the info needed to generate one suite's workflow file.
type SuiteWorkflowSpec struct {
	SuiteID        uuid.UUID
	Name           string
	TestCommand    string
	WorkflowConfig *model.WorkflowConfig
}

// ForkRepo forks a repo under the service account. Returns the fork's full name.
func (s *ForkService) ForkRepo(ctx context.Context, repoFullName string) (string, error) {
	parts := strings.SplitN(repoFullName, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid repo name: %s", repoFullName)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/forks", repoFullName)
	payload := map[string]string{
		"organization": "", // fork to user account
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create fork request: %w", err)
	}
	s.setHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fork request: %w", err)
	}
	defer resp.Body.Close()

	// 202 Accepted = fork in progress, 200 = already exists
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("fork failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		FullName string `json:"full_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode fork response: %w", err)
	}

	// Make the fork private (requires GitHub Pro/Team/Enterprise on the service account)
	s.makeForkPrivate(ctx, result.FullName)

	return result.FullName, nil
}

// makeForkPrivate sets a fork to private visibility. Best-effort — logs a warning
// if it fails (e.g. free GitHub account can't have private forks of public repos).
func (s *ForkService) makeForkPrivate(ctx context.Context, forkFullName string) {
	url := fmt.Sprintf("https://api.github.com/repos/%s", forkFullName)
	payload := map[string]interface{}{"private": true}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	s.setHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil {
		s.log.Warn().Err(err).Str("fork", forkFullName).Msg("failed to make fork private")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		s.log.Info().Str("fork", forkFullName).Msg("fork set to private")
	} else {
		respBody, _ := io.ReadAll(resp.Body)
		s.log.Warn().Str("fork", forkFullName).Int("status", resp.StatusCode).Str("body", string(respBody)).Msg("could not make fork private (requires paid GitHub plan)")
	}
}

// SyncFork syncs the fork with its upstream (parent) repository.
func (s *ForkService) SyncFork(ctx context.Context, forkFullName string, branch string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/merge-upstream", forkFullName)
	payload := map[string]string{"branch": branch}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create sync request: %w", err)
	}
	s.setHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("sync request: %w", err)
	}
	defer resp.Body.Close()

	// 200 = synced, 409 = already up to date
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sync failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// PushSuiteWorkflows generates a workflow file for every suite and pushes them
// as a single Verdox commit on top of the upstream branch HEAD. Because the
// tree is built from the upstream tree (not the previous fork tree), any
// previously-deleted suite's workflow file disappears automatically.
// Returns the commit SHA of the Verdox commit.
func (s *ForkService) PushSuiteWorkflows(ctx context.Context, forkFullName, upstreamFullName, branch string, specs []SuiteWorkflowSpec) (string, error) {
	// Get upstream branch HEAD — this becomes the parent of our Verdox commit
	upstreamHeadSHA := s.getBranchHeadSHA(ctx, upstreamFullName, branch)
	if upstreamHeadSHA == "" {
		return "", fmt.Errorf("could not resolve upstream HEAD for %s/%s", upstreamFullName, branch)
	}

	// If no suites, point the fork directly at upstream HEAD (clean state)
	if len(specs) == 0 {
		if err := s.updateRef(ctx, forkFullName, branch, upstreamHeadSHA); err != nil {
			return "", fmt.Errorf("reset fork to upstream: %w", err)
		}
		return upstreamHeadSHA, nil
	}

	// Get the upstream commit's tree SHA (accessible from fork — shared object store)
	upstreamCommit, err := s.getCommitObj(ctx, forkFullName, upstreamHeadSHA)
	if err != nil {
		return "", fmt.Errorf("get upstream commit tree: %w", err)
	}

	// Create a blob + tree entry for each suite's workflow file
	entries := make([]treeEntry, 0, len(specs))
	for _, spec := range specs {
		cfg := spec.WorkflowConfig
		content := GenerateWorkflowYAML(spec.Name, cfg)
		blobSHA, err := s.createBlob(ctx, forkFullName, content)
		if err != nil {
			return "", fmt.Errorf("create blob for suite %q: %w", spec.Name, err)
		}
		entries = append(entries, treeEntry{
			Path:    SuiteWorkflowPath(spec.SuiteID),
			BlobSHA: blobSHA,
		})
	}

	// Create tree: upstream tree + all suite workflow files overlaid
	treeSHA, err := s.createTree(ctx, forkFullName, upstreamCommit.Tree.SHA, entries)
	if err != nil {
		return "", fmt.Errorf("create workflow tree: %w", err)
	}

	// Single Verdox commit on top of upstream HEAD
	msg := fmt.Sprintf("chore(verdox): %d test workflow(s) [skip ci]", len(specs))
	commitSHA, err := s.createCommitObj(ctx, forkFullName, msg, treeSHA, upstreamHeadSHA)
	if err != nil {
		return "", fmt.Errorf("create workflow commit: %w", err)
	}

	// Force-update the fork's branch — replaces any previous Verdox commit
	if err := s.updateRef(ctx, forkFullName, branch, commitSHA); err != nil {
		return "", fmt.Errorf("update fork branch ref: %w", err)
	}

	s.log.Debug().
		Str("fork", forkFullName).
		Str("upstream_head", upstreamHeadSHA[:7]).
		Str("verdox_commit", commitSHA[:7]).
		Int("suites", len(specs)).
		Msg("suite workflows pushed (single commit maintained)")

	return commitSHA, nil
}

// DispatchWorkflow triggers a workflow_dispatch event on the fork.
func (s *ForkService) DispatchWorkflow(ctx context.Context, forkFullName, workflowID, branch string, inputs map[string]string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/actions/workflows/%s/dispatches", forkFullName, workflowID)
	payload := map[string]interface{}{
		"ref":    branch,
		"inputs": inputs,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create dispatch request: %w", err)
	}
	s.setHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("dispatch workflow: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dispatch failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// FetchArtifact downloads a named artifact from a completed workflow run.
func (s *ForkService) FetchArtifact(ctx context.Context, forkFullName string, ghaRunID int64, artifactName string) ([]byte, error) {
	// List artifacts for the run
	url := fmt.Sprintf("https://api.github.com/repos/%s/actions/runs/%d/artifacts", forkFullName, ghaRunID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	s.setHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list artifacts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list artifacts failed (%d)", resp.StatusCode)
	}

	var artifactList struct {
		Artifacts []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"artifacts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&artifactList); err != nil {
		return nil, fmt.Errorf("decode artifacts: %w", err)
	}

	for _, a := range artifactList.Artifacts {
		if a.Name == artifactName {
			// Download artifact
			dlURL := fmt.Sprintf("https://api.github.com/repos/%s/actions/artifacts/%d/zip", forkFullName, a.ID)
			dlReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, dlURL, nil)
			s.setHeaders(dlReq)

			dlResp, err := s.client.Do(dlReq)
			if err != nil {
				return nil, fmt.Errorf("download artifact: %w", err)
			}
			defer dlResp.Body.Close()

			return io.ReadAll(dlResp.Body)
		}
	}

	return nil, fmt.Errorf("artifact %q not found", artifactName)
}

// SetupFork forks the repo (or reuses existing fork), syncs with upstream,
// pushes the Verdox workflow, and updates the repository record.
// Safe to call repeatedly — handles the case where the fork already exists
// (e.g., after a DB wipe).
func (s *ForkService) SetupFork(ctx context.Context, repoID uuid.UUID, repoFullName, defaultBranch string) error {
	// Update status to forking
	if _, err := s.db.ExecContext(ctx,
		"UPDATE repositories SET fork_status = 'forking' WHERE id = $1", repoID); err != nil {
		return fmt.Errorf("update fork status: %w", err)
	}

	// Fork the repo (returns existing fork if already forked — GitHub returns 200)
	forkName, err := s.ForkRepo(ctx, repoFullName)
	if err != nil {
		s.db.ExecContext(ctx, "UPDATE repositories SET fork_status = 'failed' WHERE id = $1", repoID)
		return fmt.Errorf("fork repo: %w", err)
	}

	// Wait briefly for a newly created fork to be ready (GitHub is async).
	// For existing forks this is a no-op delay but harmless.
	time.Sleep(3 * time.Second)

	// Mark fork as ready (with fork_full_name) so RefreshSuiteWorkflows can run
	now := time.Now()
	if _, err := s.db.ExecContext(ctx,
		`UPDATE repositories SET fork_full_name = $1, fork_status = 'ready', fork_synced_at = $2 WHERE id = $3`,
		forkName, now, repoID); err != nil {
		return fmt.Errorf("update repo fork info: %w", err)
	}

	// Push per-suite workflow files (if any suites exist yet).
	// This builds a single Verdox commit on top of upstream HEAD.
	if err := s.RefreshSuiteWorkflows(ctx, repoID); err != nil {
		s.log.Warn().Err(err).Str("fork", forkName).Msg("initial workflow push failed (will retry on suite create)")
	}

	s.log.Info().
		Str("repo", repoFullName).
		Str("fork", forkName).
		Msg("fork setup complete")

	return nil
}

// getBranchHeadSHA fetches the current HEAD SHA of a branch on the fork.
func (s *ForkService) getBranchHeadSHA(ctx context.Context, forkFullName, branch string) string {
	url := fmt.Sprintf("https://api.github.com/repos/%s/git/ref/heads/%s", forkFullName, branch)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	s.setHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return ""
	}
	defer resp.Body.Close()

	var ref struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	json.NewDecoder(resp.Body).Decode(&ref)
	return ref.Object.SHA
}

// --- Git Data API helpers (used by PushWorkflow) ---

// gitCommitInfo represents a GitHub Git commit object.
type gitCommitInfo struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
	Tree    struct {
		SHA string `json:"sha"`
	} `json:"tree"`
	Parents []struct {
		SHA string `json:"sha"`
	} `json:"parents"`
}

// getCommitObj fetches a Git commit object via the Git Data API.
func (s *ForkService) getCommitObj(ctx context.Context, repo, sha string) (*gitCommitInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/git/commits/%s", repo, sha)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	s.setHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get commit %s failed (%d): %s", sha[:7], resp.StatusCode, string(respBody))
	}

	var commit gitCommitInfo
	if err := json.NewDecoder(resp.Body).Decode(&commit); err != nil {
		return nil, err
	}
	return &commit, nil
}

// createBlob creates a Git blob via the Git Data API. Returns the blob SHA.
func (s *ForkService) createBlob(ctx context.Context, repo string, content []byte) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/git/blobs", repo)
	payload := map[string]string{
		"content":  base64.StdEncoding.EncodeToString(content),
		"encoding": "base64",
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	s.setHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create blob failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		SHA string `json:"sha"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.SHA, nil
}

// treeEntry represents a file to include in a Git tree.
type treeEntry struct {
	Path    string
	BlobSHA string
}

// createTree creates a Git tree with multiple files overlaid on a base tree. Returns the tree SHA.
func (s *ForkService) createTree(ctx context.Context, repo, baseTreeSHA string, entries []treeEntry) (string, error) {
	treeItems := make([]map[string]string, len(entries))
	for i, e := range entries {
		treeItems[i] = map[string]string{
			"path": e.Path,
			"mode": "100644",
			"type": "blob",
			"sha":  e.BlobSHA,
		}
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/git/trees", repo)
	payload := map[string]interface{}{
		"base_tree": baseTreeSHA,
		"tree":      treeItems,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	s.setHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create tree failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		SHA string `json:"sha"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.SHA, nil
}

// createCommitObj creates a Git commit object via the Git Data API. Returns the commit SHA.
func (s *ForkService) createCommitObj(ctx context.Context, repo, message, treeSHA, parentSHA string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/git/commits", repo)
	payload := map[string]interface{}{
		"message": message,
		"tree":    treeSHA,
		"parents": []string{parentSHA},
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	s.setHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create commit failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		SHA string `json:"sha"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.SHA, nil
}

// updateRef force-updates a Git branch ref via the Git Data API.
func (s *ForkService) updateRef(ctx context.Context, repo, branch, sha string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/git/refs/heads/%s", repo, branch)
	payload := map[string]interface{}{
		"sha":   sha,
		"force": true,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	s.setHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update ref failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ResyncFork re-pushes all suite workflows on top of the latest upstream HEAD.
// Delegates to RefreshSuiteWorkflows which handles everything.
func (s *ForkService) ResyncFork(ctx context.Context, repoID uuid.UUID, forkFullName, upstreamFullName, defaultBranch string) error {
	return s.RefreshSuiteWorkflows(ctx, repoID)
}

// EnsureBranchWorkflows pushes suite workflow files to a specific branch on
// the fork and returns the new commit SHA. This is called before dispatching
// a workflow so the workflow file exists on the target branch.
func (s *ForkService) EnsureBranchWorkflows(ctx context.Context, repoID uuid.UUID, branch string) (string, error) {
	// Get repo info
	var repo struct {
		ForkFullName   *string `db:"fork_full_name"`
		GithubFullName string  `db:"github_full_name"`
		ForkStatus     string  `db:"fork_status"`
	}
	if err := s.db.GetContext(ctx, &repo,
		"SELECT fork_full_name, github_full_name, fork_status FROM repositories WHERE id = $1", repoID); err != nil {
		return "", fmt.Errorf("get repo: %w", err)
	}
	if repo.ForkStatus != "ready" || repo.ForkFullName == nil || *repo.ForkFullName == "" {
		return "", fmt.Errorf("fork not ready for repo %s", repoID)
	}

	// Get all suites for this repo
	var suites []model.TestSuite
	if err := s.db.SelectContext(ctx, &suites,
		"SELECT * FROM test_suites WHERE repository_id = $1 ORDER BY created_at", repoID); err != nil {
		return "", fmt.Errorf("list suites: %w", err)
	}

	// Build specs
	specs := make([]SuiteWorkflowSpec, 0, len(suites))
	for _, suite := range suites {
		var cfg *model.WorkflowConfig
		wc := model.WorkflowConfig(suite.WorkflowConfig)
		if wc.RunnerOS != "" || len(wc.Services) > 0 || len(wc.SetupSteps) > 0 {
			cfg = &wc
		}
		cmd := "make test"
		if suite.TestCommand != nil && *suite.TestCommand != "" {
			cmd = *suite.TestCommand
		}
		specs = append(specs, SuiteWorkflowSpec{
			SuiteID:        suite.ID,
			Name:           suite.Name,
			TestCommand:    cmd,
			WorkflowConfig: cfg,
		})
	}

	// Push workflows to the specified branch
	commitSHA, err := s.PushSuiteWorkflows(ctx, *repo.ForkFullName, repo.GithubFullName, branch, specs)
	if err != nil {
		return "", fmt.Errorf("push workflows to branch %s: %w", branch, err)
	}

	s.log.Debug().
		Str("fork", *repo.ForkFullName).
		Str("branch", branch).
		Str("commit", commitSHA[:7]).
		Msg("ensured workflows on branch")

	return commitSHA, nil
}

// RefreshSuiteWorkflows queries all suites for a repository, generates a
// workflow file for each, and pushes them as a single Verdox commit on top of
// the upstream HEAD. Called after suite create/update/delete and on resync.
func (s *ForkService) RefreshSuiteWorkflows(ctx context.Context, repoID uuid.UUID) error {
	// Get repo info
	var repo struct {
		ForkFullName   *string `db:"fork_full_name"`
		GithubFullName string  `db:"github_full_name"`
		DefaultBranch  string  `db:"default_branch"`
		ForkStatus     string  `db:"fork_status"`
	}
	if err := s.db.GetContext(ctx, &repo,
		"SELECT fork_full_name, github_full_name, default_branch, fork_status FROM repositories WHERE id = $1", repoID); err != nil {
		return fmt.Errorf("get repo: %w", err)
	}

	// Skip if fork not ready
	if repo.ForkStatus != "ready" || repo.ForkFullName == nil || *repo.ForkFullName == "" {
		return nil
	}

	// Get all suites for this repo
	var suites []model.TestSuite
	if err := s.db.SelectContext(ctx, &suites,
		"SELECT * FROM test_suites WHERE repository_id = $1 ORDER BY created_at", repoID); err != nil {
		return fmt.Errorf("list suites: %w", err)
	}

	// Build specs
	specs := make([]SuiteWorkflowSpec, 0, len(suites))
	for _, suite := range suites {
		var cfg *model.WorkflowConfig
		wc := model.WorkflowConfig(suite.WorkflowConfig)
		if wc.RunnerOS != "" || len(wc.Services) > 0 || len(wc.SetupSteps) > 0 {
			cfg = &wc
		}
		cmd := "make test"
		if suite.TestCommand != nil && *suite.TestCommand != "" {
			cmd = *suite.TestCommand
		}
		specs = append(specs, SuiteWorkflowSpec{
			SuiteID:        suite.ID,
			Name:           suite.Name,
			TestCommand:    cmd,
			WorkflowConfig: cfg,
		})
	}

	// Push all suite workflows as a single commit
	commitSHA, err := s.PushSuiteWorkflows(ctx, *repo.ForkFullName, repo.GithubFullName, repo.DefaultBranch, specs)
	if err != nil {
		return fmt.Errorf("push suite workflows: %w", err)
	}

	// Update fork_head_sha
	if _, err := s.db.ExecContext(ctx,
		"UPDATE repositories SET fork_head_sha = $1, fork_synced_at = now() WHERE id = $2",
		commitSHA, repoID); err != nil {
		return fmt.Errorf("update fork head sha: %w", err)
	}

	s.log.Info().
		Str("fork", *repo.ForkFullName).
		Str("head_sha", commitSHA[:7]).
		Int("suites", len(specs)).
		Msg("suite workflows refreshed")

	return nil
}

// RecoverStuckForks finds repos stuck in "forking" state and re-runs setup.
// Called on server startup to handle interrupted fork operations.
func (s *ForkService) RecoverStuckForks(ctx context.Context) {
	var repos []struct {
		ID            uuid.UUID `db:"id"`
		GithubFullName string   `db:"github_full_name"`
		DefaultBranch  string   `db:"default_branch"`
	}

	err := s.db.SelectContext(ctx, &repos,
		"SELECT id, github_full_name, default_branch FROM repositories WHERE fork_status = 'forking' AND is_active = true")
	if err != nil {
		s.log.Error().Err(err).Msg("failed to query stuck forks")
		return
	}

	if len(repos) == 0 {
		return
	}

	s.log.Info().Int("count", len(repos)).Msg("recovering stuck forks")
	for _, repo := range repos {
		if err := s.SetupFork(ctx, repo.ID, repo.GithubFullName, repo.DefaultBranch); err != nil {
			s.log.Error().Err(err).Str("repo", repo.GithubFullName).Msg("fork recovery failed")
		}
	}
}

func (s *ForkService) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+s.cfg.ServiceAccountPAT)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

// GenerateWorkflowYAML builds a complete GitHub Actions workflow from a WorkflowConfig.
// suiteName is used in the workflow `name:` field shown in the GitHub Actions UI.
// If cfg is nil, sensible defaults are used.
func GenerateWorkflowYAML(suiteName string, cfg *model.WorkflowConfig) []byte {
	var b strings.Builder

	runnerOS := "ubuntu-latest"
	if cfg != nil && cfg.CustomRunner != "" {
		runnerOS = cfg.CustomRunner
	} else if cfg != nil && cfg.RunnerOS != "" {
		runnerOS = cfg.RunnerOS
	}

	// Header — name shows in GitHub Actions UI per suite
	workflowName := "Verdox Test Runner"
	if suiteName != "" {
		workflowName = fmt.Sprintf("Verdox Test Run: %s", suiteName)
	}
	b.WriteString(fmt.Sprintf("name: '%s'\n", workflowName))
	b.WriteString("on:\n")
	b.WriteString("  workflow_dispatch:\n")
	b.WriteString("    inputs:\n")
	b.WriteString("      verdox_run_id:\n")
	b.WriteString("        description: 'Verdox test run ID'\n")
	b.WriteString("        required: true\n")
	b.WriteString("      branch:\n")
	b.WriteString("        description: 'Branch to test'\n")
	b.WriteString("        required: true\n")
	b.WriteString("      commit_hash:\n")
	b.WriteString("        description: 'Commit hash to test'\n")
	b.WriteString("        required: true\n")
	b.WriteString("      callback_url:\n")
	b.WriteString("        description: 'Webhook callback URL'\n")
	b.WriteString("        required: false\n")
	b.WriteString("      test_command:\n")
	b.WriteString("        description: 'Test command to execute'\n")
	b.WriteString("        required: false\n")
	b.WriteString("        default: 'make test'\n")
	b.WriteString("\n")

	// Concurrency
	if cfg != nil && cfg.Concurrency != nil {
		b.WriteString("concurrency:\n")
		b.WriteString(fmt.Sprintf("  group: %s\n", cfg.Concurrency.Group))
		b.WriteString(fmt.Sprintf("  cancel-in-progress: %t\n", cfg.Concurrency.CancelInProgress))
		b.WriteString("\n")
	}

	// Job-level env
	if cfg != nil && len(cfg.EnvVars) > 0 {
		b.WriteString("env:\n")
		for k, v := range cfg.EnvVars {
			b.WriteString(fmt.Sprintf("  %s: '%s'\n", k, v))
		}
		b.WriteString("\n")
	}

	b.WriteString("jobs:\n")
	b.WriteString("  test:\n")
	b.WriteString(fmt.Sprintf("    runs-on: %s\n", runnerOS))

	// Matrix strategy
	if cfg != nil && cfg.Matrix != nil && len(cfg.Matrix.Dimensions) > 0 {
		b.WriteString("    strategy:\n")
		b.WriteString(fmt.Sprintf("      fail-fast: %t\n", cfg.Matrix.FailFast))
		b.WriteString("      matrix:\n")
		for key, values := range cfg.Matrix.Dimensions {
			b.WriteString(fmt.Sprintf("        %s: [", key))
			for i, v := range values {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(fmt.Sprintf("'%s'", strings.TrimSpace(v)))
			}
			b.WriteString("]\n")
		}
	}

	// Services
	if cfg != nil && len(cfg.Services) > 0 {
		b.WriteString("    services:\n")
		for _, svc := range cfg.Services {
			b.WriteString(fmt.Sprintf("      %s:\n", svc.Name))
			b.WriteString(fmt.Sprintf("        image: %s\n", svc.Image))
			if len(svc.Ports) > 0 {
				b.WriteString("        ports:\n")
				for _, port := range svc.Ports {
					b.WriteString(fmt.Sprintf("          - '%s'\n", port))
				}
			}
			if len(svc.Env) > 0 {
				b.WriteString("        env:\n")
				for k, v := range svc.Env {
					b.WriteString(fmt.Sprintf("          %s: '%s'\n", k, v))
				}
			}
			if len(svc.Ports) > 0 {
				b.WriteString("        options: >-\n")
				b.WriteString("          --health-cmd \"exit 0\"\n")
				b.WriteString("          --health-interval 10s\n")
				b.WriteString("          --health-timeout 5s\n")
				b.WriteString("          --health-retries 5\n")
			}
		}
	}

	// Steps
	b.WriteString("    steps:\n")

	// Checkout (always first)
	b.WriteString("      - name: Checkout\n")
	b.WriteString("        uses: actions/checkout@v4\n")
	b.WriteString("        with:\n")
	b.WriteString("          ref: ${{ github.event.inputs.commit_hash }}\n")
	b.WriteString("\n")

	// User-defined setup steps
	if cfg != nil {
		for _, step := range cfg.SetupSteps {
			if step.Uses != "" {
				b.WriteString(fmt.Sprintf("      - name: %s\n", step.Name))
				b.WriteString(fmt.Sprintf("        uses: %s\n", step.Uses))
				if len(step.With) > 0 {
					b.WriteString("        with:\n")
					for k, v := range step.With {
						b.WriteString(fmt.Sprintf("          %s: '%s'\n", k, v))
					}
				}
			} else if step.Run != "" {
				b.WriteString(fmt.Sprintf("      - name: %s\n", step.Name))
				b.WriteString("        run: |\n")
				for _, line := range strings.Split(step.Run, "\n") {
					b.WriteString(fmt.Sprintf("          %s\n", line))
				}
			}
			b.WriteString("\n")
		}
	}

	// Test execution step (always present)
	b.WriteString("      - name: Run tests\n")
	b.WriteString("        id: tests\n")
	b.WriteString("        run: |\n")
	b.WriteString("          set +e\n")
	b.WriteString("          ${{ github.event.inputs.test_command }} 2>&1 | tee test-output.log\n")
	b.WriteString("          echo \"exit_code=$?\" >> $GITHUB_OUTPUT\n")
	b.WriteString("        continue-on-error: true\n")
	b.WriteString("\n")

	// Verdox result collection (always present)
	b.WriteString("      - name: Generate results JSON\n")
	b.WriteString("        if: always()\n")
	b.WriteString("        run: |\n")
	b.WriteString("          cat > verdox-results.json << 'RESULTS_EOF'\n")
	b.WriteString("          {\n")
	b.WriteString("            \"verdox_run_id\": \"${{ github.event.inputs.verdox_run_id }}\",\n")
	b.WriteString("            \"status\": \"${{ steps.tests.outcome }}\",\n")
	b.WriteString("            \"exit_code\": ${{ steps.tests.outputs.exit_code || 1 }}\n")
	b.WriteString("          }\n")
	b.WriteString("          RESULTS_EOF\n")
	b.WriteString("\n")

	b.WriteString("      - name: Upload results artifact\n")
	b.WriteString("        if: always()\n")
	b.WriteString("        uses: actions/upload-artifact@v4\n")
	b.WriteString("        with:\n")
	b.WriteString("          name: verdox-results\n")
	b.WriteString("          path: |\n")
	b.WriteString("            verdox-results.json\n")
	b.WriteString("            test-output.log\n")
	b.WriteString("          retention-days: 7\n")
	b.WriteString("\n")

	b.WriteString("      - name: Callback to Verdox\n")
	b.WriteString("        if: always() && github.event.inputs.callback_url != ''\n")
	b.WriteString("        run: |\n")
	b.WriteString("          curl -s -X POST \"${{ github.event.inputs.callback_url }}\" \\\n")
	b.WriteString("            -H \"Content-Type: application/json\" \\\n")
	b.WriteString("            -d @verdox-results.json || true\n")

	return []byte(b.String())
}
