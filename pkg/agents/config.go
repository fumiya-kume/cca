package agents

import (
	"fmt"
	"time"
)

// AgentConfiguration represents the configuration for the multi-agent system
type AgentConfiguration struct {
	Global  GlobalAgentConfig       `yaml:"global" json:"global"`
	Agents  map[AgentID]AgentConfig `yaml:"agents" json:"agents"`
	Project ProjectAgentConfig      `yaml:"project" json:"project"`
	User    UserAgentConfig         `yaml:"user" json:"user"`
}

// GlobalAgentConfig contains global settings for all agents
type GlobalAgentConfig struct {
	MaxConcurrentAgents int                  `yaml:"max_concurrent_agents" json:"max_concurrent_agents"`
	DefaultTimeout      time.Duration        `yaml:"default_timeout" json:"default_timeout"`
	MessageBusSettings  MessageBusConfig     `yaml:"message_bus" json:"message_bus"`
	HealthCheck         HealthCheckConfig    `yaml:"health_check" json:"health_check"`
	ResourceLimits      ResourceLimitsConfig `yaml:"resource_limits" json:"resource_limits"`
}

// MessageBusConfig configures the agent message bus
type MessageBusConfig struct {
	BufferSize      int           `yaml:"buffer_size" json:"buffer_size"`
	MaxMessageSize  int           `yaml:"max_message_size" json:"max_message_size"`
	RetryAttempts   int           `yaml:"retry_attempts" json:"retry_attempts"`
	RetryDelay      time.Duration `yaml:"retry_delay" json:"retry_delay"`
	PersistMessages bool          `yaml:"persist_messages" json:"persist_messages"`
}

// HealthCheckConfig configures agent health monitoring
type HealthCheckConfig struct {
	Interval           time.Duration `yaml:"interval" json:"interval"`
	Timeout            time.Duration `yaml:"timeout" json:"timeout"`
	MaxFailures        int           `yaml:"max_failures" json:"max_failures"`
	RecoveryTimeout    time.Duration `yaml:"recovery_timeout" json:"recovery_timeout"`
	EnableAutoRecovery bool          `yaml:"enable_auto_recovery" json:"enable_auto_recovery"`
}

// ResourceLimitsConfig defines resource limits for agents
type ResourceLimitsConfig struct {
	MaxMemoryMB      int           `yaml:"max_memory_mb" json:"max_memory_mb"`
	MaxCPUPercent    float64       `yaml:"max_cpu_percent" json:"max_cpu_percent"`
	MaxExecutionTime time.Duration `yaml:"max_execution_time" json:"max_execution_time"`
	MaxFileHandles   int           `yaml:"max_file_handles" json:"max_file_handles"`
}

// AgentConfig represents configuration for a specific agent
type AgentConfig struct {
	Enabled        bool                   `yaml:"enabled" json:"enabled"`
	MaxInstances   int                    `yaml:"max_instances" json:"max_instances"`
	Timeout        time.Duration          `yaml:"timeout" json:"timeout"`
	RetryAttempts  int                    `yaml:"retry_attempts" json:"retry_attempts"`
	Priority       Priority               `yaml:"priority" json:"priority"`
	ResourceLimits ResourceLimitsConfig   `yaml:"resource_limits" json:"resource_limits"`
	CustomSettings map[string]interface{} `yaml:"custom_settings" json:"custom_settings"`
}

// SecurityAgentConfig contains security-specific settings
type SecurityAgentConfig struct {
	AgentConfig `yaml:",inline"`
	Scanners    SecurityScannersConfig `yaml:"scanners" json:"scanners"`
	Compliance  ComplianceConfig       `yaml:"compliance" json:"compliance"`
	AutoFix     SecurityAutoFixConfig  `yaml:"auto_fix" json:"auto_fix"`
	Severity    SeverityConfig         `yaml:"severity" json:"severity"`
}

// SecurityScannersConfig configures security scanners
type SecurityScannersConfig struct {
	SAST            SASTConfig            `yaml:"sast" json:"sast"`
	DependencyCheck DependencyCheckConfig `yaml:"dependency_check" json:"dependency_check"`
	SecretDetection SecretDetectionConfig `yaml:"secret_detection" json:"secret_detection"`
	LicenseCheck    LicenseCheckConfig    `yaml:"license_check" json:"license_check"`
}

// SASTConfig configures Static Application Security Testing
type SASTConfig struct {
	Enabled           bool     `yaml:"enabled" json:"enabled"`
	Engine            string   `yaml:"engine" json:"engine"` // semgrep, gosec, etc.
	Rules             []string `yaml:"rules" json:"rules"`
	CustomRulesPath   string   `yaml:"custom_rules_path" json:"custom_rules_path"`
	SeverityThreshold string   `yaml:"severity_threshold" json:"severity_threshold"`
	ExcludePaths      []string `yaml:"exclude_paths" json:"exclude_paths"`
}

// DependencyCheckConfig configures dependency vulnerability scanning
type DependencyCheckConfig struct {
	Enabled         bool     `yaml:"enabled" json:"enabled"`
	Databases       []string `yaml:"databases" json:"databases"` // nvd, github_advisories, snyk
	UpdateFrequency string   `yaml:"update_frequency" json:"update_frequency"`
	AutoUpdateSafe  bool     `yaml:"auto_update_safe" json:"auto_update_safe"`
	ExcludePatterns []string `yaml:"exclude_patterns" json:"exclude_patterns"`
}

// SecretDetectionConfig configures secret detection
type SecretDetectionConfig struct {
	Enabled            bool     `yaml:"enabled" json:"enabled"`
	Patterns           []string `yaml:"patterns" json:"patterns"`
	ExcludePatterns    []string `yaml:"exclude_patterns" json:"exclude_patterns"`
	HighEntropyEnabled bool     `yaml:"high_entropy_enabled" json:"high_entropy_enabled"`
	EntropyThreshold   float64  `yaml:"entropy_threshold" json:"entropy_threshold"`
}

// LicenseCheckConfig configures license compliance checking
type LicenseCheckConfig struct {
	Enabled           bool     `yaml:"enabled" json:"enabled"`
	AllowedLicenses   []string `yaml:"allowed_licenses" json:"allowed_licenses"`
	ForbiddenLicenses []string `yaml:"forbidden_licenses" json:"forbidden_licenses"`
	RequireNotice     bool     `yaml:"require_notice" json:"require_notice"`
}

// ComplianceConfig configures compliance frameworks
type ComplianceConfig struct {
	Frameworks   []string `yaml:"frameworks" json:"frameworks"` // sox, gdpr, hipaa, pci-dss
	StrictMode   bool     `yaml:"strict_mode" json:"strict_mode"`
	AuditTrail   bool     `yaml:"audit_trail" json:"audit_trail"`
	ReportFormat string   `yaml:"report_format" json:"report_format"`
}

// SecurityAutoFixConfig configures automated security fixes
type SecurityAutoFixConfig struct {
	Enabled         bool `yaml:"enabled" json:"enabled"`
	SafeFixesOnly   bool `yaml:"safe_fixes_only" json:"safe_fixes_only"`
	MaxFixesPerRun  int  `yaml:"max_fixes_per_run" json:"max_fixes_per_run"`
	RequireApproval bool `yaml:"require_approval" json:"require_approval"`
	BackupBeforeFix bool `yaml:"backup_before_fix" json:"backup_before_fix"`
}

// SeverityConfig configures severity thresholds
type SeverityConfig struct {
	Critical string `yaml:"critical" json:"critical"`
	High     string `yaml:"high" json:"high"`
	Medium   string `yaml:"medium" json:"medium"`
	Low      string `yaml:"low" json:"low"`
}

// ArchitectureAgentConfig contains architecture-specific settings
type ArchitectureAgentConfig struct {
	AgentConfig        `yaml:",inline"`
	ComplexityAnalysis ComplexityAnalysisConfig `yaml:"complexity_analysis" json:"complexity_analysis"`
	DesignPrinciples   DesignPrinciplesConfig   `yaml:"design_principles" json:"design_principles"`
	Refactoring        RefactoringConfig        `yaml:"refactoring" json:"refactoring"`
	QualityMetrics     QualityMetricsConfig     `yaml:"quality_metrics" json:"quality_metrics"`
}

// ComplexityAnalysisConfig configures complexity analysis
type ComplexityAnalysisConfig struct {
	CyclomaticThreshold int `yaml:"cyclomatic_threshold" json:"cyclomatic_threshold"`
	CognitiveThreshold  int `yaml:"cognitive_threshold" json:"cognitive_threshold"`
	FileSizeLimit       int `yaml:"file_size_limit" json:"file_size_limit"`
	NestingDepthLimit   int `yaml:"nesting_depth_limit" json:"nesting_depth_limit"`
	ParameterCountLimit int `yaml:"parameter_count_limit" json:"parameter_count_limit"`
}

// DesignPrinciplesConfig configures design principle validation
type DesignPrinciplesConfig struct {
	SOLID SOLIDConfig `yaml:"solid" json:"solid"`
	DRY   DRYConfig   `yaml:"dry" json:"dry"`
	KISS  KISSConfig  `yaml:"kiss" json:"kiss"`
}

// SOLIDConfig configures SOLID principles validation
type SOLIDConfig struct {
	Enforce    bool `yaml:"enforce" json:"enforce"`
	StrictMode bool `yaml:"strict_mode" json:"strict_mode"`
}

// DRYConfig configures DRY principle validation
type DRYConfig struct {
	DuplicationThreshold int     `yaml:"duplication_threshold" json:"duplication_threshold"`
	SimilarityPercentage float64 `yaml:"similarity_percentage" json:"similarity_percentage"`
	MinimumBlockSize     int     `yaml:"minimum_block_size" json:"minimum_block_size"`
}

// KISSConfig configures KISS principle validation
type KISSConfig struct {
	ComplexityPenalty         float64 `yaml:"complexity_penalty" json:"complexity_penalty"`
	SimplificationSuggestions bool    `yaml:"simplification_suggestions" json:"simplification_suggestions"`
}

// RefactoringConfig configures automated refactoring
type RefactoringConfig struct {
	AutoExtractMethods       bool `yaml:"auto_extract_methods" json:"auto_extract_methods"`
	MethodLengthThreshold    int  `yaml:"method_length_threshold" json:"method_length_threshold"`
	SuggestDesignPatterns    bool `yaml:"suggest_design_patterns" json:"suggest_design_patterns"`
	AutoApplySimpleRefactors bool `yaml:"auto_apply_simple_refactors" json:"auto_apply_simple_refactors"`
}

// QualityMetricsConfig configures quality metrics collection
type QualityMetricsConfig struct {
	MaintainabilityIndex bool `yaml:"maintainability_index" json:"maintainability_index"`
	TechnicalDebtRatio   bool `yaml:"technical_debt_ratio" json:"technical_debt_ratio"`
	CodeChurnAnalysis    bool `yaml:"code_churn_analysis" json:"code_churn_analysis"`
	TestabilityScore     bool `yaml:"testability_score" json:"testability_score"`
}

// DocumentationAgentConfig contains documentation-specific settings
type DocumentationAgentConfig struct {
	AgentConfig          `yaml:",inline"`
	CoverageRequirements CoverageRequirementsConfig   `yaml:"coverage_requirements" json:"coverage_requirements"`
	AutoGeneration       AutoGenerationConfig         `yaml:"auto_generation" json:"auto_generation"`
	QualityAssessment    DocQualityAssessmentConfig   `yaml:"quality_assessment" json:"quality_assessment"`
	Templates            DocumentationTemplatesConfig `yaml:"templates" json:"templates"`
}

// CoverageRequirementsConfig defines documentation coverage requirements
type CoverageRequirementsConfig struct {
	PublicAPIs      int `yaml:"public_apis" json:"public_apis"`
	InternalAPIs    int `yaml:"internal_apis" json:"internal_apis"`
	TypeDefinitions int `yaml:"type_definitions" json:"type_definitions"`
	Examples        int `yaml:"examples" json:"examples"`
}

// AutoGenerationConfig configures automatic documentation generation
type AutoGenerationConfig struct {
	README    bool `yaml:"readme" json:"readme"`
	Changelog bool `yaml:"changelog" json:"changelog"`
	APIDocs   bool `yaml:"api_docs" json:"api_docs"`
	Examples  bool `yaml:"examples" json:"examples"`
	Tutorials bool `yaml:"tutorials" json:"tutorials"`
}

// DocQualityAssessmentConfig configures documentation quality assessment
type DocQualityAssessmentConfig struct {
	ReadabilityScore   bool `yaml:"readability_score" json:"readability_score"`
	CompletenessCheck  bool `yaml:"completeness_check" json:"completeness_check"`
	AccuracyValidation bool `yaml:"accuracy_validation" json:"accuracy_validation"`
	FreshnessCheck     bool `yaml:"freshness_check" json:"freshness_check"`
}

// DocumentationTemplatesConfig configures documentation templates
type DocumentationTemplatesConfig struct {
	FunctionTemplate string            `yaml:"function_template" json:"function_template"`
	ClassTemplate    string            `yaml:"class_template" json:"class_template"`
	READMETemplate   string            `yaml:"readme_template" json:"readme_template"`
	CustomTemplates  map[string]string `yaml:"custom_templates" json:"custom_templates"`
}

// TestingAgentConfig contains testing-specific settings
type TestingAgentConfig struct {
	AgentConfig     `yaml:",inline"`
	CoverageTargets CoverageTargetsConfig    `yaml:"coverage_targets" json:"coverage_targets"`
	TestGeneration  TestGenerationConfig     `yaml:"test_generation" json:"test_generation"`
	QualityMetrics  TestQualityMetricsConfig `yaml:"quality_metrics" json:"quality_metrics"`
	TestExecution   TestExecutionConfig      `yaml:"test_execution" json:"test_execution"`
}

// CoverageTargetsConfig defines test coverage targets
type CoverageTargetsConfig struct {
	Unit        int `yaml:"unit" json:"unit"`
	Integration int `yaml:"integration" json:"integration"`
	E2E         int `yaml:"e2e" json:"e2e"`
	Mutation    int `yaml:"mutation" json:"mutation"`
}

// TestGenerationConfig configures test generation
type TestGenerationConfig struct {
	AutoGenerate      bool     `yaml:"auto_generate" json:"auto_generate"`
	TestTypes         []string `yaml:"test_types" json:"test_types"`
	EdgeCases         bool     `yaml:"edge_cases" json:"edge_cases"`
	MockGeneration    bool     `yaml:"mock_generation" json:"mock_generation"`
	FixtureGeneration bool     `yaml:"fixture_generation" json:"fixture_generation"`
}

// TestQualityMetricsConfig configures test quality metrics
type TestQualityMetricsConfig struct {
	TestEffectiveness   bool `yaml:"test_effectiveness" json:"test_effectiveness"`
	TestMaintainability bool `yaml:"test_maintainability" json:"test_maintainability"`
	TestReliability     bool `yaml:"test_reliability" json:"test_reliability"`
	TestPerformance     bool `yaml:"test_performance" json:"test_performance"`
}

// TestExecutionConfig configures test execution
type TestExecutionConfig struct {
	Parallel      bool          `yaml:"parallel" json:"parallel"`
	Timeout       time.Duration `yaml:"timeout" json:"timeout"`
	RetryAttempts int           `yaml:"retry_attempts" json:"retry_attempts"`
	FailFast      bool          `yaml:"fail_fast" json:"fail_fast"`
	VerboseOutput bool          `yaml:"verbose_output" json:"verbose_output"`
}

// PerformanceAgentConfig contains performance-specific settings
type PerformanceAgentConfig struct {
	AgentConfig  `yaml:",inline"`
	Profiling    ProfilingConfig    `yaml:"profiling" json:"profiling"`
	Benchmarking BenchmarkingConfig `yaml:"benchmarking" json:"benchmarking"`
	Optimization OptimizationConfig `yaml:"optimization" json:"optimization"`
	Monitoring   MonitoringConfig   `yaml:"monitoring" json:"monitoring"`
}

// ProfilingConfig configures performance profiling
type ProfilingConfig struct {
	Enabled      bool          `yaml:"enabled" json:"enabled"`
	ProfileTypes []string      `yaml:"profile_types" json:"profile_types"`
	Duration     time.Duration `yaml:"duration" json:"duration"`
	SampleRate   int           `yaml:"sample_rate" json:"sample_rate"`
	AutoProfile  bool          `yaml:"auto_profile" json:"auto_profile"`
}

// BenchmarkingConfig configures performance benchmarking
type BenchmarkingConfig struct {
	BenchmarkOnChange   bool          `yaml:"benchmark_on_change" json:"benchmark_on_change"`
	RegressionThreshold string        `yaml:"regression_threshold" json:"regression_threshold"`
	BaselineDuration    time.Duration `yaml:"baseline_duration" json:"baseline_duration"`
	WarmupTime          time.Duration `yaml:"warmup_time" json:"warmup_time"`
}

// OptimizationConfig configures performance optimization
type OptimizationConfig struct {
	OptimizationSuggestions bool               `yaml:"optimization_suggestions" json:"optimization_suggestions"`
	AutoOptimizations       bool               `yaml:"auto_optimizations" json:"auto_optimizations"`
	OptimizationTypes       []string           `yaml:"optimization_types" json:"optimization_types"`
	PerformanceGoals        map[string]float64 `yaml:"performance_goals" json:"performance_goals"`
}

// MonitoringConfig configures performance monitoring
type MonitoringConfig struct {
	ContinuousMonitoring bool               `yaml:"continuous_monitoring" json:"continuous_monitoring"`
	MetricCollection     []string           `yaml:"metric_collection" json:"metric_collection"`
	AlertThresholds      map[string]float64 `yaml:"alert_thresholds" json:"alert_thresholds"`
	MonitoringInterval   time.Duration      `yaml:"monitoring_interval" json:"monitoring_interval"`
}

// ProjectAgentConfig contains project-specific overrides
type ProjectAgentConfig struct {
	Type              string                 `yaml:"type" json:"type"` // golang_web_service, react_app, etc.
	WorkflowOverrides WorkflowOverrides      `yaml:"workflow_overrides" json:"workflow_overrides"`
	QualityGates      QualityGatesConfig     `yaml:"quality_gates" json:"quality_gates"`
	CustomRules       map[string]interface{} `yaml:"custom_rules" json:"custom_rules"`
}

// WorkflowOverrides allows overriding default workflow behavior
type WorkflowOverrides struct {
	RequireSecurityReview      bool `yaml:"require_security_review" json:"require_security_review"`
	RequireArchitectureReview  bool `yaml:"require_architecture_review" json:"require_architecture_review"`
	RequirePerformanceAnalysis bool `yaml:"require_performance_analysis" json:"require_performance_analysis"`
	AutoMergeEnabled           bool `yaml:"auto_merge_enabled" json:"auto_merge_enabled"`
	ParallelExecution          bool `yaml:"parallel_execution" json:"parallel_execution"`
}

// QualityGatesConfig defines quality gates that must be met
type QualityGatesConfig struct {
	MinimumTestCoverage   int     `yaml:"minimum_test_coverage" json:"minimum_test_coverage"`
	MaximumComplexity     int     `yaml:"maximum_complexity" json:"maximum_complexity"`
	SecurityScanRequired  bool    `yaml:"security_scan_required" json:"security_scan_required"`
	DocumentationRequired bool    `yaml:"documentation_required" json:"documentation_required"`
	PerformanceThreshold  string  `yaml:"performance_threshold" json:"performance_threshold"`
	MaxTechnicalDebt      float64 `yaml:"max_technical_debt" json:"max_technical_debt"`
}

// UserAgentConfig contains user-specific preferences
type UserAgentConfig struct {
	AutomationLevel         string                  `yaml:"automation_level" json:"automation_level"` // full_auto, collaborative, manual
	NotificationPreferences NotificationPreferences `yaml:"notification_preferences" json:"notification_preferences"`
	UI                      UIPreferences           `yaml:"ui" json:"ui"`
	CustomPreferences       map[string]interface{}  `yaml:"custom_preferences" json:"custom_preferences"`
}

// NotificationPreferences configures user notifications
type NotificationPreferences struct {
	AgentProgress        bool `yaml:"agent_progress" json:"agent_progress"`
	CriticalIssues       bool `yaml:"critical_issues" json:"critical_issues"`
	CompletionSummary    bool `yaml:"completion_summary" json:"completion_summary"`
	AutoFixNotifications bool `yaml:"auto_fix_notifications" json:"auto_fix_notifications"`
	ConflictResolutions  bool `yaml:"conflict_resolutions" json:"conflict_resolutions"`
}

// UIPreferences configures user interface preferences
type UIPreferences struct {
	Theme            string `yaml:"theme" json:"theme"`   // dark, light, auto
	Layout           string `yaml:"layout" json:"layout"` // compact, full, minimal
	RealTimeUpdates  bool   `yaml:"real_time_updates" json:"real_time_updates"`
	ShowAgentDetails bool   `yaml:"show_agent_details" json:"show_agent_details"`
	AnimationSpeed   string `yaml:"animation_speed" json:"animation_speed"`
}

// DefaultAgentConfiguration returns a sensible default configuration
func DefaultAgentConfiguration() *AgentConfiguration {
	return &AgentConfiguration{
		Global: GlobalAgentConfig{
			MaxConcurrentAgents: 5,
			DefaultTimeout:      30 * time.Minute,
			MessageBusSettings: MessageBusConfig{
				BufferSize:      1000,
				MaxMessageSize:  1024 * 1024, // 1MB
				RetryAttempts:   3,
				RetryDelay:      time.Second,
				PersistMessages: true,
			},
			HealthCheck: HealthCheckConfig{
				Interval:           30 * time.Second,
				Timeout:            10 * time.Second,
				MaxFailures:        3,
				RecoveryTimeout:    5 * time.Minute,
				EnableAutoRecovery: true,
			},
			ResourceLimits: ResourceLimitsConfig{
				MaxMemoryMB:      512,
				MaxCPUPercent:    50.0,
				MaxExecutionTime: 30 * time.Minute,
				MaxFileHandles:   100,
			},
		},
		Agents: map[AgentID]AgentConfig{
			SecurityAgentID: {
				Enabled:       true,
				MaxInstances:  2,
				Timeout:       20 * time.Minute,
				RetryAttempts: 2,
				Priority:      PriorityHigh,
				ResourceLimits: ResourceLimitsConfig{
					MaxMemoryMB:      256,
					MaxCPUPercent:    30.0,
					MaxExecutionTime: 20 * time.Minute,
				},
			},
			ArchitectureAgentID: {
				Enabled:       true,
				MaxInstances:  1,
				Timeout:       15 * time.Minute,
				RetryAttempts: 2,
				Priority:      PriorityMedium,
				ResourceLimits: ResourceLimitsConfig{
					MaxMemoryMB:      128,
					MaxCPUPercent:    25.0,
					MaxExecutionTime: 15 * time.Minute,
				},
			},
			DocumentationAgentID: {
				Enabled:       true,
				MaxInstances:  1,
				Timeout:       10 * time.Minute,
				RetryAttempts: 2,
				Priority:      PriorityMedium,
				ResourceLimits: ResourceLimitsConfig{
					MaxMemoryMB:      128,
					MaxCPUPercent:    20.0,
					MaxExecutionTime: 10 * time.Minute,
				},
			},
			TestingAgentID: {
				Enabled:       true,
				MaxInstances:  2,
				Timeout:       25 * time.Minute,
				RetryAttempts: 2,
				Priority:      PriorityHigh,
				ResourceLimits: ResourceLimitsConfig{
					MaxMemoryMB:      256,
					MaxCPUPercent:    40.0,
					MaxExecutionTime: 25 * time.Minute,
				},
			},
			PerformanceAgentID: {
				Enabled:       true,
				MaxInstances:  1,
				Timeout:       20 * time.Minute,
				RetryAttempts: 2,
				Priority:      PriorityMedium,
				ResourceLimits: ResourceLimitsConfig{
					MaxMemoryMB:      256,
					MaxCPUPercent:    35.0,
					MaxExecutionTime: 20 * time.Minute,
				},
			},
		},
		Project: ProjectAgentConfig{
			Type: "golang_web_service",
			WorkflowOverrides: WorkflowOverrides{
				RequireSecurityReview:      true,
				RequireArchitectureReview:  true,
				RequirePerformanceAnalysis: false,
				AutoMergeEnabled:           false,
				ParallelExecution:          true,
			},
			QualityGates: QualityGatesConfig{
				MinimumTestCoverage:   80,
				MaximumComplexity:     15,
				SecurityScanRequired:  true,
				DocumentationRequired: true,
				PerformanceThreshold:  "5%",
				MaxTechnicalDebt:      8.0,
			},
		},
		User: UserAgentConfig{
			AutomationLevel: "collaborative",
			NotificationPreferences: NotificationPreferences{
				AgentProgress:        true,
				CriticalIssues:       true,
				CompletionSummary:    true,
				AutoFixNotifications: true,
				ConflictResolutions:  true,
			},
			UI: UIPreferences{
				Theme:            "dark",
				Layout:           "full",
				RealTimeUpdates:  true,
				ShowAgentDetails: true,
				AnimationSpeed:   "normal",
			},
		},
	}
}

// Validate validates the agent configuration
func (ac *AgentConfiguration) Validate() error {
	if ac.Global.MaxConcurrentAgents <= 0 {
		return fmt.Errorf("max_concurrent_agents must be positive")
	}

	if ac.Global.DefaultTimeout <= 0 {
		return fmt.Errorf("default_timeout must be positive")
	}

	// Validate each agent configuration
	for agentID, config := range ac.Agents {
		if err := ac.validateAgentConfig(agentID, config); err != nil {
			return fmt.Errorf("invalid config for agent %s: %w", agentID, err)
		}
	}

	return nil
}

// validateAgentConfig validates a specific agent's configuration
func (ac *AgentConfiguration) validateAgentConfig(agentID AgentID, config AgentConfig) error {
	if config.MaxInstances <= 0 {
		return fmt.Errorf("max_instances must be positive")
	}

	if config.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	if config.ResourceLimits.MaxMemoryMB <= 0 {
		return fmt.Errorf("max_memory_mb must be positive")
	}

	if config.ResourceLimits.MaxCPUPercent <= 0 || config.ResourceLimits.MaxCPUPercent > 100 {
		return fmt.Errorf("max_cpu_percent must be between 0 and 100")
	}

	return nil
}

// GetAgentConfig returns the configuration for a specific agent
func (ac *AgentConfiguration) GetAgentConfig(agentID AgentID) (AgentConfig, bool) {
	config, exists := ac.Agents[agentID]
	return config, exists
}

// IsAgentEnabled checks if a specific agent is enabled
func (ac *AgentConfiguration) IsAgentEnabled(agentID AgentID) bool {
	config, exists := ac.GetAgentConfig(agentID)
	return exists && config.Enabled
}

// GetEnabledAgents returns a list of all enabled agents
func (ac *AgentConfiguration) GetEnabledAgents() []AgentID {
	var enabled []AgentID
	for agentID, config := range ac.Agents {
		if config.Enabled {
			enabled = append(enabled, agentID)
		}
	}
	return enabled
}

// UpdateAgentConfig updates the configuration for a specific agent
func (ac *AgentConfiguration) UpdateAgentConfig(agentID AgentID, config AgentConfig) error {
	if err := ac.validateAgentConfig(agentID, config); err != nil {
		return err
	}
	ac.Agents[agentID] = config
	return nil
}

// EnableAgent enables a specific agent
func (ac *AgentConfiguration) EnableAgent(agentID AgentID) {
	if config, exists := ac.Agents[agentID]; exists {
		config.Enabled = true
		ac.Agents[agentID] = config
	}
}

// DisableAgent disables a specific agent
func (ac *AgentConfiguration) DisableAgent(agentID AgentID) {
	if config, exists := ac.Agents[agentID]; exists {
		config.Enabled = false
		ac.Agents[agentID] = config
	}
}

// Merge merges another configuration into this one, with the other taking precedence
func (ac *AgentConfiguration) Merge(other *AgentConfiguration) {
	// Merge global settings
	if other.Global.MaxConcurrentAgents > 0 {
		ac.Global.MaxConcurrentAgents = other.Global.MaxConcurrentAgents
	}
	if other.Global.DefaultTimeout > 0 {
		ac.Global.DefaultTimeout = other.Global.DefaultTimeout
	}

	// Merge agent configs
	for agentID, config := range other.Agents {
		ac.Agents[agentID] = config
	}

	// Merge project config
	if other.Project.Type != "" {
		ac.Project.Type = other.Project.Type
	}

	// Merge user config
	if other.User.AutomationLevel != "" {
		ac.User.AutomationLevel = other.User.AutomationLevel
	}
}

// Clone creates a deep copy of the configuration
func (ac *AgentConfiguration) Clone() *AgentConfiguration {
	clone := &AgentConfiguration{
		Global:  ac.Global,
		Agents:  make(map[AgentID]AgentConfig),
		Project: ac.Project,
		User:    ac.User,
	}

	for agentID, config := range ac.Agents {
		clone.Agents[agentID] = config
	}

	return clone
}
