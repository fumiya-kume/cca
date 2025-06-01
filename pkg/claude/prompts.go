package claude

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fumiya-kume/cca/internal/types"
)

// getDefaultPromptTemplates returns the built-in prompt templates
func getDefaultPromptTemplates() map[string]PromptTemplate {
	return map[string]PromptTemplate{
		"issue_analysis": {
			Name:        "issue_analysis",
			Category:    PromptIssueAnalysis,
			Description: "Analyzes GitHub issues for development planning",
			Variables:   []string{"title", "description", "labels", "comments", "code_context"},
			Template: `Please analyze this GitHub issue for development planning:

Title: {{.title}}

Description:
{{.description}}

Labels: {{.labels}}

Comments:
{{.comments}}

Code Context:
{{.code_context}}

Please provide a comprehensive analysis including:
1. Summary of requirements
2. Technical implementation details
3. Complexity assessment (low/medium/high/critical)
4. Estimated effort
5. Dependencies and prerequisites
6. Potential risks or challenges
7. Recommended approach

Format your response as structured JSON with the following fields:
- summary: string
- requirements: array of strings
- technical_details: array of strings
- complexity: string (low/medium/high/critical)
- estimated_effort_hours: number
- dependencies: array of strings
- risk_factors: array of strings
- recommendations: array of strings`,
		},

		"code_generation": {
			Name:        "code_generation",
			Category:    PromptCodeGeneration,
			Description: "Generates code based on requirements",
			Variables:   []string{"requirements", "context", "language", "framework", "constraints"},
			Template: `Please generate code based on these requirements:

Requirements:
{{.requirements}}

Programming Language: {{.language}}
Framework: {{.framework}}

Code Context:
{{.context}}

Constraints:
{{.constraints}}

Please generate:
1. Implementation code with proper structure
2. Unit tests
3. Documentation/comments
4. Installation/setup instructions

Format your response as structured JSON with the following fields:
- files: array of objects with {path, content, language, description}
- tests: array of objects with {path, content, language, description}
- documentation: string
- instructions: array of strings
- warnings: array of strings

Ensure code follows best practices, is well-commented, and includes error handling.`,
		},

		"code_review": {
			Name:        "code_review",
			Category:    PromptCodeReview,
			Description: "Performs comprehensive code review",
			Variables:   []string{"files", "context", "review_type"},
			Template: `Please perform a {{.review_type}} code review for these files:

Files to Review:
{{.files}}

Code Context:
{{.context}}

Please analyze for:
1. Security vulnerabilities
2. Performance issues
3. Code quality and maintainability
4. Best practices adherence
5. Potential bugs
6. Style and consistency

Format your response as structured JSON with the following fields:
- summary: string
- issues: array of objects with {file, line, type, severity, description, suggestion}
- suggestions: array of objects with {file, line, type, description, code}
- score: number (0-100)
- approved: boolean

Provide specific line numbers where possible and actionable suggestions for improvements.`,
		},

		"debugging": {
			Name:        "debugging",
			Category:    PromptDebugging,
			Description: "Helps debug code issues",
			Variables:   []string{"error", "code", "context", "steps_taken"},
			Template: `Please help debug this issue:

Error/Problem:
{{.error}}

Relevant Code:
{{.code}}

Context:
{{.context}}

Steps Already Taken:
{{.steps_taken}}

Please provide:
1. Root cause analysis
2. Step-by-step debugging approach
3. Potential solutions
4. Prevention strategies

Format your response with clear sections and actionable steps.`,
		},

		"test_generation": {
			Name:        "test_generation",
			Category:    PromptTesting,
			Description: "Generates comprehensive tests",
			Variables:   []string{"code", "test_type", "framework", "coverage_requirements"},
			Template: `Please generate comprehensive tests for this code:

Code to Test:
{{.code}}

Test Type: {{.test_type}}
Test Framework: {{.framework}}
Coverage Requirements: {{.coverage_requirements}}

Please generate:
1. Unit tests covering all functions/methods
2. Integration tests for component interactions
3. Edge case and error condition tests
4. Performance/load tests if applicable

Format your response as structured JSON with test files and descriptions.`,
		},

		"documentation": {
			Name:        "documentation",
			Category:    PromptDocumentation,
			Description: "Generates technical documentation",
			Variables:   []string{"code", "doc_type", "audience", "format"},
			Template: `Please generate {{.doc_type}} documentation for this code:

Code:
{{.code}}

Target Audience: {{.audience}}
Format: {{.format}}

Please include:
1. Overview and purpose
2. API/interface documentation
3. Usage examples
4. Configuration options
5. Troubleshooting guide

Make the documentation clear, comprehensive, and properly formatted.`,
		},

		"refactoring": {
			Name:        "refactoring",
			Category:    PromptRefactoring,
			Description: "Suggests code refactoring improvements",
			Variables:   []string{"code", "goals", "constraints", "patterns"},
			Template: `Please suggest refactoring improvements for this code:

Code:
{{.code}}

Refactoring Goals:
{{.goals}}

Constraints:
{{.constraints}}

Preferred Patterns:
{{.patterns}}

Please provide:
1. Analysis of current code structure
2. Specific refactoring recommendations
3. Before/after code examples
4. Benefits of each change
5. Migration strategy

Prioritize maintainability, readability, and performance.`,
		},
	}
}

// buildIssueAnalysisPrompt builds a prompt for issue analysis
func (s *Service) buildIssueAnalysisPrompt(issueData *types.IssueData, codeContext *types.CodeContext) (string, error) {
	template, exists := s.promptTemplates["issue_analysis"]
	if !exists {
		return "", fmt.Errorf("issue analysis template not found")
	}

	// Prepare template variables
	variables := map[string]string{
		"title":        issueData.Title,
		"description":  issueData.Description,
		"labels":       strings.Join(issueData.Labels, ", "),
		"comments":     formatComments(issueData.Comments),
		"code_context": formatCodeContext(codeContext),
	}

	return s.expandTemplate(template.Template, variables), nil
}

// buildCodeGenerationPrompt builds a prompt for code generation
func (s *Service) buildCodeGenerationPrompt(request *CodeGenerationRequest) (string, error) {
	template, exists := s.promptTemplates["code_generation"]
	if !exists {
		return "", fmt.Errorf("code generation template not found")
	}

	variables := map[string]string{
		"requirements": strings.Join(request.Requirements, "\n"),
		"context":      formatCodeContext(request.Context),
		"language":     request.Language,
		"framework":    request.Framework,
		"constraints":  strings.Join(request.Constraints, "\n"),
	}

	return s.expandTemplate(template.Template, variables), nil
}

// buildCodeReviewPrompt builds a prompt for code review
func (s *Service) buildCodeReviewPrompt(request *CodeReviewRequest) (string, error) {
	template, exists := s.promptTemplates["code_review"]
	if !exists {
		return "", fmt.Errorf("code review template not found")
	}

	variables := map[string]string{
		"files":       formatFileList(request.Files),
		"context":     formatCodeContext(request.Context),
		"review_type": formatReviewType(request.ReviewType),
	}

	return s.expandTemplate(template.Template, variables), nil
}

// expandTemplate expands a template with variables
func (s *Service) expandTemplate(template string, variables map[string]string) string {
	result := template
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// formatComments formats issue comments for prompt inclusion
func formatComments(comments []types.IssueComment) string {
	if len(comments) == 0 {
		return "No comments"
	}

	var formatted []string
	for i, comment := range comments {
		if i >= 5 { // Limit to first 5 comments
			formatted = append(formatted, fmt.Sprintf("... and %d more comments", len(comments)-5))
			break
		}
		formatted = append(formatted, fmt.Sprintf("@%s: %s", comment.Author, comment.Body))
	}

	return strings.Join(formatted, "\n\n")
}

// formatCodeContext formats code context for prompt inclusion
func formatCodeContext(context *types.CodeContext) string {
	if context == nil {
		return "No code context provided"
	}

	var parts []string

	// Add project structure (from project summary)
	if context.ProjectSummary != nil {
		parts = append(parts, "Project Structure:")
		parts = append(parts, fmt.Sprintf("- Project: %s", context.ProjectSummary.Name))
		parts = append(parts, fmt.Sprintf("- Type: %s", context.ProjectSummary.ProjectType))
		parts = append(parts, fmt.Sprintf("- Language: %s", context.ProjectSummary.MainLanguage))
		parts = append(parts, fmt.Sprintf("- Build System: %s", context.ProjectSummary.BuildSystem))
		parts = append(parts, "")
	}

	// Add relevant files
	if len(context.RelevantFiles) > 0 {
		parts = append(parts, "Relevant Files:")
		for _, file := range context.RelevantFiles {
			parts = append(parts, fmt.Sprintf("File: %s (%s)", file.Path, file.Purpose))
			if file.Summary != "" {
				parts = append(parts, fmt.Sprintf("Summary: %s", file.Summary))
			}
			parts = append(parts, "")
		}
	}

	// Add dependencies
	if context.Dependencies != nil && len(context.Dependencies.KeyDependencies) > 0 {
		parts = append(parts, "Dependencies:")
		for _, dep := range context.Dependencies.KeyDependencies {
			parts = append(parts, fmt.Sprintf("- %s", dep))
		}
		parts = append(parts, "")
	}

	// Add build configuration
	if context.BuildInformation != nil {
		parts = append(parts, "Build Configuration:")
		if context.BuildInformation.BuildSystem != "" {
			parts = append(parts, fmt.Sprintf("Build System: %s", context.BuildInformation.BuildSystem))
		}
		if len(context.BuildInformation.BuildTargets) > 0 {
			parts = append(parts, fmt.Sprintf("Build Targets: %v", context.BuildInformation.BuildTargets))
		}
		parts = append(parts, "")
	}

	if len(parts) == 0 {
		return "No detailed code context available"
	}

	return strings.Join(parts, "\n")
}

// formatFileList formats a list of files for review
func formatFileList(files []string) string {
	if len(files) == 0 {
		return "No files specified"
	}

	var formatted []string
	for _, file := range files {
		formatted = append(formatted, fmt.Sprintf("- %s", file))
	}

	return strings.Join(formatted, "\n")
}

// formatReviewType formats review type for prompts
func formatReviewType(reviewType ReviewType) string {
	switch reviewType {
	case ReviewSecurity:
		return "security-focused"
	case ReviewPerformance:
		return "performance-focused"
	case ReviewMaintainability:
		return "maintainability-focused"
	case ReviewFull:
		return "comprehensive"
	default:
		return "general"
	}
}

// parseIssueAnalysis parses Claude's response to issue analysis
func (s *Service) parseIssueAnalysis(response string) (*IssueAnalysis, error) {
	// Extract JSON from response
	jsonStr, err := extractJSON(response)
	if err != nil {
		return nil, fmt.Errorf("failed to extract JSON from response: %w", err)
	}

	// Parse the structured response
	analysis := &IssueAnalysis{}

	// Simple field extraction (in a real implementation, use proper JSON parsing)
	analysis.Summary = extractField(jsonStr, "summary")
	analysis.Requirements = extractArrayField(jsonStr, "requirements")
	analysis.TechnicalDetails = extractArrayField(jsonStr, "technical_details")
	analysis.Dependencies = extractArrayField(jsonStr, "dependencies")
	analysis.RiskFactors = extractArrayField(jsonStr, "risk_factors")
	analysis.Recommendations = extractArrayField(jsonStr, "recommendations")

	// Parse complexity
	complexityStr := extractField(jsonStr, "complexity")
	analysis.Complexity = parseComplexity(complexityStr)

	// Parse estimated effort
	effortStr := extractField(jsonStr, "estimated_effort_hours")
	if effortHours, err := strconv.ParseFloat(effortStr, 64); err == nil {
		analysis.EstimatedEffort = time.Duration(effortHours * float64(time.Hour))
	}

	return analysis, nil
}

// parseCodeGenerationResult parses Claude's response to code generation
func (s *Service) parseCodeGenerationResult(response string) (*CodeGenerationResult, error) {
	jsonStr, err := extractJSON(response)
	if err != nil {
		return nil, fmt.Errorf("failed to extract JSON from response: %w", err)
	}

	result := &CodeGenerationResult{
		Files:         []GeneratedFile{},
		Tests:         []GeneratedFile{},
		Documentation: extractField(jsonStr, "documentation"),
		Instructions:  extractArrayField(jsonStr, "instructions"),
		Warnings:      extractArrayField(jsonStr, "warnings"),
	}

	// Parse files (simplified - in reality would use proper JSON parsing)
	filesSection := extractSection(jsonStr, "files")
	result.Files = parseGeneratedFiles(filesSection)

	testsSection := extractSection(jsonStr, "tests")
	result.Tests = parseGeneratedFiles(testsSection)

	return result, nil
}

// parseCodeReviewResult parses Claude's response to code review
func (s *Service) parseCodeReviewResult(response string) (*CodeReviewResult, error) {
	jsonStr, err := extractJSON(response)
	if err != nil {
		return nil, fmt.Errorf("failed to extract JSON from response: %w", err)
	}

	result := &CodeReviewResult{
		Summary:     extractField(jsonStr, "summary"),
		Issues:      []ReviewIssue{},
		Suggestions: []ReviewSuggestion{},
	}

	// Parse score
	scoreStr := extractField(jsonStr, "score")
	if score, err := strconv.Atoi(scoreStr); err == nil {
		result.Score = score
	}

	// Parse approved
	approvedStr := extractField(jsonStr, "approved")
	result.Approved = approvedStr == "true"

	// Parse issues and suggestions (simplified)
	issuesSection := extractSection(jsonStr, "issues")
	result.Issues = parseReviewIssues(issuesSection)

	suggestionsSection := extractSection(jsonStr, "suggestions")
	result.Suggestions = parseReviewSuggestions(suggestionsSection)

	return result, nil
}

// Helper functions for parsing

func extractJSON(response string) (string, error) {
	// Look for JSON blocks in the response
	jsonRegex := regexp.MustCompile(`\{[\s\S]*\}`)
	matches := jsonRegex.FindString(response)
	if matches == "" {
		return "", fmt.Errorf("no JSON found in response")
	}
	return matches, nil
}

func extractField(jsonStr, field string) string {
	// Simple field extraction (in reality, use proper JSON parsing)
	pattern := fmt.Sprintf(`"%s":\s*"([^"]*)"`, field)
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(jsonStr)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func extractArrayField(jsonStr, field string) []string {
	// Simple array extraction (in reality, use proper JSON parsing)
	pattern := fmt.Sprintf(`"%s":\s*\[(.*?)\]`, field)
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(jsonStr)
	if len(matches) > 1 {
		arrayContent := matches[1]
		// Split by comma and clean up
		items := strings.Split(arrayContent, ",")
		var result []string
		for _, item := range items {
			cleaned := strings.Trim(strings.TrimSpace(item), `"`)
			if cleaned != "" {
				result = append(result, cleaned)
			}
		}
		return result
	}
	return []string{}
}

func extractSection(jsonStr, section string) string {
	// Extract a section from JSON (simplified)
	pattern := fmt.Sprintf(`"%s":\s*(\[.*?\])`, section)
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(jsonStr)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func parseComplexity(complexity string) ComplexityLevel {
	switch strings.ToLower(complexity) {
	case "low":
		return ComplexityLow
	case "medium":
		return ComplexityMedium
	case "high":
		return ComplexityHigh
	case "critical":
		return ComplexityCritical
	default:
		return ComplexityMedium
	}
}

func parseGeneratedFiles(filesSection string) []GeneratedFile {
	// Simplified parsing - in reality would use proper JSON parsing
	var files []GeneratedFile

	// This is a placeholder implementation
	// Real implementation would properly parse the JSON array
	if strings.Contains(filesSection, "path") {
		file := GeneratedFile{
			Path:        "example.go",
			Content:     "// Generated code",
			Language:    "go",
			Description: "Generated file",
		}
		files = append(files, file)
	}

	return files
}

func parseReviewIssues(issuesSection string) []ReviewIssue {
	// Simplified parsing - in reality would use proper JSON parsing
	var issues []ReviewIssue

	// Placeholder implementation
	if strings.Contains(issuesSection, "file") {
		issue := ReviewIssue{
			File:        "example.go",
			Line:        1,
			Type:        IssueBug,
			Severity:    SeverityWarning,
			Description: "Example issue",
			Suggestion:  "Fix the issue",
		}
		issues = append(issues, issue)
	}

	return issues
}

func parseReviewSuggestions(suggestionsSection string) []ReviewSuggestion {
	// Simplified parsing - in reality would use proper JSON parsing
	var suggestions []ReviewSuggestion

	// Placeholder implementation
	if strings.Contains(suggestionsSection, "file") {
		suggestion := ReviewSuggestion{
			File:        "example.go",
			Line:        1,
			Type:        SuggestionImprovement,
			Description: "Example suggestion",
			Code:        "// Improved code",
		}
		suggestions = append(suggestions, suggestion)
	}

	return suggestions
}
