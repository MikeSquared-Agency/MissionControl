package api

// --- Request types ---

// GateActionRequest is used for gate approve/reject
type GateActionRequest struct {
	Reason string `json:"reason,omitempty"`
}

// ChatRequest is the request for POST /api/chat
type ChatRequest struct {
	Message string `json:"message"`
}

// SpawnWorkerRequest is the request for POST /api/workers/spawn
type SpawnWorkerRequest struct {
	Persona string `json:"persona,omitempty"`
	Zone    string `json:"zone,omitempty"`
	Task    string `json:"task,omitempty"`
}

// CreateTaskRequest is the request for POST /api/tasks
type CreateTaskRequest struct {
	Title string `json:"title"`
	Stage string `json:"stage"`
	Zone  string `json:"zone"`
}

// UpdateTaskRequest is the request for PATCH /api/tasks/{id}
type UpdateTaskRequest struct {
	Status string `json:"status,omitempty"`
	Stage  string `json:"stage,omitempty"`
}

// TaskDepRequest is the request for POST /api/tasks/{id}/dependencies
type TaskDepRequest struct {
	Action string `json:"action"` // "add" or "remove"
	DepID  string `json:"dep_id"`
}

// StageOverrideRequest is the request for POST /api/stages/override
type StageOverrideRequest struct {
	Stage string `json:"stage"`
}

// ProjectSwitchRequest is the request for POST /api/projects/switch
type ProjectSwitchRequest struct {
	Path string `json:"path"`
}

// CheckpointRestartRequest is the request for POST /api/checkpoints/{id}/restart
type CheckpointRestartRequest struct{}

// --- Response types ---

// ErrorResponse is a standard error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// HealthResponse is the response for GET /api/health
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// Task represents a task from tasks.jsonl
type Task struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Stage        string   `json:"stage"`
	Zone         string   `json:"zone"`
	Persona      string   `json:"persona"`
	Status       string   `json:"status"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
	Dependencies []string `json:"dependencies,omitempty"`
	BlockedBy    []string `json:"blocked_by,omitempty"`
}

// GraphResponse is the response for GET /api/graph
type GraphResponse struct {
	Nodes        []GraphNode `json:"nodes"`
	Edges        []GraphEdge `json:"edges"`
	CriticalPath []string    `json:"critical_path"`
}

// GraphNode is a node in the dependency graph
type GraphNode struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Stage  string `json:"stage"`
	Zone   string `json:"zone"`
}

// GraphEdge is an edge in the dependency graph
type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// CheckpointInfo represents a checkpoint directory
type CheckpointInfo struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
}

// AuditEntry represents a line from audit.jsonl
type AuditEntry struct {
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
	Actor     string `json:"actor"`
	Category  string `json:"category"`
	Details   string `json:"details,omitempty"`
}

// RequirementsCoverage is the response for GET /api/requirements/coverage
type RequirementsCoverage struct {
	Total       int     `json:"total"`
	Implemented int     `json:"implemented"`
	Coverage    float64 `json:"coverage"`
}

// OpenClawStatus is the response for GET /api/openclaw/status
type OpenClawStatus struct {
	Connected bool `json:"connected"`
}

// CommandResult is the response for action endpoints that shell out
type CommandResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output,omitempty"`
	Error   string `json:"error,omitempty"`
}
