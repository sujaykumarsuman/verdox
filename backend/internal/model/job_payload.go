package model

// JobPayload is the JSON structure pushed to and popped from Redis job queues.
type JobPayload struct {
	TestRunID          string            `json:"test_run_id"`
	TestSuiteID        string            `json:"test_suite_id"`
	RepoID             string            `json:"repo_id"`
	RepositoryFullName string            `json:"repository_full_name"`
	LocalPath          string            `json:"local_path"`
	DefaultBranch      string            `json:"default_branch"`
	Branch             string            `json:"branch"`
	CommitHash         string            `json:"commit_hash"`
	SuiteType          string            `json:"suite_type"`
	ExecutionMode      string            `json:"execution_mode"`
	DockerImage        string            `json:"docker_image"`
	TestCommand        string            `json:"test_command"`
	GHAWorkflowID      string            `json:"gha_workflow_id"`
	ConfigPath         string            `json:"config_path"`
	TimeoutSeconds     int               `json:"timeout_seconds"`
	EnvVars            map[string]string `json:"env_vars"`
}
