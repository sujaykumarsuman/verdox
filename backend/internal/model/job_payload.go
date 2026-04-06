package model

// JobPayload is the JSON structure pushed to and popped from Redis job queues.
type JobPayload struct {
	TestRunID          string `json:"test_run_id"`
	TestSuiteID        string `json:"test_suite_id"`
	RepoID             string `json:"repo_id"`
	RepositoryFullName string `json:"repository_full_name"`
	LocalPath          string `json:"local_path"`
	DefaultBranch      string `json:"default_branch"`
	Branch             string `json:"branch"`
	CommitHash         string `json:"commit_hash"`
	TestType           string `json:"test_type"`
	ConfigPath         string `json:"config_path"`
	TimeoutSeconds     int    `json:"timeout_seconds"`
}
