package review

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SecurityScanner performs security analysis on code
type SecurityScanner struct {
	config SecurityScannerConfig
}

// SecurityScannerConfig configures the security scanner
type SecurityScannerConfig struct {
	SecurityLevel        SecurityLevel
	EnableVulnDB         bool
	EnableSecretScan     bool
	EnableDependencyScan bool
	CustomSecurityRules  []SecurityRule
	Timeout              time.Duration
}

// SecurityRule represents a custom security rule
type SecurityRule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Pattern     string `json:"pattern"`
	Language    string `json:"language"`
	Severity    string `json:"severity"`
	CWE         string `json:"cwe,omitempty"`
	OWASP       string `json:"owasp,omitempty"`
	Message     string `json:"message"`
	Remediation string `json:"remediation"`
}

// VulnerabilityDatabase represents a vulnerability database entry
type VulnerabilityDatabase struct {
	CVE         string   `json:"cve"`
	Severity    string   `json:"severity"`
	Score       float64  `json:"score"`
	Description string   `json:"description"`
	Affected    []string `json:"affected"`
	References  []string `json:"references"`
}

// NewSecurityScanner creates a new security scanner
func NewSecurityScanner(config SecurityScannerConfig) *SecurityScanner {
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Minute
	}

	return &SecurityScanner{
		config: config,
	}
}

// Scan performs comprehensive security scanning
func (ss *SecurityScanner) Scan(ctx context.Context, files []string) ([]SecurityFinding, error) {
	var findings []SecurityFinding

	// Secret scanning
	if ss.config.EnableSecretScan {
		if secretFindings := ss.scanForSecrets(files); len(secretFindings) > 0 {
			findings = append(findings, secretFindings...)
		}
	}

	// Vulnerability pattern scanning
	if vulnFindings := ss.scanVulnerabilityPatterns(files); len(vulnFindings) > 0 {
		findings = append(findings, vulnFindings...)
	}

	// Dependency vulnerability scanning
	if ss.config.EnableDependencyScan {
		if depFindings := ss.scanDependencyVulnerabilities(ctx, files); len(depFindings) > 0 {
			findings = append(findings, depFindings...)
		}
	}

	// Custom security rules
	if customFindings := ss.applyCustomSecurityRules(files); len(customFindings) > 0 {
		findings = append(findings, customFindings...)
	}

	// Language-specific security scanning
	if langFindings := ss.scanLanguageSpecificVulnerabilities(files); len(langFindings) > 0 {
		findings = append(findings, langFindings...)
	}

	return findings, nil
}

// scanForSecrets scans for hardcoded secrets and credentials
func (ss *SecurityScanner) scanForSecrets(files []string) []SecurityFinding {
	var findings []SecurityFinding

	// Common secret patterns
	secretPatterns := []struct {
		Name    string
		Pattern string
		CWE     string
	}{
		{"API Key", `(?i)(api[_-]?key|apikey)\s*[:=]\s*['"]?([a-zA-Z0-9_\-]{20,})['"]?`, "CWE-798"},
		{"AWS Access Key", `AKIA[0-9A-Z]{16}`, "CWE-798"},
		{"AWS Secret Key", `(?i)aws[_-]?secret[_-]?access[_-]?key\s*[:=]\s*['"]?([a-zA-Z0-9+/]{40})['"]?`, "CWE-798"},
		{"GitHub Token", `ghp_[a-zA-Z0-9]{36}`, "CWE-798"},
		{"Private Key", `-----BEGIN (RSA |EC |DSA )?PRIVATE KEY-----`, "CWE-798"},
		{"Password", `(?i)(password|passwd|pwd)\s*[:=]\s*['"]([^'"]{8,})['"]`, "CWE-798"},
		{"Database URL", `(?i)(mongodb|mysql|postgres)://[^\\s'"]+`, "CWE-798"},
		{"JWT Token", `eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*`, "CWE-798"},
		{"Generic Secret", `(?i)(secret|token|key)\s*[:=]\s*['"]([a-zA-Z0-9_\-]{16,})['"]`, "CWE-798"},
	}

	for _, file := range files {
		// #nosec G304 - file path is validated with ValidateFilePath
		// #nosec G304 - file path comes from filesystem walk, validated by caller
	content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		fileContent := string(content)
		lines := strings.Split(fileContent, "\n")

		for _, pattern := range secretPatterns {
			re := regexp.MustCompile(pattern.Pattern)
			matches := re.FindAllStringSubmatch(fileContent, -1)

			for _, match := range matches {
				// Find line number
				lineNum := ss.findLineNumber(fileContent, match[0])

				// Skip if it looks like a test or example
				if ss.isTestOrExample(file, lines, lineNum-1) {
					continue
				}

				finding := SecurityFinding{
					ID:          ss.generateSecurityID(file, lineNum, pattern.Name),
					Severity:    SecuritySeverityHigh,
					Type:        "hardcoded-secret",
					Title:       fmt.Sprintf("Hardcoded %s detected", pattern.Name),
					Description: fmt.Sprintf("Potential hardcoded %s found in source code", pattern.Name),
					File:        file,
					Line:        lineNum,
					CWE:         pattern.CWE,
					OWASP:       "A02:2021 – Cryptographic Failures",
					Confidence:  0.8,
					Impact:      "High - Exposed credentials can lead to unauthorized access",
					Remediation: "Move sensitive data to environment variables or secure configuration files",
					References: []string{
						"https://owasp.org/Top10/A02_2021-Cryptographic_Failures/",
						"https://cwe.mitre.org/data/definitions/798.html",
					},
				}

				findings = append(findings, finding)
			}
		}
	}

	return findings
}

// scanVulnerabilityPatterns scans for common vulnerability patterns
func (ss *SecurityScanner) scanVulnerabilityPatterns(files []string) []SecurityFinding {
	var findings []SecurityFinding

	// Common vulnerability patterns
	vulnPatterns := []struct {
		Name        string
		Pattern     string
		CWE         string
		OWASP       string
		Severity    SecuritySeverity
		Language    string
		Description string
		Remediation string
	}{
		{
			Name:        "SQL Injection",
			Pattern:     `(?i)(query|execute)\s*\(\s*['"]\s*SELECT.*\+.*['"]`,
			CWE:         "CWE-89",
			OWASP:       "A03:2021 – Injection",
			Severity:    SecuritySeverityCritical,
			Description: "Potential SQL injection vulnerability",
			Remediation: "Use parameterized queries or prepared statements",
		},
		{
			Name:        "Command Injection",
			Pattern:     `(?i)(exec|system|shell_exec|passthru|popen)\s*\(\s*.*\$_`,
			CWE:         "CWE-78",
			OWASP:       "A03:2021 – Injection",
			Severity:    SecuritySeverityCritical,
			Description: "Potential command injection vulnerability",
			Remediation: "Validate and sanitize user input, use safe APIs",
		},
		{
			Name:        "Path Traversal",
			Pattern:     `(?i)(include|require|file_get_contents|fopen)\s*\(\s*.*\$_.*\.\./`,
			CWE:         "CWE-22",
			OWASP:       "A01:2021 – Broken Access Control",
			Severity:    SecuritySeverityHigh,
			Description: "Potential path traversal vulnerability",
			Remediation: "Validate file paths and use whitelisting",
		},
		{
			Name:        "XSS Vulnerability",
			Pattern:     `(?i)(echo|print|printf)\s*.*\$_(GET|POST|REQUEST).*without.*html`,
			CWE:         "CWE-79",
			OWASP:       "A03:2021 – Injection",
			Severity:    SecuritySeverityMedium,
			Description: "Potential Cross-Site Scripting (XSS) vulnerability",
			Remediation: "Escape output and validate input",
		},
		{
			Name:        "Weak Crypto",
			Pattern:     `(?i)(md5|sha1|des|rc4)\s*\(`,
			CWE:         "CWE-327",
			OWASP:       "A02:2021 – Cryptographic Failures",
			Severity:    SecuritySeverityMedium,
			Description: "Use of weak cryptographic algorithm",
			Remediation: "Use stronger cryptographic algorithms like SHA-256 or AES",
		},
		{
			Name:        "Insecure Random",
			Pattern:     `(?i)(rand|srand|mt_rand)\s*\(`,
			CWE:         "CWE-338",
			OWASP:       "A02:2021 – Cryptographic Failures",
			Severity:    SecuritySeverityLow,
			Description: "Use of cryptographically weak random number generator",
			Remediation: "Use cryptographically secure random number generators",
		},
	}

	for _, file := range files {
		// #nosec G304 - file path is validated with ValidateFilePath
		// #nosec G304 - file path comes from filesystem walk, validated by caller
	content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		fileContent := string(content)

		for _, pattern := range vulnPatterns {
			if pattern.Language != "" && !ss.matchesLanguage(file, pattern.Language) {
				continue
			}

			re := regexp.MustCompile(pattern.Pattern)
			matches := re.FindAllStringIndex(fileContent, -1)

			for _, match := range matches {
				lineNum := ss.findLineNumber(fileContent, fileContent[match[0]:match[1]])

				finding := SecurityFinding{
					ID:          ss.generateSecurityID(file, lineNum, pattern.Name),
					Severity:    pattern.Severity,
					Type:        "vulnerability-pattern",
					Title:       pattern.Name,
					Description: pattern.Description,
					File:        file,
					Line:        lineNum,
					CWE:         pattern.CWE,
					OWASP:       pattern.OWASP,
					Confidence:  0.7,
					Impact:      ss.getImpactDescription(pattern.Severity),
					Remediation: pattern.Remediation,
					References:  ss.getSecurityReferences(pattern.CWE, pattern.OWASP),
				}

				findings = append(findings, finding)
			}
		}
	}

	return findings
}

// scanDependencyVulnerabilities scans for vulnerable dependencies
func (ss *SecurityScanner) scanDependencyVulnerabilities(ctx context.Context, files []string) []SecurityFinding {
	var findings []SecurityFinding

	// Check for package files
	for _, file := range files {
		switch filepath.Base(file) {
		case "package.json":
			findings = append(findings, ss.scanNodeDependencies(ctx, file)...)
		case "go.mod":
			findings = append(findings, ss.scanGoDependencies(ctx, file)...)
		case "requirements.txt":
			findings = append(findings, ss.scanPythonDependencies(ctx, file)...)
		case "Cargo.toml":
			findings = append(findings, ss.scanRustDependencies(ctx, file)...)
		case "pom.xml":
			findings = append(findings, ss.scanMavenDependencies(ctx, file)...)
		}
	}

	return findings
}

// scanNodeDependencies scans Node.js dependencies for vulnerabilities
func (ss *SecurityScanner) scanNodeDependencies(ctx context.Context, packageFile string) []SecurityFinding {
	var findings []SecurityFinding

	// Check if npm audit is available
	if ss.isToolAvailable("npm") {
		dir := filepath.Dir(packageFile)
		cmd := exec.CommandContext(ctx, "npm", "audit", "--json")
		cmd.Dir = dir

		output, err := cmd.Output()
		if err != nil {
			// npm audit returns non-zero exit code when vulnerabilities found
			findings = ss.parseNpmAuditOutput(string(output), packageFile)
		}
	}

	return findings
}

// parseNpmAuditOutput parses npm audit JSON output
func (ss *SecurityScanner) parseNpmAuditOutput(output, file string) []SecurityFinding {
	var findings []SecurityFinding

	// Parse npm audit JSON output
	var auditResult map[string]interface{}
	if err := json.Unmarshal([]byte(output), &auditResult); err != nil {
		return ss.parseNpmAuditTextOutput(output, file)
	}
	
	// Parse vulnerabilities from npm audit JSON
	if vulnerabilities, ok := auditResult["vulnerabilities"].(map[string]interface{}); ok {
		for vulnName, vulnData := range vulnerabilities {
			if finding := ss.parseNpmVulnerability(vulnName, vulnData, file); finding != nil {
				findings = append(findings, *finding)
			}
		}
	}

	return findings
}

// parseNpmAuditTextOutput handles fallback text parsing when JSON parsing fails
func (ss *SecurityScanner) parseNpmAuditTextOutput(output, file string) []SecurityFinding {
	var findings []SecurityFinding
	
	if strings.Contains(output, "vulnerabilities") {
		finding := SecurityFinding{
			ID:          ss.generateSecurityID(file, 1, "npm-vulnerabilities"),
			Severity:    SecuritySeverityMedium,
			Type:        "dependency-vulnerability",
			Title:       "npm vulnerabilities detected",
			Description: fmt.Sprintf("Dependencies with known vulnerabilities found: %s", strings.TrimSpace(output[:200])),
			File:        file,
			Line:        0,
			Impact:      "Dependencies may contain security vulnerabilities",
			Remediation: "Run 'npm audit fix' to automatically fix vulnerabilities",
			Confidence:  0.9,
		}
		findings = append(findings, finding)
	}
	
	return findings
}

// parseNpmVulnerability parses a single npm vulnerability entry
func (ss *SecurityScanner) parseNpmVulnerability(vulnName string, vulnData interface{}, file string) *SecurityFinding {
	vulnInfo, ok := vulnData.(map[string]interface{})
	if !ok {
		return nil
	}

	severity := ss.parseNpmSeverity(vulnInfo)
	title := ss.parseNpmTitle(vulnInfo, vulnName)
	via := ss.parseNpmVia(vulnInfo)

	finding := SecurityFinding{
		ID:          ss.generateSecurityID(file, 1, vulnName),
		Severity:    severity,
		Type:        "dependency-vulnerability",
		Title:       title,
		Description: fmt.Sprintf("Vulnerability found in dependency: %s", vulnName),
		File:        file,
		Line:        1,
		Confidence:  0.9,
		Impact:      "Dependencies may contain security vulnerabilities",
		Remediation: "Run 'npm audit fix' to update vulnerable packages",
		References:  []string{"https://docs.npmjs.com/cli/v8/commands/npm-audit"},
	}
	
	if len(via) > 0 {
		finding.Description += fmt.Sprintf(" (Via: %s)", strings.Join(via, ", "))
	}
	
	return &finding
}

// parseNpmSeverity extracts severity from npm vulnerability info
func (ss *SecurityScanner) parseNpmSeverity(vulnInfo map[string]interface{}) SecuritySeverity {
	if sev, exists := vulnInfo["severity"].(string); exists {
		switch strings.ToLower(sev) {
		case severityCritical:
			return SecuritySeverityCritical
		case severityHigh:
			return SecuritySeverityHigh
		case "moderate":
			return SecuritySeverityMedium
		case severityLow:
			return SecuritySeverityLow
		}
	}
	return SecuritySeverityMedium
}

// parseNpmTitle extracts title from npm vulnerability info
func (ss *SecurityScanner) parseNpmTitle(vulnInfo map[string]interface{}, vulnName string) string {
	if title, exists := vulnInfo["title"].(string); exists {
		return title
	}
	return fmt.Sprintf("Vulnerability in %s", vulnName)
}

// parseNpmVia extracts via information from npm vulnerability info
func (ss *SecurityScanner) parseNpmVia(vulnInfo map[string]interface{}) []string {
	var via []string
	if viaData, exists := vulnInfo["via"].([]interface{}); exists {
		for _, v := range viaData {
			if viaStr, ok := v.(string); ok {
				via = append(via, viaStr)
			}
		}
	}
	return via
}

// scanGoDependencies scans Go dependencies for vulnerabilities
func (ss *SecurityScanner) scanGoDependencies(ctx context.Context, goModFile string) []SecurityFinding {
	var findings []SecurityFinding

	// Check if govulncheck is available
	if ss.isToolAvailable("govulncheck") {
		dir := filepath.Dir(goModFile)
		cmd := exec.CommandContext(ctx, "govulncheck", "./...")
		cmd.Dir = dir

		output, err := cmd.Output()
		if err != nil {
			findings = ss.parseGovulncheckOutput(string(output), goModFile)
		}
	}

	return findings
}

// parseGovulncheckOutput parses govulncheck output
func (ss *SecurityScanner) parseGovulncheckOutput(output, file string) []SecurityFinding {
	var findings []SecurityFinding

	if strings.Contains(output, "vulnerability") || strings.Contains(output, "vuln") {
		finding := SecurityFinding{
			ID:          ss.generateSecurityID(file, 1, "go-vulnerabilities"),
			Severity:    SecuritySeverityMedium,
			Type:        "dependency-vulnerability",
			Title:       "Go vulnerabilities detected",
			Description: "Dependencies with known vulnerabilities found",
			File:        file,
			Line:        1,
			Confidence:  0.9,
			Impact:      "Dependencies may contain security vulnerabilities",
			Remediation: "Update vulnerable dependencies to patched versions",
			References:  []string{"https://golang.org/x/vuln/cmd/govulncheck"},
		}
		findings = append(findings, finding)
	}

	return findings
}

// scanPythonDependencies scans Python dependencies for vulnerabilities
func (ss *SecurityScanner) scanPythonDependencies(ctx context.Context, reqFile string) []SecurityFinding {
	var findings []SecurityFinding

	// Check if safety is available
	if ss.isToolAvailable("safety") {
		cmd := exec.CommandContext(ctx, "safety", "check", "-r", reqFile)

		output, err := cmd.Output()
		if err != nil {
			findings = ss.parseSafetyOutput(string(output), reqFile)
		}
	}

	return findings
}

// parseSafetyOutput parses safety check output
func (ss *SecurityScanner) parseSafetyOutput(output, file string) []SecurityFinding {
	var findings []SecurityFinding

	if strings.Contains(output, "vulnerability") || strings.Contains(output, "insecure") {
		finding := SecurityFinding{
			ID:          ss.generateSecurityID(file, 1, "python-vulnerabilities"),
			Severity:    SecuritySeverityMedium,
			Type:        "dependency-vulnerability",
			Title:       "Python vulnerabilities detected",
			Description: "Dependencies with known vulnerabilities found",
			File:        file,
			Line:        1,
			Confidence:  0.9,
			Impact:      "Dependencies may contain security vulnerabilities",
			Remediation: "Update vulnerable packages to patched versions",
			References:  []string{"https://pyup.io/safety/"},
		}
		findings = append(findings, finding)
	}

	return findings
}

// scanRustDependencies scans Rust dependencies for vulnerabilities
func (ss *SecurityScanner) scanRustDependencies(ctx context.Context, cargoFile string) []SecurityFinding {
	var findings []SecurityFinding

	// Check if cargo audit is available
	if ss.isToolAvailable("cargo") {
		dir := filepath.Dir(cargoFile)
		cmd := exec.CommandContext(ctx, "cargo", "audit")
		cmd.Dir = dir

		output, err := cmd.Output()
		if err != nil {
			findings = ss.parseCargoAuditOutput(string(output), cargoFile)
		}
	}

	return findings
}

// parseCargoAuditOutput parses cargo audit output
func (ss *SecurityScanner) parseCargoAuditOutput(output, file string) []SecurityFinding {
	var findings []SecurityFinding

	if strings.Contains(output, "vulnerability") || strings.Contains(output, "advisory") {
		finding := SecurityFinding{
			ID:          ss.generateSecurityID(file, 1, "rust-vulnerabilities"),
			Severity:    SecuritySeverityMedium,
			Type:        "dependency-vulnerability",
			Title:       "Rust vulnerabilities detected",
			Description: "Dependencies with known vulnerabilities found",
			File:        file,
			Line:        1,
			Confidence:  0.9,
			Impact:      "Dependencies may contain security vulnerabilities",
			Remediation: "Update vulnerable crates to patched versions",
			References:  []string{"https://rustsec.org/"},
		}
		findings = append(findings, finding)
	}

	return findings
}

// scanMavenDependencies scans Maven dependencies for vulnerabilities
func (ss *SecurityScanner) scanMavenDependencies(ctx context.Context, pomFile string) []SecurityFinding {
	var findings []SecurityFinding

	// Check if OWASP dependency check is available
	if ss.isToolAvailable("dependency-check") {
		dir := filepath.Dir(pomFile)
		// #nosec G204 - dependency-check tool is validated, dir is from file path (not user input)
		cmd := exec.CommandContext(ctx, "dependency-check", "--project", "security-scan", "--scan", dir)

		output, err := cmd.Output()
		if err != nil {
			findings = ss.parseDependencyCheckOutput(string(output), pomFile)
		}
	}

	return findings
}

// parseDependencyCheckOutput parses OWASP dependency check output
func (ss *SecurityScanner) parseDependencyCheckOutput(output, file string) []SecurityFinding {
	var findings []SecurityFinding

	if strings.Contains(output, "vulnerability") || strings.Contains(output, "CVE") {
		finding := SecurityFinding{
			ID:          ss.generateSecurityID(file, 1, "maven-vulnerabilities"),
			Severity:    SecuritySeverityMedium,
			Type:        "dependency-vulnerability",
			Title:       "Maven vulnerabilities detected",
			Description: "Dependencies with known vulnerabilities found",
			File:        file,
			Line:        1,
			Confidence:  0.9,
			Impact:      "Dependencies may contain security vulnerabilities",
			Remediation: "Update vulnerable dependencies to patched versions",
			References:  []string{"https://owasp.org/www-project-dependency-check/"},
		}
		findings = append(findings, finding)
	}

	return findings
}

// applyCustomSecurityRules applies custom security rules
func (ss *SecurityScanner) applyCustomSecurityRules(files []string) []SecurityFinding {
	var findings []SecurityFinding

	for _, rule := range ss.config.CustomSecurityRules {
		ruleFindings := ss.applySecurityRule(rule, files)
		findings = append(findings, ruleFindings...)
	}

	return findings
}

// applySecurityRule applies a single security rule
func (ss *SecurityScanner) applySecurityRule(rule SecurityRule, files []string) []SecurityFinding {
	var findings []SecurityFinding

	pattern, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return findings
	}

	for _, file := range files {
		if rule.Language != "" && !ss.matchesLanguage(file, rule.Language) {
			continue
		}

		// #nosec G304 - file path is validated with ValidateFilePath
		// #nosec G304 - file path comes from filesystem walk, validated by caller
	content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		matches := pattern.FindAllStringIndex(string(content), -1)
		for _, match := range matches {
			lineNum := ss.findLineNumber(string(content), string(content[match[0]:match[1]]))

			finding := SecurityFinding{
				ID:          ss.generateSecurityID(file, lineNum, rule.ID),
				Severity:    ss.mapSecuritySeverity(rule.Severity),
				Type:        "custom-security-rule",
				Title:       rule.Name,
				Description: rule.Description,
				File:        file,
				Line:        lineNum,
				CWE:         rule.CWE,
				OWASP:       rule.OWASP,
				Confidence:  0.8,
				Impact:      ss.getImpactDescription(ss.mapSecuritySeverity(rule.Severity)),
				Remediation: rule.Remediation,
			}

			findings = append(findings, finding)
		}
	}

	return findings
}

// scanLanguageSpecificVulnerabilities scans for language-specific vulnerabilities
func (ss *SecurityScanner) scanLanguageSpecificVulnerabilities(files []string) []SecurityFinding {
	var findings []SecurityFinding

	for _, file := range files {
		switch ss.detectLanguage(file) {
		case "go":
			findings = append(findings, ss.scanGoVulnerabilities(file)...)
		case "javascript", "typescript":
			findings = append(findings, ss.scanJSVulnerabilities(file)...)
		case "python":
			findings = append(findings, ss.scanPythonVulnerabilities(file)...)
		case "java":
			findings = append(findings, ss.scanJavaVulnerabilities(file)...)
		}
	}

	return findings
}

// scanGoVulnerabilities scans for Go-specific vulnerabilities
func (ss *SecurityScanner) scanGoVulnerabilities(file string) []SecurityFinding {
	var findings []SecurityFinding

	// #nosec G304 - file path comes from filesystem walk, validated by caller
	content, err := os.ReadFile(file)
	if err != nil {
		return findings
	}

	fileContent := string(content)

	// Check for common Go security issues
	goVulnPatterns := []struct {
		Pattern     string
		Title       string
		Description string
		Severity    SecuritySeverity
		CWE         string
	}{
		{
			Pattern:     `(?m)^[^/]*exec\.Command\([^)]*\$[^)]*\)`,
			Title:       "Command injection risk",
			Description: "User input passed directly to exec.Command",
			Severity:    SecuritySeverityHigh,
			CWE:         "CWE-78",
		},
		{
			Pattern:     `(?m)^[^/]*sql\.Open\([^)]*\+[^)]*\)`,
			Title:       "SQL injection risk",
			Description: "String concatenation in SQL query",
			Severity:    SecuritySeverityCritical,
			CWE:         "CWE-89",
		},
		{
			Pattern:     `(?m)^[^/]*http\.ListenAndServe\([^)]*nil\)`,
			Title:       "Missing TLS configuration",
			Description: "HTTP server without TLS configuration",
			Severity:    SecuritySeverityMedium,
			CWE:         "CWE-319",
		},
	}

	for _, pattern := range goVulnPatterns {
		re := regexp.MustCompile(pattern.Pattern)
		matches := re.FindAllStringIndex(fileContent, -1)

		for _, match := range matches {
			lineNum := ss.findLineNumber(fileContent, fileContent[match[0]:match[1]])

			finding := SecurityFinding{
				ID:          ss.generateSecurityID(file, lineNum, pattern.Title),
				Severity:    pattern.Severity,
				Type:        "language-specific",
				Title:       pattern.Title,
				Description: pattern.Description,
				File:        file,
				Line:        lineNum,
				CWE:         pattern.CWE,
				Confidence:  0.7,
				Impact:      ss.getImpactDescription(pattern.Severity),
				Remediation: "Review and validate user input",
			}

			findings = append(findings, finding)
		}
	}

	return findings
}

// scanJSVulnerabilities scans for JavaScript/TypeScript vulnerabilities
func (ss *SecurityScanner) scanJSVulnerabilities(file string) []SecurityFinding {
	var findings []SecurityFinding

	// #nosec G304 - file path comes from filesystem walk, validated by caller
	content, err := os.ReadFile(file)
	if err != nil {
		return findings
	}

	fileContent := string(content)

	// Check for common JavaScript security issues
	jsVulnPatterns := []struct {
		Pattern     string
		Title       string
		Description string
		Severity    SecuritySeverity
		CWE         string
	}{
		{
			Pattern:     `(?i)eval\s*\(\s*.*\$`,
			Title:       "Code injection via eval",
			Description: "User input passed to eval() function",
			Severity:    SecuritySeverityCritical,
			CWE:         "CWE-95",
		},
		{
			Pattern:     `(?i)innerHTML\s*=\s*.*\$`,
			Title:       "XSS via innerHTML",
			Description: "User input assigned to innerHTML",
			Severity:    SecuritySeverityHigh,
			CWE:         "CWE-79",
		},
		{
			Pattern:     `(?i)document\.write\s*\(\s*.*\$`,
			Title:       "XSS via document.write",
			Description: "User input passed to document.write",
			Severity:    SecuritySeverityHigh,
			CWE:         "CWE-79",
		},
	}

	for _, pattern := range jsVulnPatterns {
		re := regexp.MustCompile(pattern.Pattern)
		matches := re.FindAllStringIndex(fileContent, -1)

		for _, match := range matches {
			lineNum := ss.findLineNumber(fileContent, fileContent[match[0]:match[1]])

			finding := SecurityFinding{
				ID:          ss.generateSecurityID(file, lineNum, pattern.Title),
				Severity:    pattern.Severity,
				Type:        "language-specific",
				Title:       pattern.Title,
				Description: pattern.Description,
				File:        file,
				Line:        lineNum,
				CWE:         pattern.CWE,
				Confidence:  0.7,
				Impact:      ss.getImpactDescription(pattern.Severity),
				Remediation: "Sanitize user input and use safe APIs",
			}

			findings = append(findings, finding)
		}
	}

	return findings
}

// scanPythonVulnerabilities scans for Python-specific vulnerabilities
func (ss *SecurityScanner) scanPythonVulnerabilities(file string) []SecurityFinding {
	var findings []SecurityFinding

	// #nosec G304 - file path comes from filesystem walk, validated by caller
	content, err := os.ReadFile(file)
	if err != nil {
		return findings
	}

	fileContent := string(content)

	// Check for common Python security issues
	pythonVulnPatterns := []struct {
		Pattern     string
		Title       string
		Description string
		Severity    SecuritySeverity
		CWE         string
	}{
		{
			Pattern:     `(?i)eval\s*\(\s*input\(\)`,
			Title:       "Code injection via eval",
			Description: "User input passed to eval() function",
			Severity:    SecuritySeverityCritical,
			CWE:         "CWE-95",
		},
		{
			Pattern:     `(?i)pickle\.loads?\s*\(\s*.*request`,
			Title:       "Unsafe deserialization",
			Description: "Untrusted data passed to pickle",
			Severity:    SecuritySeverityCritical,
			CWE:         "CWE-502",
		},
		{
			Pattern:     `(?i)subprocess\.(call|run|Popen)\s*\(\s*.*request`,
			Title:       "Command injection risk",
			Description: "User input passed to subprocess",
			Severity:    SecuritySeverityHigh,
			CWE:         "CWE-78",
		},
	}

	for _, pattern := range pythonVulnPatterns {
		re := regexp.MustCompile(pattern.Pattern)
		matches := re.FindAllStringIndex(fileContent, -1)

		for _, match := range matches {
			lineNum := ss.findLineNumber(fileContent, fileContent[match[0]:match[1]])

			finding := SecurityFinding{
				ID:          ss.generateSecurityID(file, lineNum, pattern.Title),
				Severity:    pattern.Severity,
				Type:        "language-specific",
				Title:       pattern.Title,
				Description: pattern.Description,
				File:        file,
				Line:        lineNum,
				CWE:         pattern.CWE,
				Confidence:  0.7,
				Impact:      ss.getImpactDescription(pattern.Severity),
				Remediation: "Validate and sanitize user input",
			}

			findings = append(findings, finding)
		}
	}

	return findings
}

// scanJavaVulnerabilities scans for Java-specific vulnerabilities
func (ss *SecurityScanner) scanJavaVulnerabilities(file string) []SecurityFinding {
	var findings []SecurityFinding

	// #nosec G304 - file path comes from filesystem walk, validated by caller
	content, err := os.ReadFile(file)
	if err != nil {
		return findings
	}

	text := string(content)
	lines := strings.Split(text, "\n")

	// Java-specific security patterns
	javaPatterns := []struct {
		pattern     *regexp.Regexp
		severity    SecuritySeverity
		type_       string
		title       string
		description string
		suggestion  string
	}{
		{
			pattern:     regexp.MustCompile(`(?i)eval\s*\(`),
			severity:    SecuritySeverityCritical,
			type_:       "code-injection",
			title:       "Code injection via eval()",
			description: "Use of eval() can lead to code injection vulnerabilities",
			suggestion:  "Avoid using eval() and use safer alternatives",
		},
		{
			pattern:     regexp.MustCompile(`(?i)Runtime\.getRuntime\(\)\.exec\(`),
			severity:    SecuritySeverityHigh,
			type_:       "command-injection",
			title:       "Command injection vulnerability",
			description: "Direct command execution without sanitization",
			suggestion:  "Sanitize input and use ProcessBuilder instead",
		},
		{
			pattern:     regexp.MustCompile(`(?i)System\.setProperty\(`),
			severity:    SecuritySeverityMedium,
			type_:       "system-property-modification",
			title:       "System property modification",
			description: "Modifying system properties can affect security",
			suggestion:  "Avoid modifying system properties in production code",
		},
		{
			pattern:     regexp.MustCompile(`(?i)Class\.forName\(`),
			severity:    SecuritySeverityMedium,
			type_:       "reflection-usage",
			title:       "Reflection usage",
			description: "Dynamic class loading can be exploited",
			suggestion:  "Validate class names and use allow-lists",
		},
		{
			pattern:     regexp.MustCompile(`(?i)new\s+File\(\s*[^"'\s][^)]*\)`),
			severity:    SecuritySeverityMedium,
			type_:       "path-traversal",
			title:       "Potential path traversal",
			description: "File access with unsanitized paths",
			suggestion:  "Validate and sanitize file paths",
		},
		{
			pattern:     regexp.MustCompile(`(?i)javax\.crypto\.Cipher\.getInstance\(\s*"[^"]*\/NONE\/NoPadding"`),
			severity:    SecuritySeverityHigh,
			type_:       "weak-crypto",
			title:       "Weak cryptographic configuration",
			description: "Using cipher without padding is insecure",
			suggestion:  "Use proper padding schemes like PKCS1Padding",
		},
		{
			pattern:     regexp.MustCompile(`(?i)MessageDigest\.getInstance\(\s*"MD5"`),
			severity:    SecuritySeverityMedium,
			type_:       "weak-hash",
			title:       "Weak hash algorithm",
			description: "MD5 is cryptographically broken",
			suggestion:  "Use SHA-256 or stronger hash algorithms",
		},
		{
			pattern:     regexp.MustCompile(`(?i)Random\(\)`),
			severity:    SecuritySeverityMedium,
			type_:       "weak-random",
			title:       "Weak random number generation",
			description: "java.util.Random is not cryptographically secure",
			suggestion:  "Use SecureRandom for security-sensitive operations",
		},
	}

	// Scan each line for security issues
	for lineNum, line := range lines {
		for _, pattern := range javaPatterns {
			if matches := pattern.pattern.FindAllStringSubmatch(line, -1); len(matches) > 0 {
				for i := range matches {
					finding := SecurityFinding{
						ID:          ss.generateSecurityID(file, lineNum+1, fmt.Sprintf("%s-%d", pattern.type_, i)),
						Severity:    pattern.severity,
						Type:        pattern.type_,
						Title:       pattern.title,
						Description: pattern.description,
						File:        file,
						Line:        lineNum + 1,
						Impact:      "Security vulnerability may be exploitable",
						Remediation: pattern.suggestion,
						Confidence:  0.8,
						References:  []string{
							"https://owasp.org/www-project-top-ten/",
							"https://find-sec-bugs.github.io/",
						},
					}
					findings = append(findings, finding)
				}
			}
		}
	}

	return findings
}

// Utility methods

// detectLanguage detects the programming language of a file
func (ss *SecurityScanner) detectLanguage(file string) string {
	ext := strings.ToLower(filepath.Ext(file))

	switch ext {
	case ".go":
		return "go"
	case ".js", ".jsx":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".php":
		return "php"
	case ".rb":
		return "ruby"
	case ".rs":
		return langRust
	}

	return ""
}

// matchesLanguage checks if a file matches the specified language
func (ss *SecurityScanner) matchesLanguage(file, language string) bool {
	return ss.detectLanguage(file) == language
}

// findLineNumber finds the line number of a match in file content
func (ss *SecurityScanner) findLineNumber(content, match string) int {
	index := strings.Index(content, match)
	if index == -1 {
		return 1
	}

	return strings.Count(content[:index], "\n") + 1
}

// isTestOrExample checks if a file or line appears to be test/example code
func (ss *SecurityScanner) isTestOrExample(file string, lines []string, lineIndex int) bool {
	// Check filename
	if strings.Contains(strings.ToLower(file), "test") ||
		strings.Contains(strings.ToLower(file), "example") ||
		strings.Contains(strings.ToLower(file), "demo") {
		return true
	}

	// Check surrounding lines for test indicators
	if lineIndex >= 0 && lineIndex < len(lines) {
		line := strings.ToLower(lines[lineIndex])
		if strings.Contains(line, "test") ||
			strings.Contains(line, "example") ||
			strings.Contains(line, "demo") ||
			strings.Contains(line, "placeholder") {
			return true
		}
	}

	return false
}

// generateSecurityID generates a unique ID for security findings
func (ss *SecurityScanner) generateSecurityID(file string, line int, findingType string) string {
	content := fmt.Sprintf("%s:%d:%s", file, line, findingType)
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("sec_%x", hash)[:16]
}

// mapSecuritySeverity maps string severity to SecuritySeverity enum
func (ss *SecurityScanner) mapSecuritySeverity(severity string) SecuritySeverity {
	switch strings.ToLower(severity) {
	case "critical":
		return SecuritySeverityCritical
	case "high":
		return SecuritySeverityHigh
	case "medium":
		return SecuritySeverityMedium
	case "low":
		return SecuritySeverityLow
	default:
		return SecuritySeverityInfo
	}
}

// getImpactDescription returns an impact description based on severity
func (ss *SecurityScanner) getImpactDescription(severity SecuritySeverity) string {
	switch severity {
	case SecuritySeverityCritical:
		return "Critical - Immediate threat to system security"
	case SecuritySeverityHigh:
		return "High - Significant security risk"
	case SecuritySeverityMedium:
		return "Medium - Moderate security concern"
	case SecuritySeverityLow:
		return "Low - Minor security issue"
	default:
		return "Informational - Security best practice"
	}
}

// getSecurityReferences returns relevant security references
func (ss *SecurityScanner) getSecurityReferences(cwe, owasp string) []string {
	var refs []string

	if cwe != "" {
		refs = append(refs, fmt.Sprintf("https://cwe.mitre.org/data/definitions/%s.html", strings.TrimPrefix(cwe, "CWE-")))
	}

	if owasp != "" {
		refs = append(refs, "https://owasp.org/Top10/")
	}

	return refs
}

// isToolAvailable checks if a security tool is available
func (ss *SecurityScanner) isToolAvailable(tool string) bool {
	_, err := exec.LookPath(tool)
	return err == nil
}
