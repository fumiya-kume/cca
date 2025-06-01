package review

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// File extension constants
const (
	extPHP = ".php"
	extRb  = ".rb"
)

// StaticAnalyzer performs static code analysis
type StaticAnalyzer struct {
	config StaticAnalyzerConfig
}

// StaticAnalyzerConfig configures the static analyzer
type StaticAnalyzerConfig struct {
	IgnorePatterns []string
	EnableLinters  []string
	CustomRules    []CustomRule
	Timeout        time.Duration
}

// CustomRule represents a custom static analysis rule
type CustomRule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Pattern     string `json:"pattern"`
	Language    string `json:"language"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
}

// NewStaticAnalyzer creates a new static analyzer
func NewStaticAnalyzer(config StaticAnalyzerConfig) *StaticAnalyzer {
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Minute
	}

	return &StaticAnalyzer{
		config: config,
	}
}

// Analyze performs static analysis on the given files
func (sa *StaticAnalyzer) Analyze(ctx context.Context, files []string) (*StaticAnalysisResult, error) {
	result := &StaticAnalysisResult{
		LinterResults:    []LinterResult{},
		TypeCheckResults: []TypeCheckResult{},
		ImportAnalysis:   &ImportAnalysis{},
		DeadCodeReport:   &DeadCodeReport{},
		PerformanceHints: []PerformanceHint{},
	}

	// Group files by language
	filesByLanguage := sa.groupFilesByLanguage(files)

	// Run language-specific analyzers
	for language, langFiles := range filesByLanguage {
		switch language {
		case "go":
			if err := sa.analyzeGoFiles(ctx, langFiles, result); err != nil {
				return result, fmt.Errorf("go analysis failed: %w", err)
			}
		case langJavaScript, langTypeScript:
			if err := sa.analyzeJSFiles(ctx, langFiles, result); err != nil {
				return result, fmt.Errorf("javascript/typescript analysis failed: %w", err)
			}
		case langPython:
			if err := sa.analyzePythonFiles(ctx, langFiles, result); err != nil {
				return result, fmt.Errorf("python analysis failed: %w", err)
			}
		case langJava:
			if err := sa.analyzeJavaFiles(ctx, langFiles, result); err != nil {
				return result, fmt.Errorf("java analysis failed: %w", err)
			}
		case "rust":
			if err := sa.analyzeRustFiles(ctx, langFiles, result); err != nil {
				return result, fmt.Errorf("rust analysis failed: %w", err)
			}
		}
	}

	// Run custom rules
	if err := sa.runCustomRules(ctx, files, result); err != nil {
		return result, fmt.Errorf("custom rules failed: %w", err)
	}

	return result, nil
}

// groupFilesByLanguage groups files by their programming language
func (sa *StaticAnalyzer) groupFilesByLanguage(files []string) map[string][]string {
	groups := make(map[string][]string)

	for _, file := range files {
		if sa.shouldIgnoreFile(file) {
			continue
		}

		language := sa.detectLanguage(file)
		if language != "" {
			groups[language] = append(groups[language], file)
		}
	}

	return groups
}

// detectLanguage detects the programming language of a file
func (sa *StaticAnalyzer) detectLanguage(file string) string {
	ext := strings.ToLower(filepath.Ext(file))

	switch ext {
	case ".go":
		return "go"
	case ".js", ".jsx":
		return langJavaScript
	case ".ts", ".tsx":
		return langTypeScript
	case ".py":
		return langPython
	case ".java":
		return langJava
	case ".rs":
		return "rust"
	case extPHP:
		return "php"
	case extRb:
		return "ruby"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".c":
		return "c"
	case ".cs":
		return "csharp"
	}

	return ""
}

// shouldIgnoreFile checks if a file should be ignored
func (sa *StaticAnalyzer) shouldIgnoreFile(file string) bool {
	for _, pattern := range sa.config.IgnorePatterns {
		if matched, err := filepath.Match(pattern, file); err == nil && matched {
			return true
		}
		// If pattern matching fails, continue to next pattern
	}
	return false
}

// analyzeGoFiles performs Go-specific static analysis
func (sa *StaticAnalyzer) analyzeGoFiles(ctx context.Context, files []string, result *StaticAnalysisResult) error {
	// Run go vet
	if err := sa.runGoVet(ctx, files, result); err != nil {
		return fmt.Errorf("go vet failed: %w", err)
	}

	// Run golint if available
	if sa.isToolAvailable("golint") {
		if err := sa.runGolint(ctx, files, result); err != nil {
			// Log warning but continue
			fmt.Printf("Warning: golint failed: %v\n", err)
		}
	}

	// Run staticcheck if available
	if sa.isToolAvailable("staticcheck") {
		if err := sa.runStaticcheck(ctx, files, result); err != nil {
			// Log warning but continue
			fmt.Printf("Warning: staticcheck failed: %v\n", err)
		}
	}

	// Run gosec for security analysis
	if sa.isToolAvailable("gosec") {
		if err := sa.runGosec(ctx, files, result); err != nil {
			// Log warning but continue
			fmt.Printf("Warning: gosec failed: %v\n", err)
		}
	}

	// Analyze imports
	sa.analyzeGoImports(files, result)

	return nil
}

// runGoVet runs go vet on the files
func (sa *StaticAnalyzer) runGoVet(ctx context.Context, files []string, result *StaticAnalysisResult) error {
	if len(files) == 0 {
		return nil
	}

	// Run go vet on the package directory
	packageDir := filepath.Dir(files[0])

	cmd := exec.CommandContext(ctx, "go", "vet", "./...")
	cmd.Dir = packageDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		// go vet returns non-zero exit code when issues are found
		issues := sa.parseGoVetOutput(string(output))

		linterResult := LinterResult{
			Tool:    "go vet",
			Version: sa.getGoVersion(),
			Issues:  issues,
			Stats: LinterStats{
				FilesScanned: len(files),
				TotalIssues:  len(issues),
				Errors:       len(issues),
			},
		}

		result.LinterResults = append(result.LinterResults, linterResult)
	}

	return nil
}

// parseGoVetOutput parses go vet output into ReviewIssues
func (sa *StaticAnalyzer) parseGoVetOutput(output string) []ReviewIssue {
	var issues []ReviewIssue

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse go vet output format: file:line:col: message
		parts := strings.SplitN(line, ":", 4)
		if len(parts) >= 4 {
			issue := ReviewIssue{
				ID:          generateID(),
				Type:        IssueTypeLogic,
				Severity:    IssueSeverityMedium,
				Title:       "Go vet issue",
				Description: strings.TrimSpace(parts[3]),
				File:        parts[0],
				Line:        parseInt(parts[1]),
				Column:      parseInt(parts[2]),
				Rule:        "go-vet",
				Category:    "static-analysis",
				Fixable:     false,
				Tags:        []string{"go", "vet"},
			}
			issues = append(issues, issue)
		}
	}

	return issues
}

// runGolint runs golint on the files
func (sa *StaticAnalyzer) runGolint(ctx context.Context, files []string, result *StaticAnalysisResult) error {
	for _, file := range files {
		// #nosec G204 - golint tool with validated file paths from static analysis
		cmd := exec.CommandContext(ctx, "golint", file)
		output, err := cmd.CombinedOutput()

		if err != nil && len(output) == 0 {
			continue // Skip if golint not available or other error
		}

		issues := sa.parseGolintOutput(string(output))

		linterResult := LinterResult{
			Tool:     "golint",
			Version:  "unknown",
			Warnings: issues,
			Stats: LinterStats{
				FilesScanned: 1,
				TotalIssues:  len(issues),
				Warnings:     len(issues),
			},
		}

		result.LinterResults = append(result.LinterResults, linterResult)
	}

	return nil
}

// parseGolintOutput parses golint output into ReviewIssues
func (sa *StaticAnalyzer) parseGolintOutput(output string) []ReviewIssue {
	var issues []ReviewIssue

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse golint output format: file:line:col: message
		parts := strings.SplitN(line, ":", 4)
		if len(parts) >= 4 {
			issue := ReviewIssue{
				ID:          generateID(),
				Type:        IssueTypeStyle,
				Severity:    IssueSeverityLow,
				Title:       "Style issue",
				Description: strings.TrimSpace(parts[3]),
				File:        parts[0],
				Line:        parseInt(parts[1]),
				Column:      parseInt(parts[2]),
				Rule:        "golint",
				Category:    "style",
				Fixable:     true,
				Tags:        []string{"go", "style", "lint"},
			}
			issues = append(issues, issue)
		}
	}

	return issues
}

// runStaticcheck runs staticcheck on the files
func (sa *StaticAnalyzer) runStaticcheck(ctx context.Context, files []string, result *StaticAnalysisResult) error {
	if len(files) == 0 {
		return nil
	}

	packageDir := filepath.Dir(files[0])

	cmd := exec.CommandContext(ctx, "staticcheck", "./...")
	cmd.Dir = packageDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		// staticcheck returns non-zero exit code when issues are found
		issues := sa.parseStaticcheckOutput(string(output))

		linterResult := LinterResult{
			Tool:    "staticcheck",
			Version: "unknown",
			Issues:  issues,
			Stats: LinterStats{
				FilesScanned: len(files),
				TotalIssues:  len(issues),
				Errors:       len(issues),
			},
		}

		result.LinterResults = append(result.LinterResults, linterResult)
	}

	return nil
}

// parseStaticcheckOutput parses staticcheck output into ReviewIssues
func (sa *StaticAnalyzer) parseStaticcheckOutput(output string) []ReviewIssue {
	var issues []ReviewIssue

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse staticcheck output format: file:line:col: message (code)
		parts := strings.SplitN(line, ":", 4)
		if len(parts) >= 4 {
			message := strings.TrimSpace(parts[3])

			// Extract rule code if present
			rule := "staticcheck"
			if strings.Contains(message, "(") && strings.Contains(message, ")") {
				re := regexp.MustCompile(`\(([^)]+)\)`)
				matches := re.FindStringSubmatch(message)
				if len(matches) > 1 {
					rule = matches[1]
				}
			}

			issue := ReviewIssue{
				ID:          generateID(),
				Type:        IssueTypeLogic,
				Severity:    IssueSeverityMedium,
				Title:       "Static analysis issue",
				Description: message,
				File:        parts[0],
				Line:        parseInt(parts[1]),
				Column:      parseInt(parts[2]),
				Rule:        rule,
				Category:    "static-analysis",
				Fixable:     false,
				Tags:        []string{"go", "staticcheck"},
			}
			issues = append(issues, issue)
		}
	}

	return issues
}

// runGosec runs gosec security analysis
func (sa *StaticAnalyzer) runGosec(ctx context.Context, files []string, result *StaticAnalysisResult) error {
	if len(files) == 0 {
		return nil
	}

	packageDir := filepath.Dir(files[0])

	cmd := exec.CommandContext(ctx, "gosec", "./...")
	cmd.Dir = packageDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		// gosec returns non-zero exit code when issues are found
		issues := sa.parseGosecOutput(string(output))

		linterResult := LinterResult{
			Tool:    "gosec",
			Version: "unknown",
			Issues:  issues,
			Stats: LinterStats{
				FilesScanned: len(files),
				TotalIssues:  len(issues),
				Errors:       len(issues),
			},
		}

		result.LinterResults = append(result.LinterResults, linterResult)
	}

	return nil
}

// parseGosecOutput parses gosec output into ReviewIssues
func (sa *StaticAnalyzer) parseGosecOutput(output string) []ReviewIssue {
	var issues []ReviewIssue

	// gosec output is more complex, this is a simplified parser
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ".go:") && strings.Contains(line, "G") {
			issue := ReviewIssue{
				ID:          generateID(),
				Type:        IssueTypeSecurity,
				Severity:    IssueSeverityHigh,
				Title:       "Security issue",
				Description: line,
				Rule:        "gosec",
				Category:    "security",
				Fixable:     false,
				Tags:        []string{"go", "security", "gosec"},
			}
			issues = append(issues, issue)
		}
	}

	return issues
}

// analyzeGoImports analyzes Go imports for issues
func (sa *StaticAnalyzer) analyzeGoImports(files []string, result *StaticAnalysisResult) {
	result.ImportAnalysis = &ImportAnalysis{
		UnusedImports:   []string{},
		CircularImports: []string{},
		MissingImports:  []string{},
		ShadowedImports: []string{},
	}

	// This would require more sophisticated analysis
	// For now, it's a placeholder
}

// analyzeJSFiles performs JavaScript/TypeScript analysis
func (sa *StaticAnalyzer) analyzeJSFiles(ctx context.Context, files []string, result *StaticAnalysisResult) error {
	// Run ESLint if available
	if sa.isToolAvailable("eslint") {
		if err := sa.runESLint(ctx, files, result); err != nil {
			fmt.Printf("Warning: ESLint failed: %v\n", err)
		}
	}

	// Run TypeScript compiler if TypeScript files
	hasTS := false
	for _, file := range files {
		if strings.HasSuffix(file, ".ts") || strings.HasSuffix(file, ".tsx") {
			hasTS = true
			break
		}
	}

	if hasTS && sa.isToolAvailable("tsc") {
		if err := sa.runTypeScriptCheck(ctx, files, result); err != nil {
			fmt.Printf("Warning: TypeScript check failed: %v\n", err)
		}
	}

	return nil
}

// runESLint runs ESLint on JavaScript/TypeScript files
func (sa *StaticAnalyzer) runESLint(ctx context.Context, files []string, result *StaticAnalysisResult) error {
	cmd := exec.CommandContext(ctx, "eslint", "--format", "json")
	cmd.Args = append(cmd.Args, files...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// ESLint returns non-zero when issues found
		issues := sa.parseESLintOutput(string(output))

		linterResult := LinterResult{
			Tool:    "eslint",
			Version: "unknown",
			Issues:  issues,
			Stats: LinterStats{
				FilesScanned: len(files),
				TotalIssues:  len(issues),
			},
		}

		result.LinterResults = append(result.LinterResults, linterResult)
	}

	return nil
}

// parseESLintOutput parses ESLint JSON output
func (sa *StaticAnalyzer) parseESLintOutput(output string) []ReviewIssue {
	var issues []ReviewIssue

	// This would parse ESLint JSON output
	// For now, it's a simplified implementation

	return issues
}

// runTypeScriptCheck runs TypeScript type checking
func (sa *StaticAnalyzer) runTypeScriptCheck(ctx context.Context, files []string, result *StaticAnalysisResult) error {
	cmd := exec.CommandContext(ctx, "tsc", "--noEmit", "--pretty", "false")

	output, err := cmd.CombinedOutput()
	if err != nil {
		issues := sa.parseTypeScriptOutput(string(output))

		typeCheckResult := TypeCheckResult{
			Tool:   "tsc",
			Issues: issues,
		}

		result.TypeCheckResults = append(result.TypeCheckResults, typeCheckResult)
	}

	return nil
}

// parseTypeScriptOutput parses TypeScript compiler output
func (sa *StaticAnalyzer) parseTypeScriptOutput(output string) []ReviewIssue {
	var issues []ReviewIssue

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse TypeScript output format
		if strings.Contains(line, ".ts(") || strings.Contains(line, ".tsx(") {
			issue := ReviewIssue{
				ID:          generateID(),
				Type:        IssueTypeSyntax,
				Severity:    IssueSeverityMedium,
				Title:       "Type error",
				Description: line,
				Rule:        "typescript",
				Category:    "type-check",
				Fixable:     false,
				Tags:        []string{"typescript", "type-check"},
			}
			issues = append(issues, issue)
		}
	}

	return issues
}

// analyzePythonFiles performs Python-specific analysis
func (sa *StaticAnalyzer) analyzePythonFiles(ctx context.Context, files []string, result *StaticAnalysisResult) error {
	// Run pylint if available
	if sa.isToolAvailable("pylint") {
		if err := sa.runPylint(ctx, files, result); err != nil {
			fmt.Printf("Warning: pylint failed: %v\n", err)
		}
	}

	// Run flake8 if available
	if sa.isToolAvailable("flake8") {
		if err := sa.runFlake8(ctx, files, result); err != nil {
			fmt.Printf("Warning: flake8 failed: %v\n", err)
		}
	}

	// Run mypy if available
	if sa.isToolAvailable("mypy") {
		if err := sa.runMypy(ctx, files, result); err != nil {
			fmt.Printf("Warning: mypy failed: %v\n", err)
		}
	}

	return nil
}

// runPylint runs pylint on Python files
func (sa *StaticAnalyzer) runPylint(ctx context.Context, files []string, result *StaticAnalysisResult) error {
	cmd := exec.CommandContext(ctx, "pylint", "--output-format=text")
	cmd.Args = append(cmd.Args, files...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		issues := sa.parsePylintOutput(string(output))

		linterResult := LinterResult{
			Tool:    "pylint",
			Version: "unknown",
			Issues:  issues,
			Stats: LinterStats{
				FilesScanned: len(files),
				TotalIssues:  len(issues),
			},
		}

		result.LinterResults = append(result.LinterResults, linterResult)
	}

	return nil
}

// parsePylintOutput parses pylint output
func (sa *StaticAnalyzer) parsePylintOutput(output string) []ReviewIssue {
	var issues []ReviewIssue

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "*") {
			continue
		}

		// Parse pylint output format
		if strings.Contains(line, ".py:") {
			issue := ReviewIssue{
				ID:          generateID(),
				Type:        IssueTypeStyle,
				Severity:    IssueSeverityLow,
				Title:       "Pylint issue",
				Description: line,
				Rule:        "pylint",
				Category:    "style",
				Fixable:     false,
				Tags:        []string{"python", "pylint"},
			}
			issues = append(issues, issue)
		}
	}

	return issues
}

// runFlake8 runs flake8 on Python files
func (sa *StaticAnalyzer) runFlake8(ctx context.Context, files []string, result *StaticAnalysisResult) error {
	cmd := exec.CommandContext(ctx, "flake8")
	cmd.Args = append(cmd.Args, files...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		issues := sa.parseFlake8Output(string(output))

		linterResult := LinterResult{
			Tool:    "flake8",
			Version: "unknown",
			Issues:  issues,
			Stats: LinterStats{
				FilesScanned: len(files),
				TotalIssues:  len(issues),
			},
		}

		result.LinterResults = append(result.LinterResults, linterResult)
	}

	return nil
}

// parseFlake8Output parses flake8 output
func (sa *StaticAnalyzer) parseFlake8Output(output string) []ReviewIssue {
	var issues []ReviewIssue

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse flake8 output format: file:line:col: code message
		parts := strings.SplitN(line, ":", 4)
		if len(parts) >= 4 {
			issue := ReviewIssue{
				ID:          generateID(),
				Type:        IssueTypeStyle,
				Severity:    IssueSeverityLow,
				Title:       "Style issue",
				Description: strings.TrimSpace(parts[3]),
				File:        parts[0],
				Line:        parseInt(parts[1]),
				Column:      parseInt(parts[2]),
				Rule:        "flake8",
				Category:    "style",
				Fixable:     true,
				Tags:        []string{"python", "flake8"},
			}
			issues = append(issues, issue)
		}
	}

	return issues
}

// runMypy runs mypy type checker on Python files
func (sa *StaticAnalyzer) runMypy(ctx context.Context, files []string, result *StaticAnalysisResult) error {
	cmd := exec.CommandContext(ctx, "mypy")
	cmd.Args = append(cmd.Args, files...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		issues := sa.parseMypyOutput(string(output))

		typeCheckResult := TypeCheckResult{
			Tool:   "mypy",
			Issues: issues,
		}

		result.TypeCheckResults = append(result.TypeCheckResults, typeCheckResult)
	}

	return nil
}

// parseMypyOutput parses mypy output
func (sa *StaticAnalyzer) parseMypyOutput(output string) []ReviewIssue {
	var issues []ReviewIssue

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse mypy output format
		if strings.Contains(line, ".py:") {
			issue := ReviewIssue{
				ID:          generateID(),
				Type:        IssueTypeSyntax,
				Severity:    IssueSeverityMedium,
				Title:       "Type error",
				Description: line,
				Rule:        "mypy",
				Category:    "type-check",
				Fixable:     false,
				Tags:        []string{"python", "mypy", "types"},
			}
			issues = append(issues, issue)
		}
	}

	return issues
}

// analyzeJavaFiles performs Java-specific analysis
func (sa *StaticAnalyzer) analyzeJavaFiles(ctx context.Context, files []string, result *StaticAnalysisResult) error {
	// Run checkstyle if available
	if sa.isToolAvailable("checkstyle") {
		if err := sa.runCheckstyle(ctx, files, result); err != nil {
			fmt.Printf("Warning: checkstyle failed: %v\n", err)
		}
	}

	// Run SpotBugs if available
	if sa.isToolAvailable("spotbugs") {
		if err := sa.runSpotBugs(ctx, files, result); err != nil {
			fmt.Printf("Warning: SpotBugs failed: %v\n", err)
		}
	}

	return nil
}

// runCheckstyle runs Checkstyle on Java files
func (sa *StaticAnalyzer) runCheckstyle(ctx context.Context, files []string, result *StaticAnalysisResult) error {
	// Placeholder implementation
	return nil
}

// runSpotBugs runs SpotBugs on Java files
func (sa *StaticAnalyzer) runSpotBugs(ctx context.Context, files []string, result *StaticAnalysisResult) error {
	// Placeholder implementation
	return nil
}

// analyzeRustFiles performs Rust-specific analysis
func (sa *StaticAnalyzer) analyzeRustFiles(ctx context.Context, files []string, result *StaticAnalysisResult) error {
	// Run cargo clippy if available
	if sa.isToolAvailable("cargo") {
		if err := sa.runCargoClippy(ctx, files, result); err != nil {
			fmt.Printf("Warning: cargo clippy failed: %v\n", err)
		}
	}

	return nil
}

// runCargoClippy runs cargo clippy on Rust files
func (sa *StaticAnalyzer) runCargoClippy(ctx context.Context, files []string, result *StaticAnalysisResult) error {
	if len(files) == 0 {
		return nil
	}

	packageDir := filepath.Dir(files[0])

	cmd := exec.CommandContext(ctx, "cargo", "clippy", "--", "-D", "warnings")
	cmd.Dir = packageDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		issues := sa.parseCargoClippyOutput(string(output))

		linterResult := LinterResult{
			Tool:    "cargo clippy",
			Version: "unknown",
			Issues:  issues,
			Stats: LinterStats{
				FilesScanned: len(files),
				TotalIssues:  len(issues),
			},
		}

		result.LinterResults = append(result.LinterResults, linterResult)
	}

	return nil
}

// parseCargoClippyOutput parses cargo clippy output
func (sa *StaticAnalyzer) parseCargoClippyOutput(output string) []ReviewIssue {
	var issues []ReviewIssue

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse cargo clippy output
		if strings.Contains(line, ".rs:") && strings.Contains(line, "warning:") {
			issue := ReviewIssue{
				ID:          generateID(),
				Type:        IssueTypeStyle,
				Severity:    IssueSeverityLow,
				Title:       "Clippy warning",
				Description: line,
				Rule:        "clippy",
				Category:    "style",
				Fixable:     true,
				Tags:        []string{"rust", "clippy"},
			}
			issues = append(issues, issue)
		}
	}

	return issues
}

// runCustomRules runs custom static analysis rules
func (sa *StaticAnalyzer) runCustomRules(ctx context.Context, files []string, result *StaticAnalysisResult) error {
	for _, rule := range sa.config.CustomRules {
		issues := sa.applyCustomRule(rule, files)
		if len(issues) > 0 {
			linterResult := LinterResult{
				Tool:    "custom-rules",
				Version: "1.0.0",
				Issues:  issues,
				Stats: LinterStats{
					FilesScanned: len(files),
					TotalIssues:  len(issues),
				},
			}
			result.LinterResults = append(result.LinterResults, linterResult)
		}
	}

	return nil
}

// applyCustomRule applies a custom rule to the files
func (sa *StaticAnalyzer) applyCustomRule(rule CustomRule, files []string) []ReviewIssue {
	var issues []ReviewIssue

	pattern, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return issues // Skip invalid patterns
	}

	for _, file := range files {
		if rule.Language != "" && sa.detectLanguage(file) != rule.Language {
			continue // Skip files that don't match the language
		}

		// #nosec G304 - file comes from analysis filepath validation, not user input
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		matches := pattern.FindAllStringIndex(string(content), -1)
		for _, match := range matches {
			// Calculate line number
			lineNum := strings.Count(string(content[:match[0]]), "\n") + 1

			issue := ReviewIssue{
				ID:          generateID(),
				Type:        IssueTypeStyle,
				Severity:    sa.mapSeverity(rule.Severity),
				Title:       rule.Name,
				Description: rule.Message,
				File:        file,
				Line:        lineNum,
				Rule:        rule.ID,
				Category:    "custom",
				Fixable:     false,
				Tags:        []string{"custom", rule.Language},
			}
			issues = append(issues, issue)
		}
	}

	return issues
}

// mapSeverity maps string severity to IssueSeverity
func (sa *StaticAnalyzer) mapSeverity(severity string) IssueSeverity {
	switch strings.ToLower(severity) {
	case "critical":
		return IssueSeverityCritical
	case "high":
		return IssueSeverityHigh
	case "medium":
		return IssueSeverityMedium
	case "low":
		return IssueSeverityLow
	default:
		return IssueSeverityInfo
	}
}

// Utility functions

// isToolAvailable checks if a tool is available in PATH
func (sa *StaticAnalyzer) isToolAvailable(tool string) bool {
	_, err := exec.LookPath(tool)
	return err == nil
}

// getGoVersion gets the Go version
func (sa *StaticAnalyzer) getGoVersion() string {
	cmd := exec.Command("go", "version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

// parseInt safely parses a string to int
func parseInt(s string) int {
	var result int
	for _, char := range s {
		if char >= '0' && char <= '9' {
			result = result*10 + int(char-'0')
		} else {
			break
		}
	}
	return result
}
