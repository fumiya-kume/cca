package pr

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ChecksMonitor monitors CI/CD checks and status
type ChecksMonitor struct {
	config ChecksMonitorConfig
}

// ChecksMonitorConfig configures the checks monitor
type ChecksMonitorConfig struct {
	RequiredChecks    []string
	RetryAttempts     int
	RetryInterval     time.Duration
	Timeout           time.Duration
	FailureThreshold  int
	NotifyOnFailure   bool
	AutoRetryFailures bool
}

// NewChecksMonitor creates a new checks monitor
func NewChecksMonitor(config ChecksMonitorConfig) *ChecksMonitor {
	// Set defaults
	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}
	if config.RetryInterval == 0 {
		config.RetryInterval = time.Minute * 5
	}
	if config.Timeout == 0 {
		config.Timeout = time.Hour * 2
	}

	return &ChecksMonitor{
		config: config,
	}
}

// MonitorChecks monitors checks for a pull request
func (cm *ChecksMonitor) MonitorChecks(ctx context.Context, pr *PullRequest, client GitHubClient) error {
	// Get current checks
	checks, err := client.GetChecks(ctx, pr.Branch)
	if err != nil {
		return fmt.Errorf("failed to get checks: %w", err)
	}

	// Update PR with current checks
	pr.Checks = checks

	// Monitor required checks
	for _, requiredCheck := range cm.config.RequiredChecks {
		if err := cm.monitorCheck(ctx, pr, requiredCheck, client); err != nil {
			return fmt.Errorf("failed to monitor check %s: %w", requiredCheck, err)
		}
	}

	return nil
}

// monitorCheck monitors a specific check
func (cm *ChecksMonitor) monitorCheck(ctx context.Context, pr *PullRequest, checkName string, client GitHubClient) error {
	startTime := time.Now()
	attempts := 0

	for {
		if err := cm.checkTimeout(startTime, checkName); err != nil {
			return err
		}

		currentCheck, err := cm.getCurrentCheck(ctx, pr, checkName, client)
		if err != nil {
			return err
		}

		if currentCheck == nil {
			if err := cm.waitForRetry(ctx); err != nil {
				return err
			}
			continue
		}

		cm.updatePRCheck(pr, *currentCheck)

		result, shouldContinue := cm.handleCheckStatus(currentCheck, checkName, &attempts)
		if !shouldContinue {
			return result
		}

		if err := cm.waitForRetry(ctx); err != nil {
			return err
		}
	}
}

// checkTimeout verifies if the monitoring has exceeded the timeout
func (cm *ChecksMonitor) checkTimeout(startTime time.Time, checkName string) error {
	if time.Since(startTime) > cm.config.Timeout {
		return fmt.Errorf("timeout waiting for check %s", checkName)
	}
	return nil
}

// getCurrentCheck retrieves the current status of a specific check
func (cm *ChecksMonitor) getCurrentCheck(ctx context.Context, pr *PullRequest, checkName string, client GitHubClient) (*CheckStatus, error) {
	checks, err := client.GetChecks(ctx, pr.Branch)
	if err != nil {
		return nil, fmt.Errorf("failed to get checks: %w", err)
	}

	for i, check := range checks {
		if check.Name == checkName {
			return &checks[i], nil
		}
	}

	return nil, nil // Check not found
}

// waitForRetry waits for the retry interval or context cancellation
func (cm *ChecksMonitor) waitForRetry(ctx context.Context) error {
	select {
	case <-time.After(cm.config.RetryInterval):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// handleCheckStatus processes the check status and determines next action
func (cm *ChecksMonitor) handleCheckStatus(currentCheck *CheckStatus, checkName string, attempts *int) (error, bool) {
	switch currentCheck.Status {
	case CheckStateCompleted:
		return cm.handleCompletedCheck(currentCheck, checkName, attempts)
	case CheckStatePending, CheckStateQueued, CheckStateInProgress:
		return nil, true // Continue monitoring
	default:
		return nil, true // Continue for unknown states
	}
}

// handleCompletedCheck processes completed check status
func (cm *ChecksMonitor) handleCompletedCheck(currentCheck *CheckStatus, checkName string, attempts *int) (error, bool) {
	if currentCheck.Conclusion == "success" {
		return nil, false // Success, stop monitoring
	}

	if currentCheck.Conclusion == "failure" {
		*attempts++
		if cm.shouldRetryFailure(*attempts) {
			return nil, true // Retry the check
		}
		return fmt.Errorf("check %s failed: %s", checkName, currentCheck.Output.Summary), false
	}

	return nil, true // Continue for other conclusions
}

// shouldRetryFailure determines if a failed check should be retried
func (cm *ChecksMonitor) shouldRetryFailure(attempts int) bool {
	return cm.config.AutoRetryFailures && attempts < cm.config.RetryAttempts
}

// updatePRCheck updates a check status in the PR
func (cm *ChecksMonitor) updatePRCheck(pr *PullRequest, newCheck CheckStatus) {
	for i, check := range pr.Checks {
		if check.Name == newCheck.Name {
			pr.Checks[i] = newCheck
			return
		}
	}

	// Check not found, add it
	pr.Checks = append(pr.Checks, newCheck)
}

// FailureAnalyzer analyzes check failures and suggests fixes
type FailureAnalyzer struct {
	config FailureAnalyzerConfig
}

// FailureAnalyzerConfig configures the failure analyzer
type FailureAnalyzerConfig struct {
	AutoFix         bool
	SuggestFixes    bool
	AnalyzePatterns bool
	MaxAnalysisTime time.Duration
}

// FailureAnalysis represents the analysis of a failure
type FailureAnalysis struct {
	CheckName    string              `json:"check_name"`
	FailureType  FailureType         `json:"failure_type"`
	Category     FailureCategory     `json:"category"`
	Severity     FailureSeverity     `json:"severity"`
	Description  string              `json:"description"`
	RootCause    string              `json:"root_cause"`
	Suggestions  []FailureSuggestion `json:"suggestions"`
	AutoFixable  bool                `json:"auto_fixable"`
	Confidence   float64             `json:"confidence"`
	RelatedFiles []string            `json:"related_files"`
}

// FailureSuggestion represents a suggestion for fixing a failure
type FailureSuggestion struct {
	Type        SuggestionType `json:"type"`
	Description string         `json:"description"`
	Action      string         `json:"action"`
	File        string         `json:"file,omitempty"`
	Line        int            `json:"line,omitempty"`
	Code        string         `json:"code,omitempty"`
	Confidence  float64        `json:"confidence"`
}

// Enums for failure analysis

type FailureType int

const (
	FailureTypeTest FailureType = iota
	FailureTypeLint
	FailureTypeBuild
	FailureTypeSecurity
	FailureTypePerformance
	FailureTypeDeployment
	FailureTypeTypeCheck
	FailureTypeFormat
)

type FailureCategory int

const (
	FailureCategorySyntax FailureCategory = iota
	FailureCategoryLogic
	FailureCategoryConfiguration
	FailureCategoryDependency
	FailureCategoryEnvironment
	FailureCategoryPermissions
	FailureCategoryNetwork
	FailureCategoryResource
)

type FailureSeverity int

const (
	FailureSeverityLow FailureSeverity = iota
	FailureSeverityMedium
	FailureSeverityHigh
	FailureSeverityCritical
)

type SuggestionType int

const (
	SuggestionTypeCodeFix SuggestionType = iota
	SuggestionTypeConfigChange
	SuggestionTypeDependencyUpdate
	SuggestionTypeEnvironmentFix
	SuggestionTypeDocumentationUpdate
	SuggestionTypeTestUpdate
)

// NewFailureAnalyzer creates a new failure analyzer
func NewFailureAnalyzer(config FailureAnalyzerConfig) *FailureAnalyzer {
	if config.MaxAnalysisTime == 0 {
		config.MaxAnalysisTime = time.Minute * 5
	}

	return &FailureAnalyzer{
		config: config,
	}
}

// AnalyzeFailures analyzes failures in PR checks
func (fa *FailureAnalyzer) AnalyzeFailures(ctx context.Context, pr *PullRequest, client GitHubClient) ([]*FailureAnalysis, error) {
	var analyzes []*FailureAnalysis

	for _, check := range pr.Checks {
		if check.Status == CheckStateCompleted && check.Conclusion == "failure" {
			analysis, err := fa.analyzeFailure(ctx, check, pr)
			if err != nil {
				// Log error but continue with other checks
				fmt.Printf("Warning: failed to analyze failure for check %s: %v\n", check.Name, err)
				continue
			}
			analyzes = append(analyzes, analysis)
		}
	}

	return analyzes, nil
}

// analyzeFailure analyzes a specific check failure
func (fa *FailureAnalyzer) analyzeFailure(ctx context.Context, check CheckStatus, pr *PullRequest) (*FailureAnalysis, error) {
	analysis := &FailureAnalysis{
		CheckName:    check.Name,
		FailureType:  fa.classifyFailureType(check),
		Category:     fa.classifyFailureCategory(check),
		Severity:     fa.assessSeverity(check),
		Description:  check.Output.Summary,
		RootCause:    fa.analyzeRootCause(check),
		Suggestions:  []FailureSuggestion{},
		AutoFixable:  false,
		Confidence:   0.0,
		RelatedFiles: fa.extractRelatedFiles(check),
	}

	// Generate suggestions based on failure type
	suggestions := fa.generateSuggestions(check, pr)
	analysis.Suggestions = suggestions

	// Determine if auto-fixable
	analysis.AutoFixable = fa.isAutoFixable(analysis)

	// Calculate confidence
	analysis.Confidence = fa.calculateConfidence(analysis)

	return analysis, nil
}

// GenerateFixes generates automatic fixes for failures
func (fa *FailureAnalyzer) GenerateFixes(ctx context.Context, analyzes []*FailureAnalysis) []AutomaticFix {
	var fixes []AutomaticFix

	for _, analysis := range analyzes {
		if analysis.AutoFixable {
			for _, suggestion := range analysis.Suggestions {
				if suggestion.Type == SuggestionTypeCodeFix && suggestion.Confidence > 0.8 {
					fix := AutomaticFix{
						Type:        "code-fix",
						Description: suggestion.Description,
						File:        suggestion.File,
						Line:        suggestion.Line,
						Fix:         suggestion.Code,
						Confidence:  suggestion.Confidence,
					}
					fixes = append(fixes, fix)
				}
			}
		}
	}

	return fixes
}

// Helper methods for failure analysis

func (fa *FailureAnalyzer) classifyFailureType(check CheckStatus) FailureType {
	checkName := check.Name
	output := check.Output.Summary + " " + check.Output.Text

	if fa.isTestFailure(checkName, output) {
		return FailureTypeTest
	}
	if fa.isLintFailure(checkName) {
		return FailureTypeLint
	}
	if fa.isBuildFailure(checkName, output) {
		return FailureTypeBuild
	}
	if fa.isSecurityFailure(checkName) {
		return FailureTypeSecurity
	}
	if fa.isPerformanceFailure(checkName) {
		return FailureTypePerformance
	}
	if fa.isDeploymentFailure(checkName) {
		return FailureTypeDeployment
	}
	if fa.isTypeCheckFailure(checkName, output) {
		return FailureTypeTypeCheck
	}
	if fa.isFormatFailure(checkName) {
		return FailureTypeFormat
	}
	return FailureTypeBuild
}

// Helper methods for failure type classification
func (fa *FailureAnalyzer) isTestFailure(checkName, output string) bool {
	return contains(checkName, "test") || contains(output, "test failed")
}

func (fa *FailureAnalyzer) isLintFailure(checkName string) bool {
	return contains(checkName, "lint") || contains(checkName, "eslint") || contains(checkName, "golint")
}

func (fa *FailureAnalyzer) isBuildFailure(checkName, output string) bool {
	return contains(checkName, "build") || contains(output, "build failed")
}

func (fa *FailureAnalyzer) isSecurityFailure(checkName string) bool {
	return contains(checkName, "security") || contains(checkName, "audit")
}

func (fa *FailureAnalyzer) isPerformanceFailure(checkName string) bool {
	return contains(checkName, "performance") || contains(checkName, "perf")
}

func (fa *FailureAnalyzer) isDeploymentFailure(checkName string) bool {
	return contains(checkName, "deploy") || contains(checkName, "deployment")
}

func (fa *FailureAnalyzer) isTypeCheckFailure(checkName, output string) bool {
	return contains(checkName, "type") || contains(output, "type error")
}

func (fa *FailureAnalyzer) isFormatFailure(checkName string) bool {
	return contains(checkName, "format") || contains(checkName, "prettier")
}

func (fa *FailureAnalyzer) classifyFailureCategory(check CheckStatus) FailureCategory {
	output := check.Output.Summary + " " + check.Output.Text

	if fa.isSyntaxCategory(output) {
		return FailureCategorySyntax
	}
	if fa.isLogicCategory(output) {
		return FailureCategoryLogic
	}
	if fa.isConfigCategory(output) {
		return FailureCategoryConfiguration
	}
	if fa.isDependencyCategory(output) {
		return FailureCategoryDependency
	}
	if fa.isPermissionCategory(output) {
		return FailureCategoryPermissions
	}
	if fa.isNetworkCategory(output) {
		return FailureCategoryNetwork
	}
	if fa.isResourceCategory(output) {
		return FailureCategoryResource
	}
	return FailureCategoryEnvironment
}

// Helper methods for failure category classification
func (fa *FailureAnalyzer) isSyntaxCategory(output string) bool {
	return contains(output, "syntax error") || contains(output, "parse error")
}

func (fa *FailureAnalyzer) isLogicCategory(output string) bool {
	return contains(output, "logic error") || contains(output, "assertion")
}

func (fa *FailureAnalyzer) isConfigCategory(output string) bool {
	return contains(output, "config") || contains(output, "configuration")
}

func (fa *FailureAnalyzer) isDependencyCategory(output string) bool {
	return contains(output, "dependency") || contains(output, "import") || contains(output, "module")
}

func (fa *FailureAnalyzer) isPermissionCategory(output string) bool {
	return contains(output, "permission") || contains(output, "access denied")
}

func (fa *FailureAnalyzer) isNetworkCategory(output string) bool {
	return contains(output, "network") || contains(output, "connection")
}

func (fa *FailureAnalyzer) isResourceCategory(output string) bool {
	return contains(output, "memory") || contains(output, "disk") || contains(output, "resource")
}

func (fa *FailureAnalyzer) assessSeverity(check CheckStatus) FailureSeverity {
	output := check.Output.Summary + " " + check.Output.Text

	switch {
	case contains(output, "critical") || contains(output, "fatal"):
		return FailureSeverityCritical
	case contains(output, "error") && !contains(output, "warning"):
		return FailureSeverityHigh
	case contains(output, "warning"):
		return FailureSeverityMedium
	default:
		return FailureSeverityLow
	}
}

func (fa *FailureAnalyzer) analyzeRootCause(check CheckStatus) string {
	output := check.Output.Text

	// Simple pattern matching for common root causes
	patterns := map[string]string{
		"module not found":   "Missing dependency or incorrect import path",
		"syntax error":       "Code syntax is invalid",
		"type error":         "Type mismatch or incorrect type usage",
		"test failed":        "Test assertion failed or test logic error",
		"build failed":       "Compilation or build process error",
		"permission denied":  "Insufficient permissions",
		"connection refused": "Network connectivity issue",
		"out of memory":      "Insufficient memory resources",
		"timeout":            "Operation exceeded time limit",
		"undefined variable": "Variable used before declaration",
		"null pointer":       "Null reference access",
		"assertion failed":   "Test expectation not met",
	}

	for pattern, cause := range patterns {
		if contains(output, pattern) {
			return cause
		}
	}

	return "Root cause analysis pending"
}

func (fa *FailureAnalyzer) extractRelatedFiles(check CheckStatus) []string {
	var files []string
	output := check.Output.Text

	// Extract file paths from common error output patterns
	// This is a simplified implementation
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, ".go:") ||
			strings.Contains(line, ".js:") ||
			strings.Contains(line, ".ts:") ||
			strings.Contains(line, ".py:") {
			// Extract file path (simplified)
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.Contains(part, ":") && (strings.HasSuffix(part, ".go") ||
					strings.HasSuffix(part, ".js") || strings.HasSuffix(part, ".ts") ||
					strings.HasSuffix(part, ".py")) {
					files = append(files, strings.Split(part, ":")[0])
				}
			}
		}
	}

	return files
}

func (fa *FailureAnalyzer) generateSuggestions(check CheckStatus, pr *PullRequest) []FailureSuggestion {
	var suggestions []FailureSuggestion

	failureType := fa.classifyFailureType(check)
	output := check.Output.Text

	switch failureType {
	case FailureTypeTest:
		suggestions = append(suggestions, fa.generateTestSuggestions(output)...)
	case FailureTypeLint:
		suggestions = append(suggestions, fa.generateLintSuggestions(output)...)
	case FailureTypeBuild:
		suggestions = append(suggestions, fa.generateBuildSuggestions(output)...)
	case FailureTypeFormat:
		suggestions = append(suggestions, fa.generateFormatSuggestions(output)...)
	case FailureTypeTypeCheck:
		suggestions = append(suggestions, fa.generateTypeSuggestions(output)...)
	}

	return suggestions
}

func (fa *FailureAnalyzer) generateTestSuggestions(output string) []FailureSuggestion {
	var suggestions []FailureSuggestion

	if contains(output, "assertion failed") || contains(output, "expected") {
		suggestions = append(suggestions, FailureSuggestion{
			Type:        SuggestionTypeTestUpdate,
			Description: "Update test expectations to match actual behavior",
			Action:      "Review and update test assertions",
			Confidence:  0.7,
		})
	}

	if contains(output, "test not found") || contains(output, "no tests") {
		suggestions = append(suggestions, FailureSuggestion{
			Type:        SuggestionTypeTestUpdate,
			Description: "Add missing tests for new functionality",
			Action:      "Create test files for untested code",
			Confidence:  0.8,
		})
	}

	return suggestions
}

func (fa *FailureAnalyzer) generateLintSuggestions(output string) []FailureSuggestion {
	var suggestions []FailureSuggestion

	if contains(output, "unused variable") || contains(output, "unused import") {
		suggestions = append(suggestions, FailureSuggestion{
			Type:        SuggestionTypeCodeFix,
			Description: "Remove unused variables and imports",
			Action:      "Clean up unused code",
			Confidence:  0.9,
		})
	}

	if contains(output, "line too long") || contains(output, "line length") {
		suggestions = append(suggestions, FailureSuggestion{
			Type:        SuggestionTypeCodeFix,
			Description: "Break long lines to meet style guidelines",
			Action:      "Reformat long lines",
			Confidence:  0.9,
		})
	}

	return suggestions
}

func (fa *FailureAnalyzer) generateBuildSuggestions(output string) []FailureSuggestion {
	var suggestions []FailureSuggestion

	if contains(output, "module not found") || contains(output, "package not found") {
		suggestions = append(suggestions, FailureSuggestion{
			Type:        SuggestionTypeDependencyUpdate,
			Description: "Install missing dependencies",
			Action:      "Run package manager install command",
			Confidence:  0.8,
		})
	}

	if contains(output, "syntax error") {
		suggestions = append(suggestions, FailureSuggestion{
			Type:        SuggestionTypeCodeFix,
			Description: "Fix syntax errors in code",
			Action:      "Review and correct syntax",
			Confidence:  0.9,
		})
	}

	return suggestions
}

func (fa *FailureAnalyzer) generateFormatSuggestions(output string) []FailureSuggestion {
	return []FailureSuggestion{
		{
			Type:        SuggestionTypeCodeFix,
			Description: "Run code formatter to fix formatting issues",
			Action:      "Execute formatter command",
			Confidence:  0.95,
		},
	}
}

func (fa *FailureAnalyzer) generateTypeSuggestions(output string) []FailureSuggestion {
	var suggestions []FailureSuggestion

	if contains(output, "type mismatch") || contains(output, "cannot assign") {
		suggestions = append(suggestions, FailureSuggestion{
			Type:        SuggestionTypeCodeFix,
			Description: "Fix type compatibility issues",
			Action:      "Update variable types or add type conversions",
			Confidence:  0.7,
		})
	}

	return suggestions
}

func (fa *FailureAnalyzer) isAutoFixable(analysis *FailureAnalysis) bool {
	// Only certain types of failures can be auto-fixed
	switch analysis.FailureType {
	case FailureTypeFormat:
		return true
	case FailureTypeLint:
		// Only simple lint issues
		return analysis.Category == FailureCategorySyntax
	default:
		return false
	}
}

func (fa *FailureAnalyzer) calculateConfidence(analysis *FailureAnalysis) float64 {
	baseConfidence := 0.5

	// Increase confidence based on failure type clarity
	switch analysis.FailureType {
	case FailureTypeFormat:
		baseConfidence = 0.9
	case FailureTypeLint:
		baseConfidence = 0.8
	case FailureTypeBuild:
		baseConfidence = 0.7
	case FailureTypeTest:
		baseConfidence = 0.6
	}

	// Adjust based on category
	switch analysis.Category {
	case FailureCategorySyntax:
		baseConfidence += 0.1
	case FailureCategoryConfiguration:
		baseConfidence += 0.05
	}

	// Cap at 1.0
	if baseConfidence > 1.0 {
		baseConfidence = 1.0
	}

	return baseConfidence
}

// Utility function
func contains(text, substr string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(substr))
}

// String methods for enums

func (ft FailureType) String() string {
	switch ft {
	case FailureTypeTest:
		return prTypeTest
	case FailureTypeLint:
		return "lint"
	case FailureTypeBuild:
		return "build"
	case FailureTypeSecurity:
		return "security"
	case FailureTypePerformance:
		return "performance"
	case FailureTypeDeployment:
		return "deployment"
	case FailureTypeTypeCheck:
		return "type-check"
	case FailureTypeFormat:
		return "format"
	default:
		return "unknown"
	}
}
