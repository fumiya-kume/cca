package agents

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultAgentConfiguration(t *testing.T) {
	config := DefaultAgentConfiguration()

	assert.NotNil(t, config)
	assert.NotNil(t, config.Agents)

	// Test global settings
	assert.Equal(t, 5, config.Global.MaxConcurrentAgents)
	assert.Equal(t, 30*time.Minute, config.Global.DefaultTimeout)
	assert.Equal(t, 1000, config.Global.MessageBusSettings.BufferSize)
	assert.Equal(t, 30*time.Second, config.Global.HealthCheck.Interval)
	assert.Equal(t, 512, config.Global.ResourceLimits.MaxMemoryMB)

	// Test that default agents are configured
	expectedAgents := []AgentID{
		SecurityAgentID,
		ArchitectureAgentID,
		DocumentationAgentID,
		TestingAgentID,
		PerformanceAgentID,
	}

	for _, agentID := range expectedAgents {
		agentConfig, exists := config.GetAgentConfig(agentID)
		assert.True(t, exists, "Agent %s should be configured", agentID)
		assert.True(t, agentConfig.Enabled, "Agent %s should be enabled", agentID)
	}

	// Test project settings
	assert.Equal(t, "golang_web_service", config.Project.Type)
	assert.True(t, config.Project.WorkflowOverrides.RequireSecurityReview)
	assert.Equal(t, 80, config.Project.QualityGates.MinimumTestCoverage)

	// Test user settings
	assert.Equal(t, "collaborative", config.User.AutomationLevel)
	assert.True(t, config.User.NotificationPreferences.CriticalIssues)
	assert.Equal(t, "dark", config.User.UI.Theme)
}

func TestAgentConfigurationValidate(t *testing.T) {
	// Test valid configuration
	validConfig := DefaultAgentConfiguration()
	err := validConfig.Validate()
	assert.NoError(t, err)

	// Test invalid max concurrent agents
	invalidConfig := DefaultAgentConfiguration()
	invalidConfig.Global.MaxConcurrentAgents = 0
	err = invalidConfig.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_concurrent_agents must be positive")

	// Test invalid default timeout
	invalidConfig = DefaultAgentConfiguration()
	invalidConfig.Global.DefaultTimeout = 0
	err = invalidConfig.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "default_timeout must be positive")

	// Test invalid agent config - max instances
	invalidConfig = DefaultAgentConfiguration()
	agentConfig := invalidConfig.Agents[SecurityAgentID]
	agentConfig.MaxInstances = 0
	invalidConfig.Agents[SecurityAgentID] = agentConfig
	err = invalidConfig.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_instances must be positive")

	// Test invalid agent config - timeout
	invalidConfig = DefaultAgentConfiguration()
	agentConfig = invalidConfig.Agents[SecurityAgentID]
	agentConfig.Timeout = 0
	invalidConfig.Agents[SecurityAgentID] = agentConfig
	err = invalidConfig.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout must be positive")

	// Test invalid agent config - memory
	invalidConfig = DefaultAgentConfiguration()
	agentConfig = invalidConfig.Agents[SecurityAgentID]
	agentConfig.ResourceLimits.MaxMemoryMB = 0
	invalidConfig.Agents[SecurityAgentID] = agentConfig
	err = invalidConfig.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_memory_mb must be positive")

	// Test invalid agent config - CPU
	invalidConfig = DefaultAgentConfiguration()
	agentConfig = invalidConfig.Agents[SecurityAgentID]
	agentConfig.ResourceLimits.MaxCPUPercent = 150
	invalidConfig.Agents[SecurityAgentID] = agentConfig
	err = invalidConfig.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_cpu_percent must be between 0 and 100")
}

func TestAgentConfigurationGetAgentConfig(t *testing.T) {
	config := DefaultAgentConfiguration()

	// Test getting existing agent config
	securityConfig, exists := config.GetAgentConfig(SecurityAgentID)
	assert.True(t, exists)
	assert.True(t, securityConfig.Enabled)
	assert.Equal(t, PriorityHigh, securityConfig.Priority)

	// Test getting non-existent agent config
	_, exists = config.GetAgentConfig(AgentID("non_existent"))
	assert.False(t, exists)
}

func TestAgentConfigurationIsAgentEnabled(t *testing.T) {
	config := DefaultAgentConfiguration()

	// Test enabled agent
	assert.True(t, config.IsAgentEnabled(SecurityAgentID))

	// Test disabled agent
	config.DisableAgent(SecurityAgentID)
	assert.False(t, config.IsAgentEnabled(SecurityAgentID))

	// Test non-existent agent
	assert.False(t, config.IsAgentEnabled(AgentID("non_existent")))
}

func TestAgentConfigurationGetEnabledAgents(t *testing.T) {
	config := DefaultAgentConfiguration()

	// All default agents should be enabled
	enabled := config.GetEnabledAgents()
	assert.Contains(t, enabled, SecurityAgentID)
	assert.Contains(t, enabled, ArchitectureAgentID)
	assert.Contains(t, enabled, DocumentationAgentID)
	assert.Contains(t, enabled, TestingAgentID)
	assert.Contains(t, enabled, PerformanceAgentID)

	// Disable one agent
	config.DisableAgent(SecurityAgentID)
	enabled = config.GetEnabledAgents()
	assert.NotContains(t, enabled, SecurityAgentID)
	assert.Contains(t, enabled, ArchitectureAgentID)
}

func TestAgentConfigurationUpdateAgentConfig(t *testing.T) {
	config := DefaultAgentConfiguration()

	newConfig := AgentConfig{
		Enabled:       false,
		MaxInstances:  5,
		Timeout:       10 * time.Minute,
		RetryAttempts: 5,
		Priority:      PriorityLow,
		ResourceLimits: ResourceLimitsConfig{
			MaxMemoryMB:   1024,
			MaxCPUPercent: 80,
		},
	}

	err := config.UpdateAgentConfig(SecurityAgentID, newConfig)
	assert.NoError(t, err)

	// Verify config was updated
	updatedConfig, exists := config.GetAgentConfig(SecurityAgentID)
	assert.True(t, exists)
	assert.Equal(t, newConfig, updatedConfig)

	// Test updating with invalid config
	invalidConfig := AgentConfig{
		MaxInstances: 0, // Invalid
	}

	err = config.UpdateAgentConfig(SecurityAgentID, invalidConfig)
	assert.Error(t, err)
}

func TestAgentConfigurationEnableDisableAgent(t *testing.T) {
	config := DefaultAgentConfiguration()

	// Test enable/disable existing agent
	assert.True(t, config.IsAgentEnabled(SecurityAgentID))

	config.DisableAgent(SecurityAgentID)
	assert.False(t, config.IsAgentEnabled(SecurityAgentID))

	config.EnableAgent(SecurityAgentID)
	assert.True(t, config.IsAgentEnabled(SecurityAgentID))

	// Test enable/disable non-existent agent (should not panic)
	config.EnableAgent(AgentID("non_existent"))
	config.DisableAgent(AgentID("non_existent"))
}

func TestAgentConfigurationMerge(t *testing.T) {
	config1 := DefaultAgentConfiguration()
	config2 := &AgentConfiguration{
		Global: GlobalAgentConfig{
			MaxConcurrentAgents: 10, // Different from default
			DefaultTimeout:      45 * time.Minute,
		},
		Agents: map[AgentID]AgentConfig{
			SecurityAgentID: {
				Enabled:      false, // Different from default
				MaxInstances: 3,
			},
			AgentID("new_agent"): {
				Enabled:      true,
				MaxInstances: 1,
			},
		},
		Project: ProjectAgentConfig{
			Type: "python_web_service", // Different from default
		},
		User: UserAgentConfig{
			AutomationLevel: "manual", // Different from default
		},
	}

	originalMaxAgents := config1.Global.MaxConcurrentAgents
	config1.Merge(config2)

	// Test global settings were merged
	assert.Equal(t, 10, config1.Global.MaxConcurrentAgents)
	assert.NotEqual(t, originalMaxAgents, config1.Global.MaxConcurrentAgents)
	assert.Equal(t, 45*time.Minute, config1.Global.DefaultTimeout)

	// Test agent configs were merged
	securityConfig, exists := config1.GetAgentConfig(SecurityAgentID)
	assert.True(t, exists)
	assert.False(t, securityConfig.Enabled) // Should be overridden

	newAgentConfig, exists := config1.GetAgentConfig(AgentID("new_agent"))
	assert.True(t, exists)
	assert.True(t, newAgentConfig.Enabled)

	// Test project and user configs were merged
	assert.Equal(t, "python_web_service", config1.Project.Type)
	assert.Equal(t, "manual", config1.User.AutomationLevel)
}

func TestAgentConfigurationClone(t *testing.T) {
	original := DefaultAgentConfiguration()
	clone := original.Clone()

	// Verify clone is equal but separate
	assert.Equal(t, original.Global, clone.Global)
	assert.Equal(t, original.Project, clone.Project)
	assert.Equal(t, original.User, clone.User)
	assert.Equal(t, len(original.Agents), len(clone.Agents))

	// Verify modifying clone doesn't affect original
	clone.Global.MaxConcurrentAgents = 999
	assert.NotEqual(t, original.Global.MaxConcurrentAgents, clone.Global.MaxConcurrentAgents)

	clone.EnableAgent(AgentID("test_agent"))
	_, existsInOriginal := original.GetAgentConfig(AgentID("test_agent"))
	_, existsInClone := clone.GetAgentConfig(AgentID("test_agent"))
	assert.False(t, existsInOriginal)
	assert.False(t, existsInClone) // EnableAgent doesn't create new configs

	// Test agent config independence
	originalSecurityConfig, _ := original.GetAgentConfig(SecurityAgentID)
	cloneSecurityConfig, _ := clone.GetAgentConfig(SecurityAgentID)

	cloneSecurityConfig.Enabled = false
	clone.Agents[SecurityAgentID] = cloneSecurityConfig

	// Original should be unchanged
	currentOriginalConfig, _ := original.GetAgentConfig(SecurityAgentID)
	assert.Equal(t, originalSecurityConfig.Enabled, currentOriginalConfig.Enabled)
}

func TestResourceLimitsConfig(t *testing.T) {
	limits := ResourceLimitsConfig{
		MaxMemoryMB:      1024,
		MaxCPUPercent:    75.5,
		MaxExecutionTime: 30 * time.Minute,
		MaxFileHandles:   200,
	}

	assert.Equal(t, 1024, limits.MaxMemoryMB)
	assert.Equal(t, 75.5, limits.MaxCPUPercent)
	assert.Equal(t, 30*time.Minute, limits.MaxExecutionTime)
	assert.Equal(t, 200, limits.MaxFileHandles)
}

func TestSecurityAgentConfig(t *testing.T) {
	config := SecurityAgentConfig{
		AgentConfig: AgentConfig{
			Enabled:       true,
			MaxInstances:  2,
			Timeout:       15 * time.Minute,
			RetryAttempts: 3,
		},
		Scanners: SecurityScannersConfig{
			SAST: SASTConfig{
				Enabled:           true,
				Engine:            "semgrep",
				SeverityThreshold: "high",
				ExcludePaths:      []string{"vendor/", "test/"},
			},
			DependencyCheck: DependencyCheckConfig{
				Enabled:         true,
				Databases:       []string{"nvd", "github_advisories"},
				UpdateFrequency: "daily",
				AutoUpdateSafe:  true,
			},
			SecretDetection: SecretDetectionConfig{
				Enabled:            true,
				HighEntropyEnabled: true,
				EntropyThreshold:   4.0,
				ExcludePatterns:    []string{"*.test"},
			},
		},
		AutoFix: SecurityAutoFixConfig{
			Enabled:         true,
			SafeFixesOnly:   true,
			MaxFixesPerRun:  5,
			RequireApproval: true,
			BackupBeforeFix: true,
		},
	}

	assert.True(t, config.Enabled)
	assert.Equal(t, 2, config.MaxInstances)
	assert.True(t, config.Scanners.SAST.Enabled)
	assert.Equal(t, "semgrep", config.Scanners.SAST.Engine)
	assert.True(t, config.Scanners.DependencyCheck.AutoUpdateSafe)
	assert.Equal(t, 4.0, config.Scanners.SecretDetection.EntropyThreshold)
	assert.True(t, config.AutoFix.RequireApproval)
}

func TestArchitectureAgentConfig(t *testing.T) {
	config := ArchitectureAgentConfig{
		ComplexityAnalysis: ComplexityAnalysisConfig{
			CyclomaticThreshold: 10,
			CognitiveThreshold:  15,
			FileSizeLimit:       500,
			NestingDepthLimit:   4,
			ParameterCountLimit: 5,
		},
		DesignPrinciples: DesignPrinciplesConfig{
			SOLID: SOLIDConfig{
				Enforce:    true,
				StrictMode: false,
			},
			DRY: DRYConfig{
				DuplicationThreshold: 3,
				SimilarityPercentage: 0.8,
				MinimumBlockSize:     5,
			},
		},
		Refactoring: RefactoringConfig{
			AutoExtractMethods:       false,
			MethodLengthThreshold:    20,
			SuggestDesignPatterns:    true,
			AutoApplySimpleRefactors: false,
		},
	}

	assert.Equal(t, 10, config.ComplexityAnalysis.CyclomaticThreshold)
	assert.True(t, config.DesignPrinciples.SOLID.Enforce)
	assert.Equal(t, 0.8, config.DesignPrinciples.DRY.SimilarityPercentage)
	assert.True(t, config.Refactoring.SuggestDesignPatterns)
}

func TestTestingAgentConfig(t *testing.T) {
	config := TestingAgentConfig{
		CoverageTargets: CoverageTargetsConfig{
			Unit:        80,
			Integration: 70,
			E2E:         60,
			Mutation:    50,
		},
		TestGeneration: TestGenerationConfig{
			AutoGenerate:      true,
			TestTypes:         []string{"unit", "integration"},
			EdgeCases:         true,
			MockGeneration:    true,
			FixtureGeneration: false,
		},
		TestExecution: TestExecutionConfig{
			Parallel:      true,
			Timeout:       10 * time.Minute,
			RetryAttempts: 2,
			FailFast:      false,
			VerboseOutput: true,
		},
	}

	assert.Equal(t, 80, config.CoverageTargets.Unit)
	assert.True(t, config.TestGeneration.AutoGenerate)
	assert.Contains(t, config.TestGeneration.TestTypes, "unit")
	assert.True(t, config.TestExecution.Parallel)
	assert.Equal(t, 10*time.Minute, config.TestExecution.Timeout)
}

func TestPerformanceAgentConfig(t *testing.T) {
	config := PerformanceAgentConfig{
		Profiling: ProfilingConfig{
			Enabled:      true,
			ProfileTypes: []string{"cpu", "memory", "goroutine"},
			Duration:     30 * time.Second,
			SampleRate:   100,
			AutoProfile:  false,
		},
		Benchmarking: BenchmarkingConfig{
			BenchmarkOnChange:   true,
			RegressionThreshold: "5%",
			BaselineDuration:    1 * time.Minute,
			WarmupTime:          10 * time.Second,
		},
		Optimization: OptimizationConfig{
			OptimizationSuggestions: true,
			AutoOptimizations:       false,
			OptimizationTypes:       []string{"memory", "cpu"},
			PerformanceGoals: map[string]float64{
				"response_time": 100.0,  // ms
				"throughput":    1000.0, // req/s
			},
		},
	}

	assert.True(t, config.Profiling.Enabled)
	assert.Contains(t, config.Profiling.ProfileTypes, "cpu")
	assert.Equal(t, "5%", config.Benchmarking.RegressionThreshold)
	assert.True(t, config.Optimization.OptimizationSuggestions)
	assert.Equal(t, 100.0, config.Optimization.PerformanceGoals["response_time"])
}

func TestDocumentationAgentConfig(t *testing.T) {
	config := DocumentationAgentConfig{
		CoverageRequirements: CoverageRequirementsConfig{
			PublicAPIs:      90,
			InternalAPIs:    70,
			TypeDefinitions: 80,
			Examples:        50,
		},
		AutoGeneration: AutoGenerationConfig{
			README:    true,
			Changelog: true,
			APIDocs:   true,
			Examples:  false,
			Tutorials: false,
		},
		Templates: DocumentationTemplatesConfig{
			FunctionTemplate: "## Function: {{.Name}}\n{{.Description}}",
			ClassTemplate:    "# Class: {{.Name}}\n{{.Description}}",
			READMETemplate:   "# {{.ProjectName}}\n{{.Description}}",
			CustomTemplates: map[string]string{
				"api": "API: {{.Name}}",
			},
		},
	}

	assert.Equal(t, 90, config.CoverageRequirements.PublicAPIs)
	assert.True(t, config.AutoGeneration.README)
	assert.False(t, config.AutoGeneration.Examples)
	assert.Contains(t, config.Templates.FunctionTemplate, "{{.Name}}")
	assert.Equal(t, "API: {{.Name}}", config.Templates.CustomTemplates["api"])
}

func TestWorkflowOverrides(t *testing.T) {
	overrides := WorkflowOverrides{
		RequireSecurityReview:      true,
		RequireArchitectureReview:  false,
		RequirePerformanceAnalysis: true,
		AutoMergeEnabled:           false,
		ParallelExecution:          true,
	}

	assert.True(t, overrides.RequireSecurityReview)
	assert.False(t, overrides.RequireArchitectureReview)
	assert.True(t, overrides.RequirePerformanceAnalysis)
	assert.False(t, overrides.AutoMergeEnabled)
	assert.True(t, overrides.ParallelExecution)
}

func TestQualityGatesConfig(t *testing.T) {
	gates := QualityGatesConfig{
		MinimumTestCoverage:   85,
		MaximumComplexity:     12,
		SecurityScanRequired:  true,
		DocumentationRequired: true,
		PerformanceThreshold:  "3%",
		MaxTechnicalDebt:      5.0,
	}

	assert.Equal(t, 85, gates.MinimumTestCoverage)
	assert.Equal(t, 12, gates.MaximumComplexity)
	assert.True(t, gates.SecurityScanRequired)
	assert.Equal(t, "3%", gates.PerformanceThreshold)
	assert.Equal(t, 5.0, gates.MaxTechnicalDebt)
}

func TestUserAgentConfig(t *testing.T) {
	config := UserAgentConfig{
		AutomationLevel: "collaborative",
		NotificationPreferences: NotificationPreferences{
			AgentProgress:        true,
			CriticalIssues:       true,
			CompletionSummary:    false,
			AutoFixNotifications: true,
			ConflictResolutions:  false,
		},
		UI: UIPreferences{
			Theme:            "dark",
			Layout:           "compact",
			RealTimeUpdates:  true,
			ShowAgentDetails: false,
			AnimationSpeed:   "fast",
		},
	}

	assert.Equal(t, "collaborative", config.AutomationLevel)
	assert.True(t, config.NotificationPreferences.CriticalIssues)
	assert.False(t, config.NotificationPreferences.CompletionSummary)
	assert.Equal(t, "dark", config.UI.Theme)
	assert.Equal(t, "compact", config.UI.Layout)
	assert.Equal(t, "fast", config.UI.AnimationSpeed)
}

func TestMessageBusConfig(t *testing.T) {
	config := MessageBusConfig{
		BufferSize:      2000,
		MaxMessageSize:  2 * 1024 * 1024, // 2MB
		RetryAttempts:   5,
		RetryDelay:      2 * time.Second,
		PersistMessages: false,
	}

	assert.Equal(t, 2000, config.BufferSize)
	assert.Equal(t, 2*1024*1024, config.MaxMessageSize)
	assert.Equal(t, 5, config.RetryAttempts)
	assert.Equal(t, 2*time.Second, config.RetryDelay)
	assert.False(t, config.PersistMessages)
}

func TestHealthCheckConfig(t *testing.T) {
	config := HealthCheckConfig{
		Interval:           45 * time.Second,
		Timeout:            15 * time.Second,
		MaxFailures:        5,
		RecoveryTimeout:    10 * time.Minute,
		EnableAutoRecovery: false,
	}

	assert.Equal(t, 45*time.Second, config.Interval)
	assert.Equal(t, 15*time.Second, config.Timeout)
	assert.Equal(t, 5, config.MaxFailures)
	assert.Equal(t, 10*time.Minute, config.RecoveryTimeout)
	assert.False(t, config.EnableAutoRecovery)
}
