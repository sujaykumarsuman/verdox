package runner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type ParsedResult struct {
	TestName     string
	Status       string // "pass", "fail", "skip", "error"
	DurationMs   int64
	ErrorMessage string
	LogOutput    string
}

type Parser struct{}

func NewParser() *Parser { return &Parser{} }

// Parse tries each parser in order and falls back to a single result.
func (p *Parser) Parse(output string, testType string, exitCode int64) []ParsedResult {
	// Try Go JSON first (NDJSON with {"Action": ...})
	if strings.Contains(output, `"Action"`) {
		if results, err := p.ParseGoJSON(output); err == nil && len(results) > 0 {
			return results
		}
	}

	// Try pytest format
	if strings.Contains(output, "PASSED") || strings.Contains(output, "FAILED") {
		if results, err := p.ParsePytest(output); err == nil && len(results) > 0 {
			return results
		}
	}

	// Try Jest format (Unicode indicators)
	if strings.ContainsAny(output, "✓✔✗✘×○◌") {
		if results, err := p.ParseJest(output); err == nil && len(results) > 0 {
			return results
		}
	}

	return p.Fallback(output, exitCode)
}

// goTestEvent represents a single JSON line from `go test -json`.
type goTestEvent struct {
	Action  string  `json:"Action"`
	Test    string  `json:"Test"`
	Package string  `json:"Package"`
	Output  string  `json:"Output"`
	Elapsed float64 `json:"Elapsed"`
}

// ParseGoJSON parses Go test NDJSON output.
func (p *Parser) ParseGoJSON(output string) ([]ParsedResult, error) {
	type testAccum struct {
		status     string
		durationMs int64
		output     strings.Builder
	}
	tests := make(map[string]*testAccum)

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		var event goTestEvent
		if json.Unmarshal([]byte(line), &event) != nil {
			continue
		}
		if event.Test == "" {
			continue
		}

		key := event.Test
		if _, exists := tests[key]; !exists {
			tests[key] = &testAccum{}
		}
		acc := tests[key]

		switch event.Action {
		case "output":
			acc.output.WriteString(event.Output)
		case "pass":
			acc.status = "pass"
			acc.durationMs = int64(event.Elapsed * 1000)
		case "fail":
			acc.status = "fail"
			acc.durationMs = int64(event.Elapsed * 1000)
		case "skip":
			acc.status = "skip"
		}
	}

	var results []ParsedResult
	for name, acc := range tests {
		if acc.status == "" {
			continue
		}
		r := ParsedResult{
			TestName:   name,
			Status:     acc.status,
			DurationMs: acc.durationMs,
			LogOutput:  acc.output.String(),
		}
		if acc.status == "fail" {
			r.ErrorMessage = extractGoFailMessage(acc.output.String())
		}
		results = append(results, r)
	}

	return results, nil
}

func extractGoFailMessage(output string) string {
	lines := strings.Split(output, "\n")
	var errLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "---") || trimmed == "" {
			continue
		}
		errLines = append(errLines, trimmed)
	}
	if len(errLines) > 10 {
		errLines = errLines[:10]
	}
	return strings.Join(errLines, "\n")
}

var pytestPattern = regexp.MustCompile(`^(.+?::\S+)\s+(PASSED|FAILED|SKIPPED|ERROR)`)
var pytestDurationPattern = regexp.MustCompile(`\[(\d+)%\]`)

// ParsePytest parses pytest verbose output.
func (p *Parser) ParsePytest(output string) ([]ParsedResult, error) {
	var results []ParsedResult

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		match := pytestPattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		status := strings.ToLower(match[2])
		switch status {
		case "passed":
			status = "pass"
		case "failed":
			status = "fail"
		case "skipped":
			status = "skip"
		case "error":
			status = "error"
		}

		results = append(results, ParsedResult{
			TestName: match[1],
			Status:   status,
		})
	}

	return results, nil
}

var jestPassPattern = regexp.MustCompile(`^\s*[✓✔]\s+(.+?)(?:\s+\((\d+)\s*m?s\))?$`)
var jestFailPattern = regexp.MustCompile(`^\s*[✗✘×]\s+(.+?)(?:\s+\((\d+)\s*m?s\))?$`)
var jestSkipPattern = regexp.MustCompile(`^\s*[○◌]\s+(?:skipped:?\s*)?(.+)$`)

// ParseJest parses Jest/Mocha test output with Unicode indicators.
func (p *Parser) ParseJest(output string) ([]ParsedResult, error) {
	var results []ParsedResult

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		if match := jestPassPattern.FindStringSubmatch(line); match != nil {
			r := ParsedResult{TestName: strings.TrimSpace(match[1]), Status: "pass"}
			if match[2] != "" {
				if ms, err := strconv.ParseInt(match[2], 10, 64); err == nil {
					r.DurationMs = ms
				}
			}
			results = append(results, r)
			continue
		}

		if match := jestFailPattern.FindStringSubmatch(line); match != nil {
			r := ParsedResult{TestName: strings.TrimSpace(match[1]), Status: "fail"}
			if match[2] != "" {
				if ms, err := strconv.ParseInt(match[2], 10, 64); err == nil {
					r.DurationMs = ms
				}
			}
			results = append(results, r)
			continue
		}

		if match := jestSkipPattern.FindStringSubmatch(line); match != nil {
			results = append(results, ParsedResult{
				TestName: strings.TrimSpace(match[1]),
				Status:   "skip",
			})
		}
	}

	return results, nil
}

// Fallback creates a single result when no parser matches.
func (p *Parser) Fallback(output string, exitCode int64) []ParsedResult {
	status := "pass"
	var errMsg string
	if exitCode != 0 {
		status = "fail"
		errMsg = fmt.Sprintf("Process exited with code %d", exitCode)
	}
	return []ParsedResult{{
		TestName:     "full_run",
		Status:       status,
		DurationMs:   0,
		ErrorMessage: errMsg,
		LogOutput:    output,
	}}
}
