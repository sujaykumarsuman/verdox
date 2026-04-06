package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

var (
	ErrInvalidPAT   = errors.New("invalid or expired GitHub PAT")
	ErrRepoNotFound = errors.New("GitHub repository not found")
	ErrNoRepoAccess = errors.New("PAT does not have access to this repository")
	ErrGitHubAPI    = errors.New("GitHub API error")
)

type GitHubRepoInfo struct {
	ID            int64  `json:"id"`
	FullName      string `json:"full_name"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	DefaultBranch string `json:"default_branch"`
}

type GitHubCommit struct {
	SHA     string `json:"sha"`
	Message string
	Author  string
	Date    string
}

type GitHubService struct {
	client *http.Client
	log    zerolog.Logger
}

func NewGitHubService(log zerolog.Logger) *GitHubService {
	return &GitHubService{
		client: &http.Client{Timeout: 15 * time.Second},
		log:    log,
	}
}

// ValidatePAT validates a GitHub PAT by calling GET /user.
// Returns the GitHub username associated with the PAT.
func (s *GitHubService) ValidatePAT(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("github api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", ErrInvalidPAT
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: status %d", ErrGitHubAPI, resp.StatusCode)
	}

	var user struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return user.Login, nil
}

// GetRepository fetches repository metadata from GitHub.
func (s *GitHubService) GetRepository(ctx context.Context, token, owner, repo string) (*GitHubRepoInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github api: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, ErrRepoNotFound
	case http.StatusForbidden:
		return nil, ErrNoRepoAccess
	default:
		return nil, fmt.Errorf("%w: status %d", ErrGitHubAPI, resp.StatusCode)
	}

	var info GitHubRepoInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &info, nil
}

// GetCommits fetches commits for a branch from GitHub API.
func (s *GitHubService) GetCommits(ctx context.Context, token, owner, repo, branch string, perPage int) ([]GitHubCommit, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits?sha=%s&per_page=%d",
		owner, repo, branch, perPage)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrGitHubAPI, resp.StatusCode)
	}

	var raw []struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Name string `json:"name"`
				Date string `json:"date"`
			} `json:"author"`
		} `json:"commit"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	commits := make([]GitHubCommit, len(raw))
	for i, c := range raw {
		commits[i] = GitHubCommit{
			SHA:     c.SHA,
			Message: c.Commit.Message,
			Author:  c.Commit.Author.Name,
			Date:    c.Commit.Author.Date,
		}
	}

	return commits, nil
}

// ParseGitHubURL extracts owner and repo from a GitHub URL.
func ParseGitHubURL(rawURL string) (owner, repo string, err error) {
	rawURL = strings.TrimSpace(rawURL)
	rawURL = strings.TrimSuffix(rawURL, "/")
	rawURL = strings.TrimSuffix(rawURL, ".git")

	// Remove protocol
	path := rawURL
	for _, prefix := range []string{"https://github.com/", "http://github.com/"} {
		if strings.HasPrefix(strings.ToLower(path), strings.ToLower(prefix)) {
			path = rawURL[len(prefix):]
			break
		}
	}

	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid GitHub URL: %s", rawURL)
	}

	return parts[0], parts[1], nil
}
