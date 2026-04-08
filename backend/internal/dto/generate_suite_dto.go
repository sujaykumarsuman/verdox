package dto

// GenerateSuiteRequest — exactly one of WorkflowFile or WorkflowYAML must be set.
type GenerateSuiteRequest struct {
	WorkflowFile   *string `json:"workflow_file"`
	WorkflowYAML   *string `json:"workflow_yaml"`
	Model          string  `json:"model"`
	TimeoutSeconds int     `json:"timeout_seconds"`
}

// GenerateSuiteResponse contains pre-fill data for the suite creation form.
type GenerateSuiteResponse struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	TimeoutSeconds int    `json:"timeout_seconds"`
	WorkflowYAML   string `json:"workflow_yaml"`
}

// WorkflowFile represents a GHA workflow file found in the repository.
type WorkflowFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// ListWorkflowFilesResponse is returned by the workflow listing endpoint.
type ListWorkflowFilesResponse struct {
	RepositoryID string         `json:"repository_id"`
	Files        []WorkflowFile `json:"files"`
}
