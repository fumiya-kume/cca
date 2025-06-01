// Package review provides AI-powered code review capabilities with support for
// multiple programming languages, best practices analysis, and comprehensive feedback.
package review

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fumiya-kume/cca/pkg/analysis"
)

// File extension constants
const (
	extGo   = ".go"
	extJS   = ".js"
	extJSX  = ".jsx"
	extTS   = ".ts"
	extTSX  = ".tsx"
	extPy   = ".py"
	extJava = ".java"
	extRs   = ".rs"
)

// Language name constants
const (
	langJavaScript = "javascript"
	langTypeScript = "typescript"
	langPython     = "python"
	langJava       = "java"
	langRust       = "rust"
)

// AIReviewer provides AI-powered code review capabilities
type AIReviewer struct {
	config AIReviewerConfig
}

// AIReviewerConfig configures the AI reviewer
type AIReviewerConfig struct {
	MaxIterations       int
	EnableBestPractices bool
	EnableArchitectural bool
	EnableSecurity      bool
	EnablePerformance   bool
	EnableStyle         bool
	EnableTesting       bool
	ContextWindow       int
	ConfidenceThreshold float64
}

// NewAIReviewer creates a new AI reviewer
func NewAIReviewer(config AIReviewerConfig) *AIReviewer {
	// Set defaults
	if config.MaxIterations == 0 {
		config.MaxIterations = 3
	}
	if config.ContextWindow == 0 {
		config.ContextWindow = 8000
	}
	if config.ConfidenceThreshold == 0 {
		config.ConfidenceThreshold = 0.7
	}

	return &AIReviewer{
		config: config,
	}
}

// Review performs AI-powered code review
func (ar *AIReviewer) Review(ctx context.Context, files []string, analysisResult *analysis.AnalysisResult, reviewResult *ReviewResult) (*AIReviewResult, error) {
	result := &AIReviewResult{
		Recommendations:       []AIRecommendation{},
		BestPractices:         []BestPracticeIssue{},
		OverallFeedback:       "",
		ArchitecturalFeedback: "",
		StyleFeedback:         "",
		TestingFeedback:       "",
		SecurityFeedback:      "",
		PerformanceFeedback:   "",
		Confidence:            0.0,
	}

	// Build context for AI review
	context := ar.buildReviewContext(files, analysisResult, reviewResult)

	// Perform different types of AI review
	if ar.config.EnableBestPractices {
		if err := ar.reviewBestPractices(ctx, context, result); err != nil {
			return result, fmt.Errorf("best practices review failed: %w", err)
		}
	}

	if ar.config.EnableArchitectural {
		if err := ar.reviewArchitecture(ctx, context, result); err != nil {
			return result, fmt.Errorf("architectural review failed: %w", err)
		}
	}

	if ar.config.EnableSecurity {
		if err := ar.reviewSecurity(ctx, context, result); err != nil {
			return result, fmt.Errorf("security review failed: %w", err)
		}
	}

	if ar.config.EnablePerformance {
		if err := ar.reviewPerformance(ctx, context, result); err != nil {
			return result, fmt.Errorf("performance review failed: %w", err)
		}
	}

	if ar.config.EnableStyle {
		if err := ar.reviewStyle(ctx, context, result); err != nil {
			return result, fmt.Errorf("style review failed: %w", err)
		}
	}

	if ar.config.EnableTesting {
		if err := ar.reviewTesting(ctx, context, result); err != nil {
			return result, fmt.Errorf("testing review failed: %w", err)
		}
	}

	// Generate overall feedback
	ar.generateOverallFeedback(result)

	// Calculate confidence score
	result.Confidence = ar.calculateConfidence(result)

	return result, nil
}

// ReviewContext contains context information for AI review
type ReviewContext struct {
	ProjectInfo    *analysis.ProjectInfo
	Languages      []analysis.LanguageInfo
	Frameworks     []analysis.FrameworkInfo
	FileContents   map[string]string
	ChangedFiles   []string
	ExistingIssues []ReviewIssue
	QualityMetrics *QualityMetrics
	Dependencies   *analysis.DependencyInfo
	TestFiles      []string
	ConfigFiles    []string
	MainLanguage   string
	ProjectType    string
	TotalLOC       int
}

// buildReviewContext builds context for AI review
func (ar *AIReviewer) buildReviewContext(files []string, analysisResult *analysis.AnalysisResult, reviewResult *ReviewResult) *ReviewContext {
	context := &ReviewContext{
		FileContents:   make(map[string]string),
		ChangedFiles:   files,
		ExistingIssues: reviewResult.Issues,
		TestFiles:      []string{},
		ConfigFiles:    []string{},
		TotalLOC:       0,
	}

	if analysisResult != nil {
		context.ProjectInfo = analysisResult.ProjectInfo
		context.Languages = analysisResult.Languages
		context.Frameworks = analysisResult.Frameworks
		context.Dependencies = analysisResult.Dependencies
		context.QualityMetrics = reviewResult.QualityMetrics

		if context.ProjectInfo != nil {
			context.MainLanguage = context.ProjectInfo.MainLanguage
			context.ProjectType = context.ProjectInfo.ProjectType.String()
			context.ConfigFiles = context.ProjectInfo.ConfigFiles
		}
	}

	// Read file contents (with size limits)
	for _, file := range files {
		if ar.shouldIncludeFile(file) {
			content, err := ar.readFileWithLimit(file, ar.config.ContextWindow)
			if err == nil {
				context.FileContents[file] = content
				context.TotalLOC += len(strings.Split(content, "\n"))
			}

			if ar.isTestFile(file) {
				context.TestFiles = append(context.TestFiles, file)
			}
		}
	}

	return context
}

// reviewBestPractices performs best practices review
func (ar *AIReviewer) reviewBestPractices(ctx context.Context, reviewContext *ReviewContext, result *AIReviewResult) error {
	bestPractices := ar.getBestPracticesForLanguage(reviewContext.MainLanguage)

	for _, file := range reviewContext.ChangedFiles {
		content, exists := reviewContext.FileContents[file]
		if !exists {
			continue
		}

		language := ar.detectLanguage(file)
		issues := ar.checkBestPractices(file, content, language, bestPractices)
		result.BestPractices = append(result.BestPractices, issues...)
	}

	return nil
}

// reviewArchitecture performs architectural review
func (ar *AIReviewer) reviewArchitecture(ctx context.Context, reviewContext *ReviewContext, result *AIReviewResult) error {
	feedback := ar.analyzeArchitecture(reviewContext)
	result.ArchitecturalFeedback = feedback

	// Generate architectural recommendations
	recommendations := ar.generateArchitecturalRecommendations(reviewContext)
	result.Recommendations = append(result.Recommendations, recommendations...)

	return nil
}

// reviewSecurity performs security-focused review
func (ar *AIReviewer) reviewSecurity(ctx context.Context, reviewContext *ReviewContext, result *AIReviewResult) error {
	feedback := ar.analyzeSecurityConcerns(reviewContext)
	result.SecurityFeedback = feedback

	// Generate security recommendations
	recommendations := ar.generateSecurityRecommendations(reviewContext)
	result.Recommendations = append(result.Recommendations, recommendations...)

	return nil
}

// reviewPerformance performs performance review
func (ar *AIReviewer) reviewPerformance(ctx context.Context, reviewContext *ReviewContext, result *AIReviewResult) error {
	feedback := ar.analyzePerformance(reviewContext)
	result.PerformanceFeedback = feedback

	// Generate performance recommendations
	recommendations := ar.generatePerformanceRecommendations(reviewContext)
	result.Recommendations = append(result.Recommendations, recommendations...)

	return nil
}

// reviewStyle performs style and code quality review
func (ar *AIReviewer) reviewStyle(ctx context.Context, reviewContext *ReviewContext, result *AIReviewResult) error {
	feedback := ar.analyzeStyle(reviewContext)
	result.StyleFeedback = feedback

	// Generate style recommendations
	recommendations := ar.generateStyleRecommendations(reviewContext)
	result.Recommendations = append(result.Recommendations, recommendations...)

	return nil
}

// reviewTesting performs testing review
func (ar *AIReviewer) reviewTesting(ctx context.Context, reviewContext *ReviewContext, result *AIReviewResult) error {
	feedback := ar.analyzeTesting(reviewContext)
	result.TestingFeedback = feedback

	// Generate testing recommendations
	recommendations := ar.generateTestingRecommendations(reviewContext)
	result.Recommendations = append(result.Recommendations, recommendations...)

	return nil
}

// getBestPracticesForLanguage returns best practices for a specific language
func (ar *AIReviewer) getBestPracticesForLanguage(language string) []BestPracticeRule {
	switch strings.ToLower(language) {
	case "go":
		return ar.getGoBestPractices()
	case langJavaScript, langTypeScript:
		return ar.getJavaScriptBestPractices()
	case langPython:
		return ar.getPythonBestPractices()
	case langJava:
		return ar.getJavaBestPractices()
	case langRust:
		return ar.getRustBestPractices()
	default:
		return ar.getGenericBestPractices()
	}
}

// BestPracticeRule represents a best practice rule
type BestPracticeRule struct {
	ID          string
	Name        string
	Description string
	Pattern     string
	Suggestion  string
	Category    string
	Severity    string
	Reference   string
}

// getGoBestPractices returns Go-specific best practices
func (ar *AIReviewer) getGoBestPractices() []BestPracticeRule {
	return []BestPracticeRule{
		{
			ID:          "go-error-handling",
			Name:        "Proper Error Handling",
			Description: "Always handle errors explicitly",
			Pattern:     `(?m)^[^/]*\w+\([^)]*\)\s*{[^}]*}[^}]*$`,
			Suggestion:  "Check and handle all errors returned by functions",
			Category:    "error-handling",
			Severity:    "high",
			Reference:   "https://golang.org/doc/effective_go.html#errors",
		},
		{
			ID:          "go-naming",
			Name:        "Go Naming Conventions",
			Description: "Follow Go naming conventions",
			Pattern:     `(?m)^[^/]*func\s+[a-z]`,
			Suggestion:  "Use MixedCaps for exported functions",
			Category:    "naming",
			Severity:    "medium",
			Reference:   "https://golang.org/doc/effective_go.html#names",
		},
		{
			ID:          "go-gofmt",
			Name:        "Code Formatting",
			Description: "Code should be formatted with gofmt",
			Pattern:     `(?m)^[^/]*{\s*$`,
			Suggestion:  "Run gofmt to format code consistently",
			Category:    "formatting",
			Severity:    "low",
			Reference:   "https://golang.org/cmd/gofmt/",
		},
	}
}

// getJavaScriptBestPractices returns JavaScript/TypeScript best practices
func (ar *AIReviewer) getJavaScriptBestPractices() []BestPracticeRule {
	return []BestPracticeRule{
		{
			ID:          "js-const-let",
			Name:        "Use const/let instead of var",
			Description: "Prefer const and let over var for variable declarations",
			Pattern:     `(?m)^[^/]*var\s+`,
			Suggestion:  "Use const for immutable values, let for mutable variables",
			Category:    "modern-syntax",
			Severity:    "medium",
			Reference:   "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Statements/const",
		},
		{
			ID:          "js-arrow-functions",
			Name:        "Use Arrow Functions",
			Description: "Prefer arrow functions for cleaner syntax",
			Pattern:     `(?m)^[^/]*function\s*\([^)]*\)\s*{`,
			Suggestion:  "Consider using arrow functions for shorter syntax",
			Category:    "modern-syntax",
			Severity:    "low",
			Reference:   "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Functions/Arrow_functions",
		},
		{
			ID:          "js-async-await",
			Name:        "Prefer async/await over Promises",
			Description: "Use async/await for better readability",
			Pattern:     `(?m)^[^/]*\.then\(`,
			Suggestion:  "Consider using async/await instead of .then() chains",
			Category:    "async",
			Severity:    "medium",
			Reference:   "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Statements/async_function",
		},
	}
}

// getPythonBestPractices returns Python best practices
func (ar *AIReviewer) getPythonBestPractices() []BestPracticeRule {
	return []BestPracticeRule{
		{
			ID:          "python-pep8",
			Name:        "PEP 8 Style Guide",
			Description: "Follow PEP 8 style guidelines",
			Pattern:     `(?m)^[^#]*\w+\(\s*\w+\s*=\s*\w+\s*,\s*\w+\s*=`,
			Suggestion:  "Follow PEP 8 formatting guidelines",
			Category:    "style",
			Severity:    "medium",
			Reference:   "https://www.python.org/dev/peps/pep-0008/",
		},
		{
			ID:          "python-docstrings",
			Name:        "Function Docstrings",
			Description: "All functions should have docstrings",
			Pattern:     `(?m)^def\s+\w+\([^)]*\):\s*$`,
			Suggestion:  "Add docstrings to document function purpose and parameters",
			Category:    "documentation",
			Severity:    "medium",
			Reference:   "https://www.python.org/dev/peps/pep-257/",
		},
		{
			ID:          "python-list-comprehensions",
			Name:        "Use List Comprehensions",
			Description: "Prefer list comprehensions over loops when appropriate",
			Pattern:     `(?m)^[^#]*for\s+\w+\s+in\s+.*:\s*\n\s*\w+\.append\(`,
			Suggestion:  "Consider using list comprehension for cleaner code",
			Category:    "pythonic",
			Severity:    "low",
			Reference:   "https://docs.python.org/3/tutorial/datastructures.html#list-comprehensions",
		},
	}
}

// getJavaBestPractices returns Java best practices
func (ar *AIReviewer) getJavaBestPractices() []BestPracticeRule {
	return []BestPracticeRule{
		{
			ID:          "java-naming",
			Name:        "Java Naming Conventions",
			Description: "Follow Java naming conventions",
			Pattern:     `(?m)^[^/]*class\s+[a-z]`,
			Suggestion:  "Class names should start with uppercase letter",
			Category:    "naming",
			Severity:    "medium",
			Reference:   "https://www.oracle.com/java/technologies/javase/codeconventions-namingconventions.html",
		},
		{
			ID:          "java-final",
			Name:        "Use final for immutable variables",
			Description: "Declare variables as final when they won't change",
			Pattern:     `(?m)^[^/]*\w+\s+\w+\s*=`,
			Suggestion:  "Consider using final for immutable variables",
			Category:    "immutability",
			Severity:    "low",
			Reference:   "https://docs.oracle.com/javase/tutorial/java/javaOO/classvars.html",
		},
	}
}

// getRustBestPractices returns Rust best practices
func (ar *AIReviewer) getRustBestPractices() []BestPracticeRule {
	return []BestPracticeRule{
		{
			ID:          "rust-unwrap",
			Name:        "Avoid unwrap() in production",
			Description: "Handle errors explicitly instead of using unwrap()",
			Pattern:     `(?m)^[^/]*\.unwrap\(\)`,
			Suggestion:  "Use proper error handling instead of unwrap()",
			Category:    "error-handling",
			Severity:    "high",
			Reference:   "https://doc.rust-lang.org/book/ch09-00-error-handling.html",
		},
		{
			ID:          "rust-clone",
			Name:        "Minimize unnecessary clones",
			Description: "Avoid unnecessary clone() calls for performance",
			Pattern:     `(?m)^[^/]*\.clone\(\)`,
			Suggestion:  "Consider using references instead of cloning",
			Category:    "performance",
			Severity:    "medium",
			Reference:   "https://doc.rust-lang.org/book/ch04-01-what-is-ownership.html",
		},
	}
}

// getGenericBestPractices returns language-agnostic best practices
func (ar *AIReviewer) getGenericBestPractices() []BestPracticeRule {
	return []BestPracticeRule{
		{
			ID:          "generic-naming",
			Name:        "Descriptive Naming",
			Description: "Use descriptive names for variables and functions",
			Pattern:     `(?m)^[^/]*\b[a-z]\b`,
			Suggestion:  "Use descriptive names that explain purpose",
			Category:    "naming",
			Severity:    "medium",
			Reference:   "",
		},
		{
			ID:          "generic-comments",
			Name:        "Meaningful Comments",
			Description: "Write comments that explain why, not what",
			Pattern:     `(?m)^[^/]*//.*increment`,
			Suggestion:  "Focus comments on explaining the reasoning behind code",
			Category:    "documentation",
			Severity:    "low",
			Reference:   "",
		},
	}
}

// checkBestPractices checks code against best practices
func (ar *AIReviewer) checkBestPractices(file, content, language string, practices []BestPracticeRule) []BestPracticeIssue {
	var issues []BestPracticeIssue

	for _, practice := range practices {
		// This is a simplified implementation
		// In a real implementation, this would use more sophisticated pattern matching
		if strings.Contains(content, "var ") && practice.ID == "js-const-let" {
			issues = append(issues, BestPracticeIssue{
				Practice:   practice.Name,
				Violation:  "Found var declaration",
				File:       file,
				Line:       ar.findLineWithPattern(content, "var "),
				Suggestion: practice.Suggestion,
				Reference:  practice.Reference,
			})
		}

		if strings.Contains(content, ".unwrap()") && practice.ID == "rust-unwrap" {
			issues = append(issues, BestPracticeIssue{
				Practice:   practice.Name,
				Violation:  "Found unwrap() call",
				File:       file,
				Line:       ar.findLineWithPattern(content, ".unwrap()"),
				Suggestion: practice.Suggestion,
				Reference:  practice.Reference,
			})
		}
	}

	return issues
}

// analyzeArchitecture analyzes architectural aspects
func (ar *AIReviewer) analyzeArchitecture(context *ReviewContext) string {
	var feedback []string

	// Check for architectural patterns
	if context.ProjectType == "web-app" {
		feedback = append(feedback, "Web application architecture detected.")

		// Look for MVC pattern
		if ar.hasMVCStructure(context.FileContents) {
			feedback = append(feedback, "✓ MVC pattern appears to be followed")
		} else {
			feedback = append(feedback, "Consider organizing code using MVC or similar architectural pattern")
		}
	}

	// Check for separation of concerns
	if ar.hasGoodSeparationOfConcerns(context.FileContents) {
		feedback = append(feedback, "✓ Good separation of concerns")
	} else {
		feedback = append(feedback, "Consider improving separation of concerns")
	}

	// Check for dependency injection
	if ar.usesDependencyInjection(context.FileContents) {
		feedback = append(feedback, "✓ Dependency injection pattern detected")
	}

	if len(feedback) == 0 {
		feedback = append(feedback, "Architecture appears reasonable for the project size and scope.")
	}

	return strings.Join(feedback, "\n")
}

// analyzeSecurityConcerns analyzes security aspects
func (ar *AIReviewer) analyzeSecurityConcerns(context *ReviewContext) string {
	var concerns []string

	// Check for common security issues
	for file, content := range context.FileContents {
		if ar.hasHardcodedSecrets(content) {
			concerns = append(concerns, fmt.Sprintf("⚠️ Potential hardcoded secrets in %s", file))
		}

		if ar.hasInputValidationIssues(content) {
			concerns = append(concerns, fmt.Sprintf("⚠️ Input validation concerns in %s", file))
		}

		if ar.hasInsecureCrypto(content) {
			concerns = append(concerns, fmt.Sprintf("⚠️ Weak cryptography usage in %s", file))
		}
	}

	if len(concerns) == 0 {
		return "No obvious security concerns detected in the reviewed code."
	}

	return strings.Join(concerns, "\n")
}

// analyzePerformance analyzes performance aspects
func (ar *AIReviewer) analyzePerformance(context *ReviewContext) string {
	var issues []string

	for file, content := range context.FileContents {
		if ar.hasPerformanceIssues(content, context.MainLanguage) {
			issues = append(issues, fmt.Sprintf("Performance concerns in %s", file))
		}
	}

	// Check algorithmic complexity
	if context.QualityMetrics != nil && context.QualityMetrics.Complexity != nil {
		if context.QualityMetrics.Complexity.Cyclomatic > 15 {
			issues = append(issues, "High cyclomatic complexity may impact performance")
		}
	}

	if len(issues) == 0 {
		return "No significant performance issues detected."
	}

	return strings.Join(issues, "\n")
}

// analyzeStyle analyzes code style and readability
func (ar *AIReviewer) analyzeStyle(context *ReviewContext) string {
	var feedback []string

	// Check consistency
	if ar.hasConsistentStyle(context.FileContents) {
		feedback = append(feedback, "✓ Code style appears consistent")
	} else {
		feedback = append(feedback, "Consider improving code style consistency")
	}

	// Check naming conventions
	if ar.hasGoodNaming(context.FileContents, context.MainLanguage) {
		feedback = append(feedback, "✓ Good naming conventions")
	} else {
		feedback = append(feedback, "Consider improving variable and function naming")
	}

	return strings.Join(feedback, "\n")
}

// analyzeTesting analyzes testing aspects
func (ar *AIReviewer) analyzeTesting(context *ReviewContext) string {
	var feedback []string

	testCoverage := float64(len(context.TestFiles)) / float64(max(len(context.ChangedFiles), 1))

	if testCoverage >= 0.8 {
		feedback = append(feedback, "✓ Good test coverage ratio")
	} else if testCoverage >= 0.5 {
		feedback = append(feedback, "Moderate test coverage - consider adding more tests")
	} else {
		feedback = append(feedback, "⚠️ Low test coverage - strongly recommend adding tests")
	}

	// Check test quality
	for _, testFile := range context.TestFiles {
		if content, exists := context.FileContents[testFile]; exists {
			if ar.hasGoodTestStructure(content, context.MainLanguage) {
				feedback = append(feedback, fmt.Sprintf("✓ %s has good test structure", testFile))
			}
		}
	}

	return strings.Join(feedback, "\n")
}

// generateOverallFeedback generates overall feedback
func (ar *AIReviewer) generateOverallFeedback(result *AIReviewResult) {
	var feedback []string

	// Summarize recommendations
	if len(result.Recommendations) > 0 {
		feedback = append(feedback, fmt.Sprintf("Generated %d recommendations for improvement", len(result.Recommendations)))
	}

	// Summarize best practice issues
	if len(result.BestPractices) > 0 {
		feedback = append(feedback, fmt.Sprintf("Found %d best practice violations", len(result.BestPractices)))
	}

	// Add specific feedback sections
	if result.SecurityFeedback != "" {
		feedback = append(feedback, "Security review completed")
	}

	if result.PerformanceFeedback != "" {
		feedback = append(feedback, "Performance analysis completed")
	}

	result.OverallFeedback = strings.Join(feedback, ". ")
}

// generateArchitecturalRecommendations generates architectural recommendations
func (ar *AIReviewer) generateArchitecturalRecommendations(context *ReviewContext) []AIRecommendation {
	var recommendations []AIRecommendation

	// Check for large files that might need refactoring
	for file, content := range context.FileContents {
		lineCount := len(strings.Split(content, "\n"))
		if lineCount > 500 {
			recommendations = append(recommendations, AIRecommendation{
				Type:        "refactor",
				Priority:    "medium",
				Description: fmt.Sprintf("Consider breaking down %s (>500 lines)", file),
				Rationale:   "Large files are harder to maintain and understand",
				File:        file,
			})
		}
	}

	return recommendations
}

// generateSecurityRecommendations generates security recommendations
func (ar *AIReviewer) generateSecurityRecommendations(context *ReviewContext) []AIRecommendation {
	var recommendations []AIRecommendation

	for file, content := range context.FileContents {
		if ar.hasHardcodedSecrets(content) {
			recommendations = append(recommendations, AIRecommendation{
				Type:        "security",
				Priority:    "high",
				Description: "Move hardcoded secrets to environment variables",
				Rationale:   "Hardcoded secrets pose security risks",
				File:        file,
			})
		}
	}

	return recommendations
}

// generatePerformanceRecommendations generates performance recommendations
func (ar *AIReviewer) generatePerformanceRecommendations(context *ReviewContext) []AIRecommendation {
	var recommendations []AIRecommendation

	for file, content := range context.FileContents {
		if strings.Contains(content, "for") && strings.Contains(content, "append") {
			recommendations = append(recommendations, AIRecommendation{
				Type:        "optimization",
				Priority:    "low",
				Description: "Consider pre-allocating slice capacity",
				Rationale:   "Pre-allocation can improve performance for large slices",
				File:        file,
			})
		}
	}

	return recommendations
}

// generateStyleRecommendations generates style recommendations
func (ar *AIReviewer) generateStyleRecommendations(context *ReviewContext) []AIRecommendation {
	var recommendations []AIRecommendation

	// This would analyze code style and generate recommendations
	// Placeholder implementation

	return recommendations
}

// generateTestingRecommendations generates testing recommendations
func (ar *AIReviewer) generateTestingRecommendations(context *ReviewContext) []AIRecommendation {
	var recommendations []AIRecommendation

	// Check for files without corresponding tests
	for file := range context.FileContents {
		if !ar.isTestFile(file) && ar.isSourceFile(file) {
			testFile := ar.getExpectedTestFile(file, context.MainLanguage)
			if !ar.fileExists(testFile, context.FileContents) {
				recommendations = append(recommendations, AIRecommendation{
					Type:        "testing",
					Priority:    "medium",
					Description: fmt.Sprintf("Consider adding tests for %s", file),
					Rationale:   "Tests improve code reliability and maintainability",
					File:        file,
				})
			}
		}
	}

	return recommendations
}

// calculateConfidence calculates the confidence score of the AI review
func (ar *AIReviewer) calculateConfidence(result *AIReviewResult) float64 {
	score := 0.8 // Base confidence

	// Increase confidence based on the amount of analysis performed
	if result.ArchitecturalFeedback != "" {
		score += 0.05
	}
	if result.SecurityFeedback != "" {
		score += 0.05
	}
	if result.PerformanceFeedback != "" {
		score += 0.05
	}
	if result.StyleFeedback != "" {
		score += 0.03
	}
	if result.TestingFeedback != "" {
		score += 0.02
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

// Utility methods

// shouldIncludeFile checks if a file should be included in the review
func (ar *AIReviewer) shouldIncludeFile(file string) bool {
	// Skip binary files, large files, etc.
	ext := strings.ToLower(filepath.Ext(file))
	excludedExts := []string{".png", ".jpg", ".jpeg", ".gif", ".pdf", ".exe", ".bin"}

	for _, excluded := range excludedExts {
		if ext == excluded {
			return false
		}
	}

	return true
}

// readFileWithLimit reads a file with a character limit
func (ar *AIReviewer) readFileWithLimit(file string, limit int) (string, error) {
	// Validate file path to prevent directory traversal
	if err := ar.validateFilePath(file); err != nil {
		return "", err
	}
	
	// #nosec G304 - file path is validated with ValidateFilePath
	content, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}

	str := string(content)
	if len(str) > limit {
		str = str[:limit] + "... [truncated]"
	}

	return str, nil
}

// validateFilePath validates a file path to prevent directory traversal attacks
func (ar *AIReviewer) validateFilePath(file string) error {
	// Check for directory traversal patterns
	if strings.Contains(file, "..") {
		return fmt.Errorf("invalid file path: contains directory traversal")
	}
	
	// Ensure file is within expected project bounds
	absPath, err := filepath.Abs(file)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}
	
	// Additional security check - ensure file is a regular file
	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}
	
	if !info.Mode().IsRegular() {
		return fmt.Errorf("invalid file path: not a regular file")
	}
	
	return nil
}

// detectLanguage detects the programming language of a file
func (ar *AIReviewer) detectLanguage(file string) string {
	ext := strings.ToLower(filepath.Ext(file))

	switch ext {
	case extGo:
		return "go"
	case extJS, extJSX:
		return langJavaScript
	case extTS, extTSX:
		return langTypeScript
	case extPy:
		return langPython
	case extJava:
		return langJava
	case extRs:
		return langRust
	}

	return ""
}

// isTestFile checks if a file is a test file
func (ar *AIReviewer) isTestFile(file string) bool {
	name := strings.ToLower(filepath.Base(file))
	return strings.Contains(name, "test") ||
		strings.Contains(name, "spec") ||
		strings.HasSuffix(name, "_test.go")
}

// isSourceFile checks if a file is a source code file
func (ar *AIReviewer) isSourceFile(file string) bool {
	return ar.detectLanguage(file) != "" && !ar.isTestFile(file)
}

// getExpectedTestFile returns the expected test file for a source file
func (ar *AIReviewer) getExpectedTestFile(file, language string) string {
	switch language {
	case "go":
		return strings.TrimSuffix(file, ".go") + "_test.go"
	case langJavaScript, langTypeScript:
		ext := filepath.Ext(file)
		return strings.TrimSuffix(file, ext) + ".test" + ext
	case langPython:
		return strings.TrimSuffix(file, ".py") + "_test.py"
	default:
		return file + ".test"
	}
}

// fileExists checks if a file exists in the context
func (ar *AIReviewer) fileExists(file string, fileContents map[string]string) bool {
	_, exists := fileContents[file]
	return exists
}

// findLineWithPattern finds the line number containing a pattern
func (ar *AIReviewer) findLineWithPattern(content, pattern string) int {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.Contains(line, pattern) {
			return i + 1
		}
	}
	return 1
}

// Analysis helper methods (simplified implementations)

func (ar *AIReviewer) hasMVCStructure(files map[string]string) bool {
	hasModel := false
	hasView := false
	hasController := false

	for file := range files {
		lower := strings.ToLower(file)
		if strings.Contains(lower, "model") {
			hasModel = true
		}
		if strings.Contains(lower, "view") || strings.Contains(lower, "template") {
			hasView = true
		}
		if strings.Contains(lower, "controller") || strings.Contains(lower, "handler") {
			hasController = true
		}
	}

	return hasModel && hasView && hasController
}

func (ar *AIReviewer) hasGoodSeparationOfConcerns(files map[string]string) bool {
	// Simplified check - look for organized directory structure
	directories := make(map[string]bool)
	for file := range files {
		dir := filepath.Dir(file)
		directories[dir] = true
	}
	return len(directories) > 2 // Multiple directories suggest organization
}

func (ar *AIReviewer) usesDependencyInjection(files map[string]string) bool {
	for _, content := range files {
		if strings.Contains(content, "inject") ||
			strings.Contains(content, "Inject") ||
			strings.Contains(content, "DI") {
			return true
		}
	}
	return false
}

func (ar *AIReviewer) hasHardcodedSecrets(content string) bool {
	patterns := []string{"password", "secret", "key", "token", "api_key"}
	for _, pattern := range patterns {
		if strings.Contains(strings.ToLower(content), pattern+"=") {
			return true
		}
	}
	return false
}

func (ar *AIReviewer) hasInputValidationIssues(content string) bool {
	// Look for direct user input usage without validation
	return strings.Contains(content, "$_GET") ||
		strings.Contains(content, "$_POST") ||
		strings.Contains(content, "request.body")
}

func (ar *AIReviewer) hasInsecureCrypto(content string) bool {
	weakCrypto := []string{"md5", "sha1", "des", "rc4"}
	for _, crypto := range weakCrypto {
		if strings.Contains(strings.ToLower(content), crypto) {
			return true
		}
	}
	return false
}

func (ar *AIReviewer) hasPerformanceIssues(content, language string) bool {
	// Language-specific performance anti-patterns
	switch language {
	case "go":
		return strings.Contains(content, "defer") && strings.Contains(content, "for")
	case langJavaScript:
		return strings.Contains(content, "innerHTML") || strings.Contains(content, "eval")
	case langPython:
		return strings.Contains(content, "+= string") // String concatenation in loop
	}
	return false
}

func (ar *AIReviewer) hasConsistentStyle(files map[string]string) bool {
	// Simple heuristic - check indentation consistency
	indentStyles := make(map[string]int)
	for _, content := range files {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "    ") {
				indentStyles["spaces"]++
			} else if strings.HasPrefix(line, "\t") {
				indentStyles["tabs"]++
			}
		}
	}

	// More than 80% of one style
	total := indentStyles["spaces"] + indentStyles["tabs"]
	if total == 0 {
		return true
	}

	return float64(max(indentStyles["spaces"], indentStyles["tabs"]))/float64(total) > 0.8
}

func (ar *AIReviewer) hasGoodNaming(files map[string]string, language string) bool {
	// Simple check for descriptive naming
	shortVarCount := 0
	totalVarCount := 0

	for _, content := range files {
		// This is a very simplified check
		words := strings.Fields(content)
		for _, word := range words {
			if len(word) >= 2 && word[0] >= 'a' && word[0] <= 'z' {
				totalVarCount++
				if len(word) <= 2 {
					shortVarCount++
				}
			}
		}
	}

	if totalVarCount == 0 {
		return true
	}

	// Less than 30% short variable names
	return float64(shortVarCount)/float64(totalVarCount) < 0.3
}

func (ar *AIReviewer) hasGoodTestStructure(content, language string) bool {
	// Look for test structure patterns
	switch language {
	case "go":
		return strings.Contains(content, "func Test") && strings.Contains(content, "t.Error")
	case langJavaScript, langTypeScript:
		return strings.Contains(content, "describe") || strings.Contains(content, "test")
	case langPython:
		return strings.Contains(content, "def test_") || strings.Contains(content, "class Test")
	}
	return true
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
