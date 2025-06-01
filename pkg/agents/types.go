package agents

import (
	"context"
	"time"
)

// AgentID represents a unique identifier for an agent
type AgentID string

// Agent IDs for the specialized agents
const (
	ClaudeCodeAgentID      AgentID = "claude-code"
	SecurityAgentID        AgentID = "security"
	ArchitectureAgentID    AgentID = "architecture"
	DocumentationAgentID   AgentID = "documentation"
	TestingAgentID         AgentID = "testing"
	PerformanceAgentID     AgentID = "performance"
	GitAgentID             AgentID = "git"
	GitHubAgentID          AgentID = "github"
	AnalysisAgentID        AgentID = "analysis"
	WorkflowOrchestratorID AgentID = "workflow_orchestrator"
)

// MessageType represents the type of message being sent
type MessageType string

const (
	TaskAssignment       MessageType = "task_assignment"
	ProgressUpdate       MessageType = "progress_update"
	ResultReporting      MessageType = "result_reporting"
	ErrorNotification    MessageType = "error_notification"
	HealthCheck          MessageType = "health_check"
	ResourceRequest      MessageType = "resource_request"
	CollaborationRequest MessageType = "collaboration_request"
	FeedbackProvision    MessageType = "feedback_provision"
)

// Priority represents the priority level of a message or task
type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// AgentStatus represents the current status of an agent
type AgentStatus string

const (
	StatusIdle     AgentStatus = "idle"
	StatusBusy     AgentStatus = "busy"
	StatusError    AgentStatus = "error"
	StatusOffline  AgentStatus = "offline"
	StatusStarting AgentStatus = "starting"
	StatusStopping AgentStatus = "stopping"
)

// MessageContext provides additional context for a message
type MessageContext struct {
	WorkflowID  string                 `json:"workflow_id"`
	StageID     string                 `json:"stage_id,omitempty"`
	IssueNumber int                    `json:"issue_number,omitempty"`
	Repository  string                 `json:"repository,omitempty"`
	Branch      string                 `json:"branch,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AgentMessage represents a message exchanged between agents
type AgentMessage struct {
	ID            string         `json:"id"`
	Type          MessageType    `json:"type"`
	Sender        AgentID        `json:"sender"`
	Receiver      AgentID        `json:"receiver"`
	Payload       interface{}    `json:"payload"`
	Timestamp     time.Time      `json:"timestamp"`
	Priority      Priority       `json:"priority"`
	CorrelationID string         `json:"correlation_id,omitempty"`
	Context       MessageContext `json:"context"`
}

// Agent represents a specialized AI agent in the system
type Agent interface {
	// GetID returns the unique identifier of the agent
	GetID() AgentID

	// GetStatus returns the current status of the agent
	GetStatus() AgentStatus

	// Start initializes and starts the agent
	Start(ctx context.Context) error

	// Stop gracefully shuts down the agent
	Stop(ctx context.Context) error

	// ProcessMessage handles an incoming message
	ProcessMessage(ctx context.Context, msg *AgentMessage) error

	// GetCapabilities returns the agent's capabilities
	GetCapabilities() []string

	// HealthCheck performs a health check on the agent
	HealthCheck(ctx context.Context) error
}

// ResourceLimits defines resource constraints for an agent
type ResourceLimits struct {
	MaxMemoryMB    int     `json:"max_memory_mb"`
	MaxCPUPercent  float64 `json:"max_cpu_percent"`
	MaxDiskSpaceGB int     `json:"max_disk_space_gb"`
}

// CollaborationType represents types of collaboration between agents
type CollaborationType string

const (
	DataSharing           CollaborationType = "data_sharing"
	ResultValidation      CollaborationType = "result_validation"
	ConflictResolution    CollaborationType = "conflict_resolution"
	ExpertiseConsultation CollaborationType = "expertise_consultation"
)

// CollaborationContext provides context for agent collaboration
type CollaborationContext struct {
	Purpose     string                 `json:"purpose"`
	SharedData  interface{}            `json:"shared_data,omitempty"`
	Constraints map[string]interface{} `json:"constraints,omitempty"`
	Deadline    *time.Time             `json:"deadline,omitempty"`
}

// AgentCollaboration represents a collaboration request between agents
type AgentCollaboration struct {
	RequestingAgent   AgentID              `json:"requesting_agent"`
	TargetAgent       AgentID              `json:"target_agent"`
	CollaborationType CollaborationType    `json:"collaboration_type"`
	Context           CollaborationContext `json:"context"`
	Priority          Priority             `json:"priority"`
	RequestTime       time.Time            `json:"request_time"`
}

// AgentResult represents the result of an agent's execution
type AgentResult struct {
	AgentID   AgentID                `json:"agent_id"`
	Success   bool                   `json:"success"`
	Results   map[string]interface{} `json:"results,omitempty"`
	Errors    []error                `json:"errors,omitempty"`
	Warnings  []string               `json:"warnings,omitempty"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Timestamp time.Time              `json:"timestamp"`
}

// ResultAggregation represents aggregated results from multiple agents
type ResultAggregation struct {
	WorkflowID   string                   `json:"workflow_id"`
	AgentResults map[AgentID]*AgentResult `json:"agent_results"`
	Conflicts    []ConflictReport         `json:"conflicts,omitempty"`
	Priorities   []PriorityItem           `json:"priorities"`
	Summary      string                   `json:"summary"`
	Timestamp    time.Time                `json:"timestamp"`
}

// ConflictReport represents a conflict between agent recommendations
type ConflictReport struct {
	ConflictingAgents []AgentID `json:"conflicting_agents"`
	IssueType         string    `json:"issue_type"`
	Description       string    `json:"description"`
	Resolution        string    `json:"resolution,omitempty"`
	RequiresHuman     bool      `json:"requires_human"`
}

// PriorityItem represents a prioritized issue or recommendation
type PriorityItem struct {
	ID          string      `json:"id"`
	AgentID     AgentID     `json:"agent_id"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Severity    Priority    `json:"severity"`
	Impact      string      `json:"impact"`
	AutoFixable bool        `json:"auto_fixable"`
	FixDetails  interface{} `json:"fix_details,omitempty"`
}

// WorkspaceConfig represents workspace configuration
type WorkspaceConfig struct {
	WorkspacePath string                 `json:"workspace_path"`
	ProjectType   string                 `json:"project_type"`
	Configuration map[string]interface{} `json:"configuration"`
}

// StageResult represents the result of a workflow stage
type StageResult struct {
	Name     string                 `json:"name"`
	Status   string                 `json:"status"`
	Duration time.Duration          `json:"duration"`
	Output   []string               `json:"output"`
	Metadata map[string]interface{} `json:"metadata"`
}

// HealthMonitorMetrics represents health monitoring metrics
type HealthMonitorMetrics struct {
	TotalAgents        int       `json:"total_agents"`
	HealthyAgents      int       `json:"healthy_agents"`
	TotalHealthChecks  int       `json:"total_health_checks"`
	FailedHealthChecks int       `json:"failed_health_checks"`
	LastCheckTime      time.Time `json:"last_check_time"`
}

// MessageBusHealth represents message bus health information
type MessageBusHealth struct {
	TotalMessages     int64         `json:"total_messages"`
	FailedMessages    int64         `json:"failed_messages"`
	AverageLatency    time.Duration `json:"average_latency"`
	ActiveSubscribers int           `json:"active_subscribers"`
	LastActivity      time.Time     `json:"last_activity"`
}

// AgentRegistryMetrics represents agent registry metrics
type AgentRegistryMetrics struct {
	TotalAgents   int       `json:"total_agents"`
	ActiveAgents  int       `json:"active_agents"`
	FailedAgents  int       `json:"failed_agents"`
	Registrations int64     `json:"registrations"`
	LastUpdate    time.Time `json:"last_update"`
}
