package agents

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSecurityAgent(t *testing.T) {
	config := AgentConfig{
		Enabled:      true,
		MaxInstances: 1,
		Timeout:      time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewSecurityAgent(config, messageBus)
	require.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Equal(t, SecurityAgentID, agent.GetID())
	assert.Equal(t, StatusOffline, agent.GetStatus())
}

func TestSecurityAgent_Lifecycle(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
		Timeout: time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewSecurityAgent(config, messageBus)
	require.NoError(t, err)

	// Test starting the agent
	ctx := context.Background()
	err = agent.Start(ctx)
	require.NoError(t, err)
	assert.Equal(t, StatusIdle, agent.GetStatus())

	// Test health check
	err = agent.HealthCheck(ctx)
	require.NoError(t, err)

	// Test stopping the agent
	err = agent.Stop(ctx)
	require.NoError(t, err)
}

func TestSecurityAgent_GetCapabilities(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewSecurityAgent(config, messageBus)
	require.NoError(t, err)

	capabilities := agent.GetCapabilities()
	assert.NotEmpty(t, capabilities)

	// Should have core security capabilities
	assert.Contains(t, capabilities, "sast_scanning")
	assert.Contains(t, capabilities, "dependency_checking")
	assert.Contains(t, capabilities, "secret_detection")
	assert.Contains(t, capabilities, "vulnerability_assessment")
	assert.Contains(t, capabilities, "security_fix_generation")
	assert.Contains(t, capabilities, "compliance_checking")
}

func TestSecurityAgent_ProcessMessage_TaskAssignment(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
		Timeout: time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewSecurityAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Test security analysis task
	message := &AgentMessage{
		ID:       "sec-msg-1",
		Type:     TaskAssignment,
		Sender:   "test-agent",
		Receiver: SecurityAgentID,
		Payload: map[string]interface{}{
			"action": "analyze",
			"path":   "./testdata",
			"parameters": map[string]interface{}{
				"deep_scan":     true,
				"check_secrets": true,
			},
		},
		Timestamp: time.Now(),
		Priority:  PriorityHigh,
	}

	err = agent.ProcessMessage(context.Background(), message)
	require.NoError(t, err)
}

func TestSecurityAgent_ProcessMessage_ApplyFix(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
		Timeout: time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewSecurityAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Test security fix task - test error case for unknown fixer type
	message := &AgentMessage{
		ID:       "sec-fix-1",
		Type:     TaskAssignment,
		Sender:   "test-agent",
		Receiver: SecurityAgentID,
		Payload: map[string]interface{}{
			"action":      "apply_fix",
			"item_id":     "sec-unknown-file.go-42",
			"fix_details": "Use parameterized queries",
		},
		Timestamp: time.Now(),
		Priority:  PriorityHigh,
	}

	err = agent.ProcessMessage(context.Background(), message)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no fixer available")
}

func TestSecurityAgent_ProcessMessage_ApplyFix_Success(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
		Timeout: time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewSecurityAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Add a valid fixer to the agent for testing
	agent.fixers["test"] = &SQLInjectionFixer{logger: agent.logger}

	// Test security fix task with valid fixer
	message := &AgentMessage{
		ID:       "sec-fix-success",
		Type:     TaskAssignment,
		Sender:   "test-agent",
		Receiver: SecurityAgentID,
		Payload: map[string]interface{}{
			"action":      "apply_fix",
			"item_id":     "sec-test-file.go-42", // This should match the "test" fixer key
			"fix_details": "Use parameterized queries",
		},
		Timestamp: time.Now(),
		Priority:  PriorityHigh,
	}

	err = agent.ProcessMessage(context.Background(), message)
	require.NoError(t, err)
}

func TestSecurityAgent_ProcessMessage_CollaborationRequest(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
		Timeout: time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewSecurityAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Test collaboration request
	message := &AgentMessage{
		ID:       "sec-collab-1",
		Type:     CollaborationRequest,
		Sender:   ArchitectureAgentID,
		Receiver: SecurityAgentID,
		Payload: &AgentCollaboration{
			CollaborationType: ExpertiseConsultation,
			Context: CollaborationContext{
				Purpose: "Need security review for refactoring",
				SharedData: map[string]interface{}{
					"target_files": []string{"auth.go", "crypto.go"},
					"concerns":     []string{"authentication", "encryption"},
				},
			},
		},
		Timestamp: time.Now(),
		Priority:  PriorityMedium,
	}

	err = agent.ProcessMessage(context.Background(), message)
	require.NoError(t, err)
}

func TestSecurityAgent_CountBySeverity(t *testing.T) {
	agent := &SecurityAgent{}

	vulnerabilities := []Vulnerability{
		{Severity: "critical"},
		{Severity: "high"},
		{Severity: "critical"},
		{Severity: "medium"},
		{Severity: "low"},
		{Severity: "high"},
	}

	assert.Equal(t, 2, agent.countBySeverity(vulnerabilities, "critical"))
	assert.Equal(t, 2, agent.countBySeverity(vulnerabilities, "high"))
	assert.Equal(t, 1, agent.countBySeverity(vulnerabilities, "medium"))
	assert.Equal(t, 1, agent.countBySeverity(vulnerabilities, "low"))
	assert.Equal(t, 0, agent.countBySeverity(vulnerabilities, "unknown"))
}

func TestSecurityAgent_CountAutoFixable(t *testing.T) {
	agent := &SecurityAgent{}

	vulnerabilities := []Vulnerability{
		{FixAvailable: true},
		{FixAvailable: false},
		{FixAvailable: true},
		{FixAvailable: true},
		{FixAvailable: false},
	}

	assert.Equal(t, 3, agent.countAutoFixable(vulnerabilities))
}

func TestSecurityAgent_MapSeverityToPriority(t *testing.T) {
	agent := &SecurityAgent{}

	tests := []struct {
		severity string
		expected Priority
	}{
		{"critical", PriorityCritical},
		{"CRITICAL", PriorityCritical},
		{"high", PriorityHigh},
		{"HIGH", PriorityHigh},
		{"medium", PriorityMedium},
		{"MEDIUM", PriorityMedium},
		{"low", PriorityLow},
		{"LOW", PriorityLow},
		{"unknown", PriorityLow},
		{"", PriorityLow},
	}

	for _, tt := range tests {
		result := agent.mapSeverityToPriority(tt.severity)
		assert.Equal(t, tt.expected, result, "Severity %s should map to %s", tt.severity, tt.expected)
	}
}

func TestSecurityAgent_AssessImpact(t *testing.T) {
	agent := &SecurityAgent{}

	tests := []struct {
		name     string
		vuln     Vulnerability
		expected string
	}{
		{
			name: "OWASP vulnerability",
			vuln: Vulnerability{
				Type:  "injection",
				OWASP: []string{"A03:2021"},
			},
			expected: "OWASP A03:2021 - injection",
		},
		{
			name: "CWE vulnerability",
			vuln: Vulnerability{
				Type: "buffer-overflow",
				CWE:  []string{"CWE-120"},
			},
			expected: "CWE CWE-120 - buffer-overflow",
		},
		{
			name: "Both OWASP and CWE (OWASP takes precedence)",
			vuln: Vulnerability{
				Type:  "xss",
				OWASP: []string{"A03:2021"},
				CWE:   []string{"CWE-79"},
			},
			expected: "OWASP A03:2021 - xss",
		},
		{
			name: "Neither OWASP nor CWE",
			vuln: Vulnerability{
				Type: "custom-vulnerability",
			},
			expected: "custom-vulnerability",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agent.assessImpact(tt.vuln)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSecurityAgent_GenerateWarnings(t *testing.T) {
	agent := &SecurityAgent{}

	tests := []struct {
		name            string
		vulnerabilities []Vulnerability
		expectedCount   int
		expectedContent string
	}{
		{
			name: "No critical vulnerabilities",
			vulnerabilities: []Vulnerability{
				{Severity: "high"},
				{Severity: "medium"},
			},
			expectedCount: 0,
		},
		{
			name: "One critical vulnerability",
			vulnerabilities: []Vulnerability{
				{Severity: "critical"},
				{Severity: "high"},
			},
			expectedCount:   1,
			expectedContent: "1 critical security vulnerabilities detected",
		},
		{
			name: "Multiple critical vulnerabilities",
			vulnerabilities: []Vulnerability{
				{Severity: "critical"},
				{Severity: "critical"},
				{Severity: "critical"},
				{Severity: "high"},
			},
			expectedCount:   1,
			expectedContent: "3 critical security vulnerabilities detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := agent.generateWarnings(tt.vulnerabilities)
			assert.Len(t, warnings, tt.expectedCount)
			if tt.expectedCount > 0 {
				assert.Contains(t, warnings[0], tt.expectedContent)
			}
		})
	}
}

func TestSecurityAgent_DeduplicateAndPrioritize(t *testing.T) {
	agent := &SecurityAgent{}

	vulnerabilities := []Vulnerability{
		{ID: "vuln-1", File: "file1.go", Line: 10, Severity: "high"},
		{ID: "vuln-1", File: "file1.go", Line: 10, Severity: "high"}, // Duplicate
		{ID: "vuln-2", File: "file2.go", Line: 20, Severity: "critical"},
		{ID: "vuln-1", File: "file1.go", Line: 15, Severity: "medium"}, // Different line
	}

	result := agent.deduplicateAndPrioritize(vulnerabilities)

	// Should have 3 unique vulnerabilities (duplicates removed)
	assert.Len(t, result, 3)

	// Check that the vulnerabilities are unique by key
	keys := make(map[string]bool)
	for _, v := range result {
		key := v.ID + "-" + v.File + "-" + string(rune(v.Line))
		assert.False(t, keys[key], "Duplicate key found: %s", key)
		keys[key] = true
	}
}

func TestSecurityAgent_GeneratePriorityItems(t *testing.T) {
	agent := &SecurityAgent{
		BaseAgent: &BaseAgent{id: SecurityAgentID},
	}

	vulnerabilities := []Vulnerability{
		{
			ID:            "sql-injection",
			Description:   "SQL injection vulnerability",
			File:          "database.go",
			Line:          42,
			Severity:      "critical",
			FixAvailable:  true,
			FixSuggestion: "Use parameterized queries",
		},
		{
			ID:           "xss",
			Description:  "Cross-site scripting vulnerability",
			File:         "handler.go",
			Line:         15,
			Severity:     "high",
			FixAvailable: false,
		},
	}

	items := agent.generatePriorityItems(vulnerabilities)

	assert.Len(t, items, 2)

	// Check first item
	item1 := items[0]
	assert.Equal(t, "sec-sql-injection-database.go-42", item1.ID)
	assert.Equal(t, SecurityAgentID, item1.AgentID)
	assert.Equal(t, "Security Vulnerability", item1.Type)
	assert.Equal(t, "SQL injection vulnerability", item1.Description)
	assert.Equal(t, PriorityCritical, item1.Severity)
	assert.True(t, item1.AutoFixable)
	assert.Equal(t, "Use parameterized queries", item1.FixDetails)

	// Check second item
	item2 := items[1]
	assert.Equal(t, "sec-xss-handler.go-15", item2.ID)
	assert.Equal(t, SecurityAgentID, item2.AgentID)
	assert.Equal(t, PriorityHigh, item2.Severity)
	assert.False(t, item2.AutoFixable)
}

// Test Scanner Interfaces

func TestSemgrepScanner_GetName(t *testing.T) {
	scanner := &SemgrepScanner{}
	assert.Equal(t, "semgrep", scanner.GetName())
}

func TestGoSecScanner_GetName(t *testing.T) {
	scanner := &GoSecScanner{}
	assert.Equal(t, "gosec", scanner.GetName())
}

func TestSecretScanner_GetName(t *testing.T) {
	scanner := &SecretScanner{}
	assert.Equal(t, "secrets", scanner.GetName())
}

// Test Fixer Interfaces

func TestSQLInjectionFixer_CanFix(t *testing.T) {
	fixer := &SQLInjectionFixer{}

	tests := []struct {
		name     string
		vuln     Vulnerability
		expected bool
	}{
		{
			name: "Can fix SQL injection",
			vuln: Vulnerability{
				Type:         "sql-injection",
				FixAvailable: true,
			},
			expected: true,
		},
		{
			name: "Cannot fix - wrong type",
			vuln: Vulnerability{
				Type:         "xss",
				FixAvailable: true,
			},
			expected: false,
		},
		{
			name: "Cannot fix - no fix available",
			vuln: Vulnerability{
				Type:         "sql-injection",
				FixAvailable: false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fixer.CanFix(tt.vuln)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSecretRemovalFixer_CanFix(t *testing.T) {
	fixer := &SecretRemovalFixer{}

	tests := []struct {
		name     string
		vuln     Vulnerability
		expected bool
	}{
		{
			name: "Can fix hardcoded secret",
			vuln: Vulnerability{
				Type:         "hardcoded-secret",
				FixAvailable: true,
			},
			expected: true,
		},
		{
			name: "Cannot fix - wrong type",
			vuln: Vulnerability{
				Type:         "sql-injection",
				FixAvailable: true,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fixer.CanFix(tt.vuln)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCryptoUpgradeFixer_CanFix(t *testing.T) {
	fixer := &CryptoUpgradeFixer{}

	tests := []struct {
		name     string
		vuln     Vulnerability
		expected bool
	}{
		{
			name: "Can fix weak crypto",
			vuln: Vulnerability{
				Type:         "weak-crypto",
				FixAvailable: true,
			},
			expected: true,
		},
		{
			name: "Cannot fix - wrong type",
			vuln: Vulnerability{
				Type:         "xss",
				FixAvailable: true,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fixer.CanFix(tt.vuln)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test Error Handling

func TestSecurityAgent_ErrorHandling(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewSecurityAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Test invalid message payload
	message := &AgentMessage{
		ID:       "invalid-msg",
		Type:     TaskAssignment,
		Sender:   "test-agent",
		Receiver: SecurityAgentID,
		Payload:  "invalid-payload", // Should be map[string]interface{}
	}

	err = agent.ProcessMessage(context.Background(), message)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task data format")
}

func TestSecurityAgent_InvalidAction(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewSecurityAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Test unknown action
	message := &AgentMessage{
		ID:       "unknown-action",
		Type:     TaskAssignment,
		Sender:   "test-agent",
		Receiver: SecurityAgentID,
		Payload: map[string]interface{}{
			"action": "unknown_action",
		},
	}

	err = agent.ProcessMessage(context.Background(), message)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown action")
}

func TestSecurityAgent_Metrics(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewSecurityAgent(config, messageBus)
	require.NoError(t, err)

	// Get initial metrics
	metrics := agent.GetMetrics()
	assert.NotNil(t, &metrics) // Use pointer to avoid copying lock
	assert.Equal(t, int64(0), metrics.MessagesReceived)
	assert.Equal(t, int64(0), metrics.MessagesProcessed)
}

// Test Data Structures

func TestVulnerability_Structure(t *testing.T) {
	vuln := Vulnerability{
		ID:            "SEC-001",
		Type:          "sql-injection",
		Severity:      "critical",
		Description:   "SQL injection vulnerability in user query",
		File:          "user_service.go",
		Line:          142,
		Column:        25,
		CodeSnippet:   `query := "SELECT * FROM users WHERE id = " + userID`,
		CWE:           []string{"CWE-89"},
		OWASP:         []string{"A03:2021"},
		FixAvailable:  true,
		FixSuggestion: "Use parameterized queries with prepared statements",
	}

	assert.Equal(t, "SEC-001", vuln.ID)
	assert.Equal(t, "sql-injection", vuln.Type)
	assert.Equal(t, "critical", vuln.Severity)
	assert.Equal(t, "SQL injection vulnerability in user query", vuln.Description)
	assert.Equal(t, "user_service.go", vuln.File)
	assert.Equal(t, 142, vuln.Line)
	assert.Equal(t, 25, vuln.Column)
	assert.Contains(t, vuln.CodeSnippet, "SELECT * FROM users")
	assert.Contains(t, vuln.CWE, "CWE-89")
	assert.Contains(t, vuln.OWASP, "A03:2021")
	assert.True(t, vuln.FixAvailable)
	assert.Contains(t, vuln.FixSuggestion, "parameterized queries")
}

func TestSecurityScanResult_Structure(t *testing.T) {
	timestamp := time.Now()
	duration := time.Second * 5

	result := SecurityScanResult{
		Scanner:   "test-scanner",
		Timestamp: timestamp,
		Vulnerabilities: []Vulnerability{
			{ID: "vuln-1", Severity: "high"},
			{ID: "vuln-2", Severity: "medium"},
		},
		Errors:       []string{"warning: deprecated API used"},
		ScanDuration: duration,
	}

	assert.Equal(t, "test-scanner", result.Scanner)
	assert.Equal(t, timestamp, result.Timestamp)
	assert.Len(t, result.Vulnerabilities, 2)
	assert.Equal(t, "vuln-1", result.Vulnerabilities[0].ID)
	assert.Equal(t, "high", result.Vulnerabilities[0].Severity)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "deprecated API")
	assert.Equal(t, duration, result.ScanDuration)
}

func TestSecurityRule_Structure(t *testing.T) {
	rule := SecurityRule{
		ID:          "RULE-001",
		Name:        "SQL Injection Detection",
		Description: "Detects potential SQL injection vulnerabilities",
		Severity:    "critical",
		FileTypes:   []string{".go", ".js", ".py"},
	}

	assert.Equal(t, "RULE-001", rule.ID)
	assert.Equal(t, "SQL Injection Detection", rule.Name)
	assert.Equal(t, "Detects potential SQL injection vulnerabilities", rule.Description)
	assert.Equal(t, "critical", rule.Severity)
	assert.Len(t, rule.FileTypes, 3)
	assert.Contains(t, rule.FileTypes, ".go")
	assert.Contains(t, rule.FileTypes, ".js")
	assert.Contains(t, rule.FileTypes, ".py")
}
