package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/sujaykumarsuman/verdox/backend/internal/config"
	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
)

type DiscoveryService struct {
	repoRepo repository.RepositoryRepository
	cfg      *config.Config
	client   *http.Client
	log      zerolog.Logger
}

func NewDiscoveryService(
	repoRepo repository.RepositoryRepository,
	cfg *config.Config,
	log zerolog.Logger,
) *DiscoveryService {
	return &DiscoveryService{
		repoRepo: repoRepo,
		cfg:      cfg,
		client:   &http.Client{Timeout: 60 * time.Second},
		log:      log,
	}
}

func (s *DiscoveryService) Discover(ctx context.Context, repoID uuid.UUID) (*dto.DiscoveryResponse, error) {
	if s.cfg.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("AI discovery is not configured (VERDOX_OPENAI_API_KEY not set)")
	}

	repo, err := s.repoRepo.GetByID(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("get repository: %w", err)
	}

	if repo.LocalPath == nil {
		return nil, fmt.Errorf("repository has no local clone")
	}

	// Scan repository file tree
	fileTree := s.scanFileTree(*repo.LocalPath)

	// Read key files for context
	keyFiles := s.readKeyFiles(*repo.LocalPath)

	// Build prompt
	prompt := s.buildPrompt(repo.GithubFullName, fileTree, keyFiles)

	// Call OpenAI API
	suggestions, err := s.callOpenAI(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI analysis failed: %w", err)
	}

	return &dto.DiscoveryResponse{
		RepositoryID: repoID.String(),
		Suggestions:  suggestions,
		ScannedAt:    time.Now().Format(time.RFC3339),
	}, nil
}

func (s *DiscoveryService) scanFileTree(localPath string) string {
	var builder strings.Builder
	filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(localPath, path)
		if strings.HasPrefix(rel, ".git") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(rel, "node_modules") || strings.HasPrefix(rel, "vendor") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.IsDir() {
			builder.WriteString(rel + "\n")
		}
		return nil
	})
	// Limit to 5000 chars
	result := builder.String()
	if len(result) > 5000 {
		result = result[:5000] + "\n... (truncated)"
	}
	return result
}

func (s *DiscoveryService) readKeyFiles(localPath string) string {
	keyPatterns := []string{
		"package.json", "go.mod", "requirements.txt", "Pipfile", "Cargo.toml",
		"Makefile", "Dockerfile", ".github/workflows/*.yml", ".github/workflows/*.yaml",
		"verdox.yaml", "jest.config.js", "jest.config.ts", "pytest.ini", "setup.py",
	}

	var builder strings.Builder
	for _, pattern := range keyPatterns {
		matches, _ := filepath.Glob(filepath.Join(localPath, pattern))
		for _, match := range matches {
			data, err := os.ReadFile(match)
			if err != nil {
				continue
			}
			rel, _ := filepath.Rel(localPath, match)
			content := string(data)
			if len(content) > 2000 {
				content = content[:2000] + "\n... (truncated)"
			}
			builder.WriteString(fmt.Sprintf("=== %s ===\n%s\n\n", rel, content))
		}
	}

	result := builder.String()
	if len(result) > 10000 {
		result = result[:10000]
	}
	return result
}

func (s *DiscoveryService) buildPrompt(repoName, fileTree, keyFiles string) string {
	return fmt.Sprintf(`Analyze this repository and identify test suites that can be configured in Verdox (a test orchestration platform).

Repository: %s

File tree:
%s

Key configuration files:
%s

For each test suite you identify, provide:
- name: A descriptive name (e.g., "Unit Tests", "Lint Check", "E2E Tests")
- type: The type (e.g., "unit", "integration", "e2e", "lint", "smoke")
- execution_mode: "container" for tests that can run in a Docker container, or "gha" for tests that need GitHub Actions
- docker_image: The Docker image to use (only for container mode)
- test_command: The shell command to run the tests (only for container mode)
- gha_workflow: The workflow file name (only for gha mode, e.g., "test.yml")
- confidence: A number 0-1 indicating how confident you are
- reasoning: Brief explanation of why you suggest this suite

Respond with a JSON array of suggestions. Example:
[
  {
    "name": "Unit Tests",
    "type": "unit",
    "execution_mode": "container",
    "docker_image": "golang:1.26-alpine",
    "test_command": "go test -v -json ./...",
    "confidence": 0.95,
    "reasoning": "Go project with test files in multiple packages"
  }
]`, repoName, fileTree, keyFiles)
}

func (s *DiscoveryService) callOpenAI(ctx context.Context, prompt string) ([]dto.DiscoverySuggestion, error) {
	reqBody := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "system", "content": "You are a test infrastructure expert. Analyze repositories and suggest test suite configurations. Respond only with valid JSON."},
			{"role": "user", "content": prompt},
		},
		"response_format": map[string]string{"type": "json_object"},
		"temperature":     0.2,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
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

	// Try parsing as array directly
	var suggestions []dto.DiscoverySuggestion
	if err := json.Unmarshal([]byte(content), &suggestions); err != nil {
		// Try parsing as object with "suggestions" key
		var wrapper struct {
			Suggestions []dto.DiscoverySuggestion `json:"suggestions"`
		}
		if err := json.Unmarshal([]byte(content), &wrapper); err != nil {
			return nil, fmt.Errorf("failed to parse AI response: %w", err)
		}
		suggestions = wrapper.Suggestions
	}

	return suggestions, nil
}
