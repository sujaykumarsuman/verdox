package runner

import (
	"encoding/json"
	"fmt"
)

// VerdoxResultsFile is the standard results contract.
// Test scripts write this to /tmp/verdox-results.json inside the container.
type VerdoxResultsFile struct {
	Version int                  `json:"version"`
	Status  string               `json:"status"`
	Summary VerdoxResultsSummary `json:"summary"`
	Results []VerdoxTestResult   `json:"results"`
}

type VerdoxResultsSummary struct {
	Total      int   `json:"total"`
	Passed     int   `json:"passed"`
	Failed     int   `json:"failed"`
	Skipped    int   `json:"skipped"`
	DurationMs int64 `json:"duration_ms"`
}

type VerdoxTestResult struct {
	TestName     string `json:"test_name"`
	Status       string `json:"status"` // "pass", "fail", "skip", "error"
	DurationMs   int64  `json:"duration_ms"`
	ErrorMessage string `json:"error_message,omitempty"`
	LogOutput    string `json:"log_output,omitempty"`
}

// ResultsFilePath is the path inside the container where test scripts
// should write their results JSON.
const ResultsFilePath = "/tmp/verdox-results.json"

// ParseVerdoxResultsJSON parses the standard .verdox/results.json format
// into the common ParsedResult slice.
func ParseVerdoxResultsJSON(data []byte) ([]ParsedResult, error) {
	var file VerdoxResultsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse results JSON: %w", err)
	}

	if len(file.Results) == 0 {
		return nil, fmt.Errorf("results file contains no test results")
	}

	results := make([]ParsedResult, len(file.Results))
	for i, r := range file.Results {
		results[i] = ParsedResult{
			TestName:     r.TestName,
			Status:       r.Status,
			DurationMs:   r.DurationMs,
			ErrorMessage: r.ErrorMessage,
			LogOutput:    r.LogOutput,
		}
	}

	return results, nil
}

// ParseWithResultsFile tries the standard results file first,
// then falls back to stdout parsing.
func (p *Parser) ParseWithResultsFile(resultsJSON []byte, stdout string, exitCode int64) []ParsedResult {
	if len(resultsJSON) > 0 {
		if results, err := ParseVerdoxResultsJSON(resultsJSON); err == nil && len(results) > 0 {
			return results
		}
	}

	// Fall back to stdout parsing
	return p.Parse(stdout, "", exitCode)
}
