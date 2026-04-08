package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/sujaykumarsuman/verdox/backend/internal/config"
	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
)

type GenerateSuiteService struct {
	repoRepo repository.RepositoryRepository
	cfg      *config.Config
	client   *http.Client
	log      zerolog.Logger
}

func NewGenerateSuiteService(
	repoRepo repository.RepositoryRepository,
	cfg *config.Config,
	log zerolog.Logger,
) *GenerateSuiteService {
	return &GenerateSuiteService{
		repoRepo: repoRepo,
		cfg:      cfg,
		client:   &http.Client{Timeout: 600 * time.Second},
		log:      log,
	}
}

// ListWorkflowFiles lists GHA workflow files from the upstream repo via GitHub Contents API.
func (s *GenerateSuiteService) ListWorkflowFiles(ctx context.Context, repoID uuid.UUID) (*dto.ListWorkflowFilesResponse, error) {
	pat := s.cfg.ServiceAccountPAT
	if pat == "" {
		return nil, fmt.Errorf("service account PAT not configured")
	}

	repo, err := s.repoRepo.GetByID(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("get repository: %w", err)
	}
	if repo.ForkFullName == nil || repo.ForkStatus != "ready" {
		return nil, fmt.Errorf("fork is not ready for this repository")
	}

	// List workflow files from the fork repo
	url := fmt.Sprintf("https://api.github.com/repos/%s/contents/.github/workflows", *repo.ForkFullName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	s.setGitHubHeaders(req, pat)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// No .github/workflows directory
		return &dto.ListWorkflowFilesResponse{
			RepositoryID: repoID.String(),
			Files:        []dto.WorkflowFile{},
		}, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github API returned %d: %s", resp.StatusCode, string(body))
	}

	var entries []struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("parse github response: %w", err)
	}

	var files []dto.WorkflowFile
	for _, e := range entries {
		if e.Type != "file" {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name))
		if ext != ".yml" && ext != ".yaml" {
			continue
		}
		files = append(files, dto.WorkflowFile{
			Name: e.Name,
			Path: e.Path,
		})
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })

	return &dto.ListWorkflowFilesResponse{
		RepositoryID: repoID.String(),
		Files:        files,
	}, nil
}

// GenerateSuite converts an existing GHA workflow into a Verdox-compatible workflow using GPT.
func (s *GenerateSuiteService) GenerateSuite(ctx context.Context, repoID uuid.UUID, req *dto.GenerateSuiteRequest) (*dto.GenerateSuiteResponse, error) {
	if s.cfg.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("AI import is not configured (VERDOX_OPENAI_API_KEY not set)")
	}

	repo, err := s.repoRepo.GetByID(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("get repository: %w", err)
	}
	if repo.ForkFullName == nil || repo.ForkStatus != "ready" {
		return nil, fmt.Errorf("fork is not ready for this repository")
	}

	// Determine source workflow YAML
	var sourceYAML string
	if req.WorkflowFile != nil && *req.WorkflowFile != "" {
		// Fetch file content from fork repo via GitHub Contents API
		content, err := s.fetchFileFromGitHub(ctx, *repo.ForkFullName, *req.WorkflowFile)
		if err != nil {
			return nil, fmt.Errorf("fetch workflow file: %w", err)
		}
		sourceYAML = content
	} else if req.WorkflowYAML != nil && *req.WorkflowYAML != "" {
		sourceYAML = *req.WorkflowYAML
	} else {
		return nil, fmt.Errorf("either workflow_file or workflow_yaml must be provided")
	}

	// Load AI context from disk
	schema, example, err := s.loadAIContext()
	if err != nil {
		s.log.Warn().Err(err).Msg("failed to load full AI context, proceeding with partial context")
	}

	// Resolve model and timeout
	model := req.Model
	if model == "" {
		model = "gpt-4o"
	}
	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	if timeout <= 0 || timeout > 600*time.Second {
		timeout = 300 * time.Second
	}

	// Build prompt and call OpenAI
	prompt := s.buildGeneratePrompt(repo.GithubFullName, sourceYAML, schema, example)
	resp, err := s.callOpenAI(ctx, prompt, model, timeout)
	if err != nil {
		return nil, fmt.Errorf("AI analysis failed: %w", err)
	}

	return resp, nil
}

// fetchFileFromGitHub reads a file's content from the upstream repo via GitHub Contents API.
func (s *GenerateSuiteService) fetchFileFromGitHub(ctx context.Context, repoFullName, filePath string) (string, error) {
	pat := s.cfg.ServiceAccountPAT
	if pat == "" {
		return "", fmt.Errorf("service account PAT not configured")
	}

	// Validate path does not escape
	cleaned := filepath.Clean(filePath)
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("invalid file path")
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", repoFullName, cleaned)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	s.setGitHubHeaders(req, pat)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("github API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("file not found: %s", filePath)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github API returned %d: %s", resp.StatusCode, string(body))
	}

	var entry struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return "", fmt.Errorf("parse github response: %w", err)
	}

	if entry.Encoding != "base64" {
		return "", fmt.Errorf("unexpected encoding: %s", entry.Encoding)
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(entry.Content, "\n", ""))
	if err != nil {
		return "", fmt.Errorf("decode base64 content: %w", err)
	}

	return string(decoded), nil
}

// detectGoVersion fetches go.mod from the repo and extracts the Go version directive.
func (s *GenerateSuiteService) setGitHubHeaders(req *http.Request, pat string) {
	req.Header.Set("Authorization", "Bearer "+pat)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

func (s *GenerateSuiteService) loadAIContext() (schema, example string, err error) {
	basePath := s.cfg.AIContextPath
	if basePath == "" {
		basePath = "./ai/res/docs"
	}

	readFile := func(path string, maxLen int) string {
		data, err := os.ReadFile(path)
		if err != nil {
			s.log.Debug().Str("path", path).Err(err).Msg("could not read AI context file")
			return ""
		}
		content := string(data)
		if len(content) > maxLen {
			content = content[:maxLen] + "\n... (truncated)"
		}
		return content
	}

	schema = readFile(filepath.Join(basePath, "schema.json"), 15000)
	// Use only the careerdock example (concise, 5 jobs) — system prompt has all templates inline
	example = readFile(filepath.Join(basePath, "verdox-workflows", "careerdock-ci.yml"), 20000)
	return schema, example, nil
}

func (s *GenerateSuiteService) buildGeneratePrompt(repoName, sourceYAML, schema, example string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Repository: %s\n\n", repoName))
	b.WriteString("=== SOURCE WORKFLOW ===\n")
	b.WriteString(sourceYAML)
	b.WriteString("\n\n")

	if schema != "" {
		b.WriteString("=== VERDOX RESULTS SCHEMA (verdox-results.json) ===\n")
		b.WriteString(schema)
		b.WriteString("\n\n")
	}

	if example != "" {
		b.WriteString("=== REFERENCE: Complete Verdox Workflow (5 jobs — Go + Node.js) ===\n")
		b.WriteString(example)
		b.WriteString("\n\n")
	}

	return b.String()
}

func (s *GenerateSuiteService) callOpenAI(ctx context.Context, userPrompt, model string, timeout time.Duration) (*dto.GenerateSuiteResponse, error) {
	systemPrompt := `You convert GitHub Actions workflows into Verdox-compatible workflows.
Verdox runs tests on forked repositories. The workflow runs on the FORK, not the upstream repo.

═══ HARD RULES ═══
• NEVER use reusable workflows (uses: ./.github/workflows/...) — they reference upstream repo internals and will fail on the fork. Inline all test logic directly.
• NEVER reference outputs from jobs that don't exist in your workflow (e.g. needs.get-go-version.outputs...).
• Use actions/checkout@v4, actions/setup-go@v5, actions/upload-artifact@v4, actions/download-artifact@v4 (latest v4 versions).
• Every checkout must use: ref: ${{ github.event.inputs.branch }}
• callback_url input must be required: false (not true).
• No noop jobs or placeholder scripts.
• MATCH configurations from the source workflow — Go version, env vars (GOPRIVATE, etc.), services, working directories, go-tags, build flags, conditional logic. The generated workflow must mirror the source as closely as possible.
• If the source uses a reusable workflow to detect Go version (e.g. reusable-get-go-version.yml), look at how other jobs in the source reference it and hardcode that same version string in the generated workflow.
• If a race detection job in the source has a package-names-command or grep filter to exclude heavy packages, preserve that exact filter in the generated job. Do NOT run -race on all ./... if the source explicitly excluded packages.
• If the source has a build job whose artifact is downloaded by test jobs (e.g. dev-build → upload binary → test jobs download it), replicate this pattern: build job uploads via actions/upload-artifact, test jobs download via actions/download-artifact.
• Keep ALL jobs including enterprise-conditional ones (if: endsWith(github.repository, '-enterprise')). Admins may configure the necessary secrets on the fork. Preserve the conditional logic as-is.
• PREREQUISITES: When expanding a reusable workflow into inline steps, you MUST include all tool installation steps that the reusable workflow handled internally. Common examples:
  - Lint jobs that call "make lint" or "golangci-lint run" NEED golangci-lint installed first. Use "golangci/golangci-lint-action@v7" or "go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest".
  - Protobuf jobs need protoc and proto plugins installed (e.g. "make proto-tools").
  - Node.js jobs need "npm ci" or "yarn install" before running scripts.
  - Python jobs need "pip install" before running test commands.
  - If the source references a Makefile target, trace what tools that target requires and ensure they are installed in prior steps.
  Think about what a fresh ubuntu-latest runner has (Go, Python3, Node, make, git) and what it does NOT have (golangci-lint, protoc, custom tools). Install anything missing.

═══ WORKFLOW HEADER (use exactly) ═══
name: "verdox: <original workflow name>"

on:
  workflow_dispatch:
    inputs:
      verdox_run_id:
        description: 'Verdox test run ID'
        required: true
      branch:
        description: 'Branch to test'
        required: true
      commit_hash:
        description: 'Commit hash to test'
        required: true
      callback_url:
        description: 'Webhook callback URL'
        required: false

permissions:
  contents: read

═══ GO TEST JOB TEMPLATE ═══
For every job that runs Go tests, use this exact pattern:

    steps:
      - uses: actions/checkout@v4
        with:
          ref: ${{ github.event.inputs.branch }}
      - uses: actions/setup-go@v5
        with:
          go-version: "<version from source or env>"
      - name: Run tests
        working-directory: <dir if needed>
        run: |
          set +e
          go test -json -count=1 -timeout <timeout> ./... 2>&1 | tee test-output.json
          exit 0
      - name: Parse test results
        if: always()
        working-directory: <dir if needed>
        run: |
          python3 << 'VERDOX_PARSER'
          import json, sys
          tests, pkg_durations = {}, {}
          try:
              with open("test-output.json") as f:
                  for line in f:
                      line = line.strip()
                      if not line: continue
                      try: ev = json.loads(line)
                      except: continue
                      test, pkg, action = ev.get("Test",""), ev.get("Package",""), ev.get("Action","")
                      if not test and pkg and action in ("pass","fail"): pkg_durations[pkg] = ev.get("Elapsed",0); continue
                      if not test: continue
                      key = f"{pkg}/{test}"
                      if key not in tests: tests[key] = {"name":test,"package":pkg,"status":"","duration_s":0,"output":[],"error":""}
                      t = tests[key]
                      if action == "output": t["output"].append(ev.get("Output",""))
                      elif action == "pass": t["status"]="passed"; t["duration_s"]=ev.get("Elapsed",0)
                      elif action == "fail": t["status"]="failed"; t["duration_s"]=ev.get("Elapsed",0); t["error"]="".join(t["output"])[-3000:]
                      elif action == "skip": t["status"]="skipped"
          except Exception as e: print(f"Warning: {e}", file=sys.stderr)
          packages = {}
          for key, t in tests.items():
              if not t["status"]: continue
              packages.setdefault(t["package"],[]).append(t)
          result = {"packages": {p: [{"name":c["name"],"status":c["status"],"duration_seconds":c["duration_s"],"error_message":c["error"]} for c in cs] for p, cs in packages.items()}, "pkg_durations": pkg_durations}
          with open("verdox-test-results.json","w") as f: json.dump(result, f)
          print(f"Verdox: {sum(len(v) for v in result['packages'].values())} tests across {len(result['packages'])} packages")
          VERDOX_PARSER
      - uses: actions/upload-artifact@v4
        if: always()
        with:
          name: <job-id>-results
          path: <working-dir/>verdox-test-results.json

IMPORTANT: "set +e" and "exit 0" ensure the test step doesn't fail the job, so the parser always runs. The parser must be embedded inline — no external scripts.

═══ LINT / BUILD JOB PATTERN ═══
Lint and build jobs do NOT need the parser or artifact upload. Keep them simple. Their pass/fail status is captured by the verdox-report job via needs.<job>.result.

═══ VERDOX-REPORT JOB (required, always last) ═══
Must include ALL other jobs in "needs". Must use "if: always()". Use this exact structure:

  verdox-report:
    name: Verdox Report
    runs-on: ubuntu-latest
    if: always()
    needs: [<all other job ids>]
    steps:
      - name: Download all artifacts
        uses: actions/download-artifact@v4
        with:
          path: artifacts
        continue-on-error: true
      - name: Build Verdox report
        env:
          VERDOX_RUN_ID: ${{ github.event.inputs.verdox_run_id }}
          VERDOX_REPO: ${{ github.repository }}
          VERDOX_BRANCH: ${{ github.event.inputs.branch }}
          VERDOX_COMMIT: ${{ github.event.inputs.commit_hash }}
          VERDOX_SUITES: |
            [
              {"id": "<job-id>", "name": "<Human Name>", "type": "<unit|lint|build|race|...>", "result": "${{ needs.<job-id>.result }}", "results_file": "artifacts/<job-id>-results/verdox-test-results.json"},
              ...for lint/build jobs omit results_file...
              {"id": "<lint-job-id>", "name": "<Lint Name>", "type": "lint", "result": "${{ needs.<lint-job-id>.result }}"}
            ]
        run: |
          python3 << 'VERDOX_REPORT'
          import json, os, sys
          from datetime import datetime, timezone
          run_id = os.environ.get("VERDOX_RUN_ID", "")
          repo = os.environ.get("VERDOX_REPO", "")
          branch = os.environ.get("VERDOX_BRANCH", "")
          commit = os.environ.get("VERDOX_COMMIT", "")
          suites_config = json.loads(os.environ.get("VERDOX_SUITES", "[]"))
          suites = []
          all_cases = all_passed = all_failed = all_skipped = 0
          all_duration = 0.0
          for sc in suites_config:
              suite_id, suite_name, suite_type = sc["id"], sc["name"], sc["type"]
              job_result = sc.get("result", "success")
              results_file = sc.get("results_file", "")
              tests = []; s_passed = s_failed = s_skipped = s_total = 0; s_duration = 0.0
              if results_file and os.path.exists(results_file):
                  with open(results_file) as f: data = json.load(f)
                  pkg_dur = data.get("pkg_durations", {})
                  for pkg, cases in data.get("packages", {}).items():
                      pkg_slug = pkg.replace("/","-").replace(".","-")[-60:]
                      pkg_name = pkg.split("/")[-1] if "/" in pkg else pkg
                      pkg_elapsed = pkg_dur.get(pkg, 0)
                      test_cases = []; t_p = t_f = t_s = 0; t_dur = 0.0
                      for c in cases:
                          st = c.get("status","unknown"); dur = c.get("duration_seconds",0)
                          tc = {"case_id":c["name"],"name":c["name"],"status":st,"duration_seconds":dur}
                          if c.get("error_message"): tc["error_message"] = c["error_message"]
                          test_cases.append(tc)
                          if st=="passed": t_p+=1
                          elif st=="failed": t_f+=1
                          elif st=="skipped": t_s+=1
                          t_dur += dur
                      if pkg_elapsed > t_dur: t_dur = pkg_elapsed
                      t_total = len(test_cases); t_status = "failed" if t_f>0 else "passed"
                      t_rate = round(t_p/t_total*100,2) if t_total>0 else 0
                      tests.append({"test_id":pkg_slug,"name":pkg_name,"package":pkg,"status":t_status,"stats":{"total":t_total,"passed":t_p,"failed":t_f,"skipped":t_s,"duration_seconds":round(t_dur,2),"pass_rate":t_rate},"cases":test_cases})
                      s_passed+=t_p; s_failed+=t_f; s_skipped+=t_s; s_total+=t_total; s_duration+=t_dur
              else:
                  st = "passed" if job_result=="success" else "failed"
                  tests.append({"test_id":suite_id,"name":suite_name,"package":"","status":st,"stats":{"total":1,"passed":1 if st=="passed" else 0,"failed":1 if st=="failed" else 0,"skipped":0,"duration_seconds":0,"pass_rate":100 if st=="passed" else 0},"cases":[{"case_id":suite_id,"name":suite_name,"status":st,"duration_seconds":0}]})
                  s_total=1; s_passed=1 if st=="passed" else 0; s_failed=1 if st=="failed" else 0
              suite_status = "failed" if s_failed>0 else "passed"
              suite_rate = round(s_passed/s_total*100,2) if s_total>0 else 0
              suites.append({"job_id":suite_id,"name":suite_name,"type":suite_type,"status":suite_status,"stats":{"total":s_total,"passed":s_passed,"failed":s_failed,"skipped":s_skipped,"duration_seconds":round(s_duration,2),"pass_rate":suite_rate},"tests":tests})
              all_cases+=s_total; all_passed+=s_passed; all_failed+=s_failed; all_skipped+=s_skipped; all_duration+=s_duration
          all_rate = round(all_passed/all_cases*100,2) if all_cases>0 else 0
          payload = {"repo":repo,"run_id":run_id,"branch":branch,"commit_sha":commit,"timestamp":datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),"summary":{"total_jobs":len(suites),"total_tests":sum(len(s["tests"]) for s in suites),"total_cases":all_cases,"passed":all_passed,"failed":all_failed,"skipped":all_skipped,"duration_seconds":round(all_duration,2),"pass_rate":all_rate},"jobs":suites}
          with open("verdox-results.json","w") as f: json.dump(payload, f, indent=2)
          print(f"Verdox: {len(suites)} jobs, {all_cases} cases ({all_passed}P/{all_failed}F/{all_skipped}S)")
          VERDOX_REPORT
      - uses: actions/upload-artifact@v4
        if: always()
        with:
          name: verdox-results
          path: verdox-results.json
          retention-days: 7
      - name: Callback to Verdox
        if: always() && github.event.inputs.callback_url != ''
        run: |
          curl -s -X POST "${{ github.event.inputs.callback_url }}" \
            -H "Content-Type: application/json" \
            -d @verdox-results.json || true

═══ ARTIFACT NAMING (CRITICAL — must be exact) ═══
Every test job uploads its parsed results with a PINNED artifact name derived from the job id:
  artifact name:  <job-id>-results           (e.g. go-test-ce-results)
  file inside:    verdox-test-results.json    (always this exact filename)

The verdox-report job downloads ALL artifacts into artifacts/ directory.
The results_file in VERDOX_SUITES MUST follow this exact pattern:
  artifacts/<job-id>-results/verdox-test-results.json

Example chain for a job with id "go-test-ce":
  1. Job uploads:   name: go-test-ce-results, path: verdox-test-results.json
  2. VERDOX_SUITES: "results_file": "artifacts/go-test-ce-results/verdox-test-results.json"
These three names (job-id, artifact name, results_file path) MUST be consistent. If they mismatch, results are silently lost.

═══ KEY DECISIONS ═══
• For each test job in the source that uses a reusable workflow, expand it into a direct go test invocation with the VERDOX_PARSER.
• Keep services (postgres, redis, etc.) from the source workflow.
• Keep env vars (GOPRIVATE, etc.) and working-directory from the source workflow.
• Match Go version, go-tags, build flags, and conditional logic exactly from the source workflow.
• For race detection jobs, keep -race flag, keep any package exclusion filters, and add -json.
• Keep enterprise-conditional jobs with their original "if" conditions intact.
• If a build job produces an artifact consumed by test jobs, preserve the upload/download-artifact chain so test jobs have the built binary.

═══ RESPONSE FORMAT ═══
Respond with a JSON object containing exactly:
- name: suggested suite name (string)
- type: one of "unit", "integration", "e2e", "lint", "build", "race", "compatibility", "load"
- timeout_seconds: integer between 300 and 3600
- workflow_yaml: the complete workflow YAML as a string

ONLY valid JSON. No markdown fences.`

	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"response_format": map[string]string{"type": "json_object"},
		"temperature":     0.2,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	// Per-request timeout (separate from the global HTTP client timeout)
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.OpenAIAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	content := result.Choices[0].Message.Content

	// Parse response — try direct object first, then wrapped
	var importResp dto.GenerateSuiteResponse
	if err := json.Unmarshal([]byte(content), &importResp); err != nil {
		var wrapper struct {
			Result dto.GenerateSuiteResponse `json:"result"`
		}
		if err := json.Unmarshal([]byte(content), &wrapper); err != nil {
			return nil, fmt.Errorf("failed to parse AI response: %w", err)
		}
		importResp = wrapper.Result
	}

	// Validate required fields
	if importResp.WorkflowYAML == "" {
		return nil, fmt.Errorf("AI response missing workflow_yaml")
	}
	if importResp.Name == "" {
		importResp.Name = "Imported Suite"
	}
	if importResp.Type == "" {
		importResp.Type = "unit"
	}
	if importResp.TimeoutSeconds < 30 || importResp.TimeoutSeconds > 3600 {
		importResp.TimeoutSeconds = 600
	}

	return &importResp, nil
}
