// Package types provides core data structures and type definitions used across
// the ccAgents application for GitHub integration and workflow management.
package types

import "time"

// IssueReference represents a parsed GitHub issue reference
type IssueReference struct {
	Owner  string
	Repo   string
	Number int
	Source string // "url", "shorthand", "context"
}

// IssueData contains comprehensive issue information
type IssueData struct {
	Number           int            `json:"number"`
	Title            string         `json:"title"`
	Body             string         `json:"body"`
	Description      string         `json:"description"` // Alias for Body
	State            string         `json:"state"`
	Repository       string         `json:"repository"`
	Labels           []string       `json:"labels"`
	Assignees        []string       `json:"assignees"`
	Milestone        string         `json:"milestone"`
	Comments         []IssueComment `json:"comments"`
	WorkingDirectory string         `json:"working_directory"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// Comment represents an issue comment
type Comment struct {
	ID        int       `json:"id"`
	Body      string    `json:"body"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
}

// IssueComment represents an issue comment (alias for backward compatibility)
type IssueComment = Comment

// CodeContext represents comprehensive codebase context for analysis
type CodeContext struct {
	ProjectSummary    *ProjectSummaryInfo    `json:"project_summary"`
	TechnicalStack    *TechnicalStackInfo    `json:"technical_stack"`
	Architecture      *ArchitectureInfo      `json:"architecture"`
	CodeConventions   *CodeConventionsInfo   `json:"code_conventions"`
	RelevantFiles     []RelevantFileInfo     `json:"relevant_files"`
	Dependencies      *DependencyContextInfo `json:"dependencies"`
	TestingStrategy   *TestingStrategyInfo   `json:"testing_strategy"`
	BuildInformation  *BuildInformationInfo  `json:"build_information"`
	DocumentationRefs *DocumentationRefsInfo `json:"documentation_refs"`
	IssueContext      *IssueContextInfo      `json:"issue_context,omitempty"`
	Metadata          *ContextMetadata       `json:"metadata"`
}

// ProjectSummaryInfo contains project overview information
type ProjectSummaryInfo struct {
	Name                 string             `json:"name"`
	Description          string             `json:"description"`
	Version              string             `json:"version"`
	License              string             `json:"license"`
	MainLanguage         string             `json:"main_language"`
	ProjectType          string             `json:"project_type"`
	BuildSystem          string             `json:"build_system"`
	PackageManager       string             `json:"package_manager"`
	Tags                 []string           `json:"tags"`
	LanguageDistribution map[string]float64 `json:"language_distribution"`
	PrimaryFrameworks    []string           `json:"primary_frameworks"`
	TotalFiles           int                `json:"total_files"`
	TotalLines           int                `json:"total_lines"`
	ProjectSize          int64              `json:"project_size"`
}

// TechnicalStackInfo contains detailed technical stack information
type TechnicalStackInfo struct {
	Languages        []LanguageInfo  `json:"languages"`
	Frameworks       []FrameworkInfo `json:"frameworks"`
	BuildTools       []string        `json:"build_tools"`
	TestingTools     []string        `json:"testing_tools"`
	DevelopmentTools []string        `json:"development_tools"`
	Databases        []string        `json:"databases"`
	CloudServices    []string        `json:"cloud_services"`
}

// LanguageInfo contains language-specific information
type LanguageInfo struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Percentage  float64  `json:"percentage"`
	FileCount   int      `json:"file_count"`
	LineCount   int      `json:"line_count"`
	Extensions  []string `json:"extensions"`
	ConfigFiles []string `json:"config_files"`
}

// FrameworkInfo contains framework-specific information
type FrameworkInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"`
	Purpose string `json:"purpose"`
}

// ArchitectureInfo contains architecture and design information
type ArchitectureInfo struct {
	ProjectStructure map[string]interface{}   `json:"project_structure"`
	DesignPatterns   []string                 `json:"design_patterns"`
	DatabaseSchema   map[string]interface{}   `json:"database_schema"`
	APIEndpoints     []map[string]interface{} `json:"api_endpoints"`
	ConfigStructure  map[string]interface{}   `json:"config_structure"`
	DeploymentInfo   map[string]interface{}   `json:"deployment_info"`
}

// CodeConventionsInfo contains coding conventions and standards
type CodeConventionsInfo struct {
	IndentationStyle   string                 `json:"indentation_style"`
	IndentationSize    int                    `json:"indentation_size"`
	LineEndings        string                 `json:"line_endings"`
	QuoteStyle         string                 `json:"quote_style"`
	NamingConventions  map[string]string      `json:"naming_conventions"`
	FileNaming         string                 `json:"file_naming"`
	DirectoryNaming    string                 `json:"directory_naming"`
	CommentStyle       []string               `json:"comment_style"`
	FormattingRules    map[string]interface{} `json:"formatting_rules"`
	ImportStyle        string                 `json:"import_style"`
	FileOrganization   string                 `json:"file_organization"`
	TestingPatterns    []string               `json:"testing_patterns"`
	ErrorHandling      string                 `json:"error_handling"`
	DocumentationStyle string                 `json:"documentation_style"`
}

// RelevantFileInfo contains information about files relevant to the context
type RelevantFileInfo struct {
	Path         string    `json:"path"`
	Purpose      string    `json:"purpose"`
	Language     string    `json:"language"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	Importance   float64   `json:"importance"`
	Summary      string    `json:"summary"`
}

// DependencyContextInfo contains dependency-related context
type DependencyContextInfo struct {
	DirectDependencies  int            `json:"direct_dependencies"`
	DevDependencies     int            `json:"dev_dependencies"`
	TotalDependencies   int            `json:"total_dependencies"`
	SecurityIssues      int            `json:"security_issues"`
	LicenseDistribution map[string]int `json:"license_distribution"`
	UpdatesAvailable    int            `json:"updates_available"`
	KeyDependencies     []string       `json:"key_dependencies"`
	DependencyConflicts int            `json:"dependency_conflicts"`
}

// TestingStrategyInfo contains testing approach and tools
type TestingStrategyInfo struct {
	TestFrameworks  []string `json:"test_frameworks"`
	TestDirectories []string `json:"test_directories"`
	CoverageTools   []string `json:"coverage_tools"`
	TestPatterns    []string `json:"test_patterns"`
	TestingApproach string   `json:"testing_approach"`
	MockingStrategy string   `json:"mocking_strategy"`
	E2ETestingTools []string `json:"e2e_testing_tools"`
}

// BuildInformationInfo contains build system information
type BuildInformationInfo struct {
	BuildSystem       string            `json:"build_system"`
	BuildFiles        []string          `json:"build_files"`
	BuildTargets      []string          `json:"build_targets"`
	ArtifactTypes     []string          `json:"artifact_types"`
	BuildScripts      map[string]string `json:"build_scripts"`
	CompilerFlags     []string          `json:"compiler_flags"`
	OptimizationLevel string            `json:"optimization_level"`
	OutputDirectories []string          `json:"output_directories"`
}

// DocumentationRefsInfo contains documentation references
type DocumentationRefsInfo struct {
	ReadmeFile       string   `json:"readme_file"`
	DocDirectories   []string `json:"doc_directories"`
	InlineDocStyle   string   `json:"inline_doc_style"`
	APIDocumentation []string `json:"api_documentation"`
	Guides           []string `json:"guides"`
	Examples         []string `json:"examples"`
	ChangelogFile    string   `json:"changelog_file"`
	ContributingFile string   `json:"contributing_file"`
}

// IssueContextInfo contains issue-specific context information
type IssueContextInfo struct {
	IssueNumber         int           `json:"issue_number"`
	Title               string        `json:"title"`
	Description         string        `json:"description"`
	Labels              []string      `json:"labels"`
	Requirements        []string      `json:"requirements"`
	AcceptanceCriteria  []string      `json:"acceptance_criteria"`
	MentionedFiles      []string      `json:"mentioned_files"`
	RelatedIssues       []int         `json:"related_issues"`
	Complexity          string        `json:"complexity"`
	EstimatedEffort     time.Duration `json:"estimated_effort"`
	RiskFactors         []string      `json:"risk_factors"`
	TestStrategy        string        `json:"test_strategy"`
	ImplementationHints []string      `json:"implementation_hints"`
}

// ContextMetadata contains metadata about the context generation
type ContextMetadata struct {
	GeneratedAt      time.Time     `json:"generated_at"`
	AnalysisVersion  string        `json:"analysis_version"`
	ContextVersion   string        `json:"context_version"`
	AnalysisDuration time.Duration `json:"analysis_duration"`
	FilesAnalyzed    int           `json:"files_analyzed"`
	DataSources      []string      `json:"data_sources"`
	Confidence       float64       `json:"confidence"`
	Completeness     float64       `json:"completeness"`
}

// FileContext represents a file with its content and metadata
type FileContext struct {
	Path         string    `json:"path"`
	Content      string    `json:"content"`
	Language     string    `json:"language"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
}

// PRData contains pull request information
type PRData struct {
	Number     int       `json:"number"`
	Title      string    `json:"title"`
	Body       string    `json:"body"`
	State      string    `json:"state"`
	Branch     string    `json:"branch"`
	BaseBranch string    `json:"base_branch"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// WorkflowState represents the current state of the workflow
type WorkflowState int

const (
	WorkflowIdle WorkflowState = iota
	WorkflowRunning
	WorkflowPaused
	WorkflowCompleted
	WorkflowFailed
)

// StageStatus represents the status of a workflow stage
type StageStatus int

const (
	StagePending StageStatus = iota
	StageRunning
	StageCompleted
	StageFailed
	StageSkipped
)

// Stage represents a single workflow stage
type Stage struct {
	Name         string
	Status       StageStatus
	Output       []string
	Dependencies []string
	RetryCount   int
	MaxRetries   int
	StartTime    time.Time
	EndTime      time.Time
}

// ProcessState represents the state of an external process
type ProcessState struct {
	ID        string
	Command   string
	Args      []string
	Status    string
	Output    []string
	ExitCode  int
	StartTime time.Time
	EndTime   time.Time
}
