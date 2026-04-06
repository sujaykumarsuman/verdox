package dto

type DiscoverySuggestion struct {
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	ExecutionMode string            `json:"execution_mode"`
	DockerImage   string            `json:"docker_image,omitempty"`
	TestCommand   string            `json:"test_command,omitempty"`
	GHAWorkflow   string            `json:"gha_workflow,omitempty"`
	Confidence    float64           `json:"confidence"`
	Reasoning     string            `json:"reasoning"`
}

type DiscoveryResponse struct {
	RepositoryID string                `json:"repository_id"`
	Suggestions  []DiscoverySuggestion `json:"suggestions"`
	ScannedAt    string                `json:"scanned_at"`
}

type DiscoveryRequest struct {
	// empty for now — could add params like scan depth, frameworks filter, etc.
}
