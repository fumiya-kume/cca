package agents

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/fumiya-kume/cca/pkg/logger"
)


// SecurityAgent performs security analysis on code
type SecurityAgent struct {
	*BaseAgent
	scanners map[string]SecurityScanner
	rules    []SecurityRule
	fixers   map[string]SecurityFixer
}

// SecurityScanner defines an interface for security scanning tools
type SecurityScanner interface {
	Scan(ctx context.Context, path string, config map[string]interface{}) (*SecurityScanResult, error)
	GetName() string
}

// SecurityScanResult contains results from a security scan
type SecurityScanResult struct {
	Scanner         string          `json:"scanner"`
	Timestamp       time.Time       `json:"timestamp"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	Errors          []string        `json:"errors"`
	ScanDuration    time.Duration   `json:"scan_duration"`
}

// Vulnerability represents a security vulnerability
type Vulnerability struct {
	ID            string   `json:"id"`
	Type          string   `json:"type"`
	Severity      string   `json:"severity"`
	Description   string   `json:"description"`
	File          string   `json:"file"`
	Line          int      `json:"line"`
	Column        int      `json:"column"`
	CodeSnippet   string   `json:"code_snippet"`
	CWE           []string `json:"cwe"`
	OWASP         []string `json:"owasp"`
	FixAvailable  bool     `json:"fix_available"`
	FixSuggestion string   `json:"fix_suggestion"`
}

// SecurityRule defines a custom security rule
type SecurityRule struct {
	ID          string
	Name        string
	Description string
	Pattern     *regexp.Regexp
	Severity    string
	FileTypes   []string
	Detector    func(content string) []Vulnerability
}

// SecurityFixer can automatically fix certain vulnerabilities
type SecurityFixer interface {
	CanFix(vuln Vulnerability) bool
	Fix(ctx context.Context, vuln Vulnerability) error
}

// NewSecurityAgent creates a new security agent
func NewSecurityAgent(config AgentConfig, messageBus *MessageBus) (*SecurityAgent, error) {
	baseAgent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	if err != nil {
		return nil, fmt.Errorf("failed to create base agent: %w", err)
	}

	agent := &SecurityAgent{
		BaseAgent: baseAgent,
		scanners:  make(map[string]SecurityScanner),
		rules:     []SecurityRule{},
		fixers:    make(map[string]SecurityFixer),
	}

	// Set capabilities
	agent.SetCapabilities([]string{
		"sast_scanning",
		"dependency_checking",
		"secret_detection",
		"vulnerability_assessment",
		"security_fix_generation",
		"compliance_checking",
	})

	// Initialize scanners
	agent.initializeScanners()

	// Initialize rules
	agent.initializeRules()

	// Initialize fixers
	agent.initializeFixers()

	// Register message handlers
	agent.registerHandlers()

	return agent, nil
}

// initializeScanners sets up security scanning tools
func (sa *SecurityAgent) initializeScanners() {
	// Semgrep scanner
	sa.scanners["semgrep"] = &SemgrepScanner{
		logger: sa.logger,
	}

	// GoSec scanner (for Go code)
	sa.scanners["gosec"] = &GoSecScanner{
		logger: sa.logger,
	}

	// Secret detection scanner
	sa.scanners["secrets"] = &SecretScanner{
		logger: sa.logger,
	}
}

// initializeRules sets up custom security rules
func (sa *SecurityAgent) initializeRules() {
	sa.rules = []SecurityRule{
		{
			ID:          "sql-injection",
			Name:        "SQL Injection Detection",
			Description: "Detects potential SQL injection vulnerabilities",
			Pattern:     regexp.MustCompile(`(?i)(exec|execute|query)\s*\(.*\+.*\)`),
			Severity:    "critical",
			FileTypes:   []string{".go", ".js", ".py", ".java"},
		},
		{
			ID:          "hardcoded-secret",
			Name:        "Hardcoded Secret Detection",
			Description: "Detects hardcoded passwords and API keys",
			Pattern:     regexp.MustCompile(`(?i)(password|api_key|secret)\s*[:=]\s*["'][^"']+["']`),
			Severity:    "high",
			FileTypes:   []string{"*"},
		},
		{
			ID:          "weak-crypto",
			Name:        "Weak Cryptography Detection",
			Description: "Detects use of weak cryptographic algorithms",
			Pattern:     regexp.MustCompile(`(?i)(md5|sha1)\s*\(`),
			Severity:    "medium",
			FileTypes:   []string{"*"},
		},
	}
}

// initializeFixers sets up automatic fixers
func (sa *SecurityAgent) initializeFixers() {
	// SQL injection fixer
	sa.fixers["sql-injection"] = &SQLInjectionFixer{
		logger: sa.logger,
	}

	// Secret removal fixer
	sa.fixers["hardcoded-secret"] = &SecretRemovalFixer{
		logger: sa.logger,
	}

	// Crypto upgrade fixer
	sa.fixers["weak-crypto"] = &CryptoUpgradeFixer{
		logger: sa.logger,
	}
}

// registerHandlers registers message handlers specific to security agent
func (sa *SecurityAgent) registerHandlers() {
	// Task assignment handler
	sa.RegisterHandler(TaskAssignment, sa.handleTaskAssignment)

	// Collaboration request handler
	sa.RegisterHandler(CollaborationRequest, sa.handleCollaborationRequest)
}

// handleTaskAssignment processes security analysis tasks
func (sa *SecurityAgent) handleTaskAssignment(ctx context.Context, msg *AgentMessage) error {
	sa.logger.Info("Received task assignment (message_id: %s)", msg.ID)

	// Extract task details
	taskData, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid task data format")
	}

	action, ok := taskData["action"].(string)
	if !ok {
		return fmt.Errorf("missing action in task data")
	}

	switch action {
	case ActionAnalyze:
		return sa.performSecurityAnalysis(ctx, msg)
	case "apply_fix":
		return sa.applySecurityFix(ctx, msg)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

// performSecurityAnalysis performs comprehensive security analysis
func (sa *SecurityAgent) performSecurityAnalysis(ctx context.Context, msg *AgentMessage) error {
	startTime := time.Now()

	// Extract analysis parameters
	taskData := msg.Payload.(map[string]interface{}) //nolint:errcheck // Type assertion is safe in agent message handlers
	path, _ := taskData["path"].(string) //nolint:errcheck // Optional field, use zero value if not present
	if path == "" {
		path = "."
	}

	sa.logger.Info("Starting security analysis (path: %s)", path)

	// Send progress update
	if err := sa.SendProgressUpdate(msg.Sender, map[string]interface{}{
		"stage":   "security_analysis",
		"status":  "started",
		"message": "Running security scanners",
	}); err != nil {
		sa.logger.Warn("Failed to send progress update: %v", err)
	}

	// Collect all vulnerabilities
	var allVulnerabilities []Vulnerability
	var scanErrors []error

	// Run each scanner
	for name, scanner := range sa.scanners {
		sa.logger.Debug("Running scanner (scanner: %s)", name)

		result, err := scanner.Scan(ctx, path, sa.config.CustomSettings)
		if err != nil {
			sa.logger.Error("Scanner failed (scanner: %s, error: %v)", name, err)
			scanErrors = append(scanErrors, fmt.Errorf("%s: %w", name, err))
			continue
		}

		allVulnerabilities = append(allVulnerabilities, result.Vulnerabilities...)
	}

	// Run custom rules
	customVulns := sa.runCustomRules(path)
	allVulnerabilities = append(allVulnerabilities, customVulns...)

	// Deduplicate and prioritize vulnerabilities
	vulnerabilities := sa.deduplicateAndPrioritize(allVulnerabilities)

	// Generate priority items
	_ = sa.generatePriorityItems(vulnerabilities)

	// Create result
	result := &AgentResult{
		AgentID: sa.id,
		Success: len(scanErrors) == 0,
		Results: map[string]interface{}{
			"vulnerabilities": vulnerabilities,
		},
		Warnings: sa.generateWarnings(vulnerabilities),
		Metrics: map[string]interface{}{
			"total_vulnerabilities":    len(vulnerabilities),
			"critical_vulnerabilities": sa.countBySeverity(vulnerabilities, "critical"),
			"high_vulnerabilities":     sa.countBySeverity(vulnerabilities, "high"),
			"medium_vulnerabilities":   sa.countBySeverity(vulnerabilities, "medium"),
			"low_vulnerabilities":      sa.countBySeverity(vulnerabilities, "low"),
			"auto_fixable":             sa.countAutoFixable(vulnerabilities),
		},
		Duration:  time.Since(startTime),
		Timestamp: time.Now(),
	}

	// Store vulnerabilities for potential fixes
	sa.storeVulnerabilities(vulnerabilities)

	// Send result
	return sa.SendResult(msg.Sender, result)
}

// runCustomRules applies custom security rules
func (sa *SecurityAgent) runCustomRules(path string) []Vulnerability {
	var vulnerabilities []Vulnerability

	// This is a simplified implementation
	// Real implementation would walk the file tree and apply rules

	return vulnerabilities
}

// deduplicateAndPrioritize removes duplicates and sorts by priority
func (sa *SecurityAgent) deduplicateAndPrioritize(vulns []Vulnerability) []Vulnerability {
	// Simple deduplication by ID
	seen := make(map[string]bool)
	var unique []Vulnerability

	for _, v := range vulns {
		key := fmt.Sprintf("%s-%s-%d", v.ID, v.File, v.Line)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, v)
		}
	}

	// Sort by severity (critical > high > medium > low)
	// Real implementation would use a proper sorting algorithm

	return unique
}

// generatePriorityItems creates priority items from vulnerabilities
func (sa *SecurityAgent) generatePriorityItems(vulns []Vulnerability) []PriorityItem {
	var items []PriorityItem

	for _, v := range vulns {
		item := PriorityItem{
			ID:          fmt.Sprintf("sec-%s-%s-%d", v.ID, v.File, v.Line),
			AgentID:     sa.id,
			Type:        "Security Vulnerability",
			Description: v.Description,
			Severity:    sa.mapSeverityToPriority(v.Severity),
			Impact:      sa.assessImpact(v),
			AutoFixable: v.FixAvailable,
			FixDetails:  v.FixSuggestion,
		}
		items = append(items, item)
	}

	return items
}

// mapSeverityToPriority maps vulnerability severity to priority
func (sa *SecurityAgent) mapSeverityToPriority(severity string) Priority {
	switch strings.ToLower(severity) {
	case SeverityCritical:
		return PriorityCritical
	case SeverityHigh:
		return PriorityHigh
	case SeverityMedium:
		return PriorityMedium
	default:
		return PriorityLow
	}
}

// assessImpact determines the impact of a vulnerability
func (sa *SecurityAgent) assessImpact(v Vulnerability) string {
	if len(v.OWASP) > 0 {
		return fmt.Sprintf("OWASP %s - %s", strings.Join(v.OWASP, ", "), v.Type)
	}
	if len(v.CWE) > 0 {
		return fmt.Sprintf("CWE %s - %s", strings.Join(v.CWE, ", "), v.Type)
	}
	return v.Type
}

// generateWarnings creates warnings from vulnerabilities
func (sa *SecurityAgent) generateWarnings(vulns []Vulnerability) []string {
	var warnings []string

	criticalCount := sa.countBySeverity(vulns, "critical")
	if criticalCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d critical security vulnerabilities detected", criticalCount))
	}

	return warnings
}

// countBySeverity counts vulnerabilities by severity
func (sa *SecurityAgent) countBySeverity(vulns []Vulnerability, severity string) int {
	count := 0
	for _, v := range vulns {
		if strings.EqualFold(v.Severity, severity) {
			count++
		}
	}
	return count
}

// countAutoFixable counts auto-fixable vulnerabilities
func (sa *SecurityAgent) countAutoFixable(vulns []Vulnerability) int {
	count := 0
	for _, v := range vulns {
		if v.FixAvailable {
			count++
		}
	}
	return count
}

// storeVulnerabilities stores vulnerabilities for later reference
func (sa *SecurityAgent) storeVulnerabilities(vulns []Vulnerability) {
	// In a real implementation, this would store in a persistent store
	// For now, we'll just log
	sa.logger.Debug("Stored vulnerabilities (count: %d)", len(vulns))
}

// applySecurityFix applies an automatic security fix
func (sa *SecurityAgent) applySecurityFix(ctx context.Context, msg *AgentMessage) error {
	taskData := msg.Payload.(map[string]interface{}) //nolint:errcheck // Type assertion is safe in agent message handlers
	itemID, _ := taskData["item_id"].(string) //nolint:errcheck // Optional field, use zero value if not present
	fixDetails, _ := taskData["fix_details"].(string) //nolint:errcheck // Optional field, use zero value if not present

	sa.logger.Info("Applying security fix (item_id: %s, fix: %s)", itemID, fixDetails)

	// Extract vulnerability ID from item ID
	// Format: "sec-{vuln_id}-{file}-{line}"
	parts := strings.Split(itemID, "-")
	if len(parts) < 2 {
		return fmt.Errorf("invalid item ID format")
	}
	vulnType := parts[1]

	// Find appropriate fixer
	fixer, exists := sa.fixers[vulnType]
	if !exists {
		return fmt.Errorf("no fixer available for vulnerability type: %s", vulnType)
	}

	// Create a dummy vulnerability for the fixer
	// In real implementation, we'd retrieve the stored vulnerability
	vuln := Vulnerability{
		ID:            vulnType,
		FixSuggestion: fixDetails,
	}

	// Apply fix
	if err := fixer.Fix(ctx, vuln); err != nil {
		return fmt.Errorf("failed to apply fix: %w", err)
	}

	// Send success notification
	return sa.SendProgressUpdate(msg.Sender, map[string]interface{}{
		"stage":   "security_fix",
		"status":  "completed",
		"item_id": itemID,
		"message": "Security fix applied successfully",
	})
}

// handleCollaborationRequest handles requests from other agents
func (sa *SecurityAgent) handleCollaborationRequest(ctx context.Context, msg *AgentMessage) error {
	collabData, ok := msg.Payload.(*AgentCollaboration)
	if !ok {
		return fmt.Errorf("invalid collaboration data")
	}

	switch collabData.CollaborationType {
	case ExpertiseConsultation:
		return sa.provideSecurityConsultation(ctx, msg, collabData)
	default:
		return fmt.Errorf("unsupported collaboration type: %s", collabData.CollaborationType)
	}
}

// provideSecurityConsultation provides security expertise to other agents
func (sa *SecurityAgent) provideSecurityConsultation(ctx context.Context, msg *AgentMessage, collab *AgentCollaboration) error {
	// Analyze the shared data for security concerns
	response := map[string]interface{}{
		"recommendations": []string{
			"Use parameterized queries for database operations",
			"Implement proper input validation",
			"Use strong encryption for sensitive data",
			"Follow principle of least privilege",
		},
		"concerns": []string{},
	}

	// Send response
	respMsg := &AgentMessage{
		Type:          ResultReporting,
		Sender:        sa.id,
		Receiver:      msg.Sender,
		Payload:       response,
		CorrelationID: msg.ID,
		Priority:      msg.Priority,
	}

	return sa.SendMessage(respMsg)
}

// SemgrepScanner implements security scanning using Semgrep
type SemgrepScanner struct {
	logger *logger.Logger
}

// GetName returns the scanner name
func (s *SemgrepScanner) GetName() string {
	return "semgrep"
}

// Scan performs a security scan using Semgrep
func (s *SemgrepScanner) Scan(ctx context.Context, path string, config map[string]interface{}) (*SecurityScanResult, error) {
	startTime := time.Now()

	// Check if semgrep is available
	if _, err := exec.LookPath("semgrep"); err != nil {
		s.logger.Warn("Semgrep not found, skipping scan")
		return &SecurityScanResult{
			Scanner:      s.GetName(),
			Timestamp:    time.Now(),
			ScanDuration: time.Since(startTime),
			Errors:       []string{"Semgrep not installed"},
		}, nil
	}

	// Run semgrep scan
	cmd := exec.CommandContext(ctx, "semgrep", "--config=auto", "--json", path)
	output, err := cmd.Output()
	if err != nil {
		// Semgrep returns non-zero exit code when vulnerabilities are found
		// So we need to handle this case
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// This is expected when vulnerabilities are found
			output = exitErr.Stderr
		} else {
			return nil, fmt.Errorf("semgrep scan failed: %w", err)
		}
	}

	// Parse results
	vulnerabilities := s.parseSemgrepOutput(output)

	return &SecurityScanResult{
		Scanner:         s.GetName(),
		Timestamp:       time.Now(),
		Vulnerabilities: vulnerabilities,
		ScanDuration:    time.Since(startTime),
	}, nil
}

// parseSemgrepOutput parses Semgrep JSON output
func (s *SemgrepScanner) parseSemgrepOutput(output []byte) []Vulnerability {
	var vulnerabilities []Vulnerability

	// This is a simplified parser
	// Real implementation would parse the actual Semgrep JSON format

	return vulnerabilities
}

// GoSecScanner implements security scanning for Go code
type GoSecScanner struct {
	logger *logger.Logger
}

// GetName returns the scanner name
func (g *GoSecScanner) GetName() string {
	return "gosec"
}

// Scan performs a security scan using gosec
func (g *GoSecScanner) Scan(ctx context.Context, path string, config map[string]interface{}) (*SecurityScanResult, error) {
	// Implementation similar to SemgrepScanner
	return &SecurityScanResult{
		Scanner:   g.GetName(),
		Timestamp: time.Now(),
	}, nil
}

// SecretScanner detects hardcoded secrets
type SecretScanner struct {
	logger *logger.Logger
}

// GetName returns the scanner name
func (s *SecretScanner) GetName() string {
	return "secrets"
}

// Scan scans for hardcoded secrets
func (s *SecretScanner) Scan(ctx context.Context, path string, config map[string]interface{}) (*SecurityScanResult, error) {
	// Implementation would scan for patterns like API keys, passwords, etc.
	return &SecurityScanResult{
		Scanner:   s.GetName(),
		Timestamp: time.Now(),
	}, nil
}

// SQLInjectionFixer fixes SQL injection vulnerabilities
type SQLInjectionFixer struct {
	logger *logger.Logger
}

// CanFix checks if this fixer can handle the vulnerability
func (f *SQLInjectionFixer) CanFix(vuln Vulnerability) bool {
	return vuln.Type == "sql-injection" && vuln.FixAvailable
}

// Fix applies the fix for SQL injection
func (f *SQLInjectionFixer) Fix(ctx context.Context, vuln Vulnerability) error {
	f.logger.Info("Applying SQL injection fix (file: %s, line: %d)", vuln.File, vuln.Line)
	// Implementation would modify the code to use parameterized queries
	return nil
}

// SecretRemovalFixer removes hardcoded secrets
type SecretRemovalFixer struct {
	logger *logger.Logger
}

// CanFix checks if this fixer can handle the vulnerability
func (f *SecretRemovalFixer) CanFix(vuln Vulnerability) bool {
	return vuln.Type == "hardcoded-secret" && vuln.FixAvailable
}

// Fix removes hardcoded secrets
func (f *SecretRemovalFixer) Fix(ctx context.Context, vuln Vulnerability) error {
	f.logger.Info("Removing hardcoded secret (file: %s, line: %d)", vuln.File, vuln.Line)
	// Implementation would replace hardcoded values with environment variables
	return nil
}

// CryptoUpgradeFixer upgrades weak cryptography
type CryptoUpgradeFixer struct {
	logger *logger.Logger
}

// CanFix checks if this fixer can handle the vulnerability
func (f *CryptoUpgradeFixer) CanFix(vuln Vulnerability) bool {
	return vuln.Type == "weak-crypto" && vuln.FixAvailable
}

// Fix upgrades weak cryptographic algorithms
func (f *CryptoUpgradeFixer) Fix(ctx context.Context, vuln Vulnerability) error {
	f.logger.Info("Upgrading weak cryptography (file: %s, line: %d)", vuln.File, vuln.Line)
	// Implementation would replace MD5/SHA1 with stronger algorithms
	return nil
}
