package claude

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/fumiya-kume/cca/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDefaultPromptTemplates(t *testing.T) {
	templates := getDefaultPromptTemplates()

	assert.NotEmpty(t, templates)

	// Check that expected templates exist
	expectedTemplates := []string{
		"issue_analysis",
		"code_review",
	}

	for _, expected := range expectedTemplates {
		assert.Contains(t, templates, expected, "Expected template %s not found", expected)
	}
}

func TestPromptTemplate_IssueAnalysis(t *testing.T) {
	templates := getDefaultPromptTemplates()

	issueTemplate, exists := templates["issue_analysis"]
	require.True(t, exists, "issue_analysis template should exist")

	assert.Equal(t, "issue_analysis", issueTemplate.Name)
	assert.Equal(t, PromptIssueAnalysis, issueTemplate.Category)
	assert.NotEmpty(t, issueTemplate.Description)
	assert.NotEmpty(t, issueTemplate.Template)

	// Check that required variables are present
	expectedVars := []string{"title", "description", "labels", "comments", "code_context"}
	for _, expectedVar := range expectedVars {
		assert.Contains(t, issueTemplate.Variables, expectedVar)
	}

	// Check that template contains variable placeholders
	for _, variable := range expectedVars {
		placeholder := "{{." + variable + "}}"
		assert.Contains(t, issueTemplate.Template, placeholder,
			"Template should contain placeholder for %s", variable)
	}
}

func TestPromptTemplate_CodeReview(t *testing.T) {
	templates := getDefaultPromptTemplates()

	reviewTemplate, exists := templates["code_review"]
	require.True(t, exists, "code_review template should exist")

	assert.Equal(t, "code_review", reviewTemplate.Name)
	assert.Equal(t, PromptCodeReview, reviewTemplate.Category)
	assert.NotEmpty(t, reviewTemplate.Description)
	assert.NotEmpty(t, reviewTemplate.Template)

	// Check for code review specific content
	assert.Contains(t, reviewTemplate.Template, "review")
}

// Remove these tests since the templates don't exist in the actual implementation

func TestPromptCategory_Values(t *testing.T) {
	// Test that prompt categories are defined correctly
	templates := getDefaultPromptTemplates()

	categories := make(map[PromptCategory]bool)
	for _, template := range templates {
		categories[template.Category] = true
	}

	// Should have multiple different categories
	assert.True(t, len(categories) > 1, "Should have multiple prompt categories")

	// Check specific categories exist
	expectedCategories := []PromptCategory{
		PromptIssueAnalysis,
		PromptCodeReview,
	}

	for _, expected := range expectedCategories {
		assert.True(t, categories[expected], "Category %v should exist", expected)
	}
}

func TestPromptTemplate_Validation(t *testing.T) {
	templates := getDefaultPromptTemplates()

	for name, template := range templates {
		t.Run(name, func(t *testing.T) {
			// Each template should have required fields
			assert.NotEmpty(t, template.Name, "Template name should not be empty")
			assert.NotEmpty(t, template.Description, "Template description should not be empty")
			assert.NotEmpty(t, template.Template, "Template content should not be empty")

			// Name should match the key
			assert.Equal(t, name, template.Name, "Template name should match map key")

			// Template should contain at least one variable placeholder
			if len(template.Variables) > 0 {
				hasPlaceholder := false
				for _, variable := range template.Variables {
					placeholder := "{{." + variable + "}}"
					if strings.Contains(template.Template, placeholder) {
						hasPlaceholder = true
						break
					}
				}
				assert.True(t, hasPlaceholder, "Template should contain at least one variable placeholder")
			}
		})
	}
}

func TestPromptTemplate_VariableConsistency(t *testing.T) {
	templates := getDefaultPromptTemplates()

	for name, template := range templates {
		t.Run(name, func(t *testing.T) {
			// Check that all declared variables are actually used in the template
			for _, variable := range template.Variables {
				placeholder := "{{." + variable + "}}"
				assert.Contains(t, template.Template, placeholder,
					"Template %s should use declared variable %s", name, variable)
			}

			// Check that template doesn't use undeclared variables
			// This is a simple check for {{.variable}} patterns
			placeholderPattern := `\{\{\.[a-zA-Z_][a-zA-Z0-9_]*\}\}`
			matches := findMatches(template.Template, placeholderPattern)

			for _, match := range matches {
				// Extract variable name from {{.variable}}
				varName := strings.TrimPrefix(strings.TrimSuffix(match, "}}"), "{{.")
				assert.Contains(t, template.Variables, varName,
					"Template %s uses undeclared variable %s", name, varName)
			}
		})
	}
}

func TestPromptTemplate_JSONStructure(t *testing.T) {
	templates := getDefaultPromptTemplates()

	// Templates that should return JSON should mention JSON formatting
	jsonTemplates := []string{"issue_analysis"}

	for _, templateName := range jsonTemplates {
		template, exists := templates[templateName]
		require.True(t, exists)

		// Should mention JSON in the template
		assert.Contains(t, strings.ToLower(template.Template), "json",
			"Template %s should mention JSON formatting", templateName)
	}
}

func TestPromptTemplate_StructuredOutput(t *testing.T) {
	templates := getDefaultPromptTemplates()

	issueTemplate := templates["issue_analysis"]

	// Should request structured fields
	expectedFields := []string{
		"summary",
		"requirements",
		"technical_details",
		"complexity",
		"estimated_effort",
	}

	for _, field := range expectedFields {
		assert.Contains(t, issueTemplate.Template, field,
			"Issue analysis template should request %s field", field)
	}
}

func TestPromptTemplate_IssueDataIntegration(t *testing.T) {
	// Test that templates work with actual issue data structures
	issueData := &types.IssueData{
		Number:     123,
		Title:      "Add user authentication",
		Body:       "Implement JWT-based authentication system",
		State:      "open",
		Labels:     []string{"feature", "backend"},
		Repository: "test/repo",
		Comments:   []types.IssueComment{},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	templates := getDefaultPromptTemplates()
	issueTemplate := templates["issue_analysis"]

	// Template should be compatible with issue data structure
	assert.Contains(t, issueTemplate.Variables, "title")
	assert.Contains(t, issueTemplate.Variables, "description")
	assert.Contains(t, issueTemplate.Variables, "labels")

	// Verify we can extract the needed data from IssueData
	assert.Equal(t, "Add user authentication", issueData.Title)
	assert.Equal(t, "Implement JWT-based authentication system", issueData.Body)
	assert.Contains(t, issueData.Labels, "feature")
}

func TestPromptTemplate_ComplexityLevels(t *testing.T) {
	templates := getDefaultPromptTemplates()
	issueTemplate := templates["issue_analysis"]

	// Should define expected complexity levels
	complexityLevels := []string{"low", "medium", "high", "critical"}

	for _, level := range complexityLevels {
		assert.Contains(t, issueTemplate.Template, level,
			"Template should mention complexity level: %s", level)
	}
}

func TestPromptTemplate_TimeEstimation(t *testing.T) {
	templates := getDefaultPromptTemplates()
	issueTemplate := templates["issue_analysis"]

	// Should request time estimation
	timeKeywords := []string{"effort", "hours", "estimated"}

	foundTimeKeyword := false
	for _, keyword := range timeKeywords {
		if strings.Contains(strings.ToLower(issueTemplate.Template), keyword) {
			foundTimeKeyword = true
			break
		}
	}

	assert.True(t, foundTimeKeyword, "Template should request time estimation")
}

func TestPromptTemplate_Categories(t *testing.T) {
	// Test that all prompt categories are covered
	templates := getDefaultPromptTemplates()

	categoryCount := make(map[PromptCategory]int)
	for _, template := range templates {
		categoryCount[template.Category]++
	}

	// Should have at least one template per major category
	assert.True(t, categoryCount[PromptIssueAnalysis] > 0)
	assert.True(t, categoryCount[PromptCodeReview] > 0)
}

func TestPromptTemplate_ContentQuality(t *testing.T) {
	templates := getDefaultPromptTemplates()

	for name, template := range templates {
		t.Run(name, func(t *testing.T) {
			// Template should be substantial (not just a placeholder)
			assert.True(t, len(template.Template) > 100,
				"Template %s should have substantial content", name)

			// Should have clear instructions
			instructionKeywords := []string{"analyze", "provide", "include", "format", "please"}
			hasInstructions := false
			for _, keyword := range instructionKeywords {
				if strings.Contains(strings.ToLower(template.Template), keyword) {
					hasInstructions = true
					break
				}
			}
			assert.True(t, hasInstructions,
				"Template %s should contain clear instructions", name)

			// Should not have obvious placeholder text
			badPhrases := []string{"TODO", "FIXME", "placeholder", "example text"}
			for _, phrase := range badPhrases {
				assert.NotContains(t, strings.ToLower(template.Template), strings.ToLower(phrase),
					"Template %s should not contain placeholder text: %s", name, phrase)
			}
		})
	}
}

// Helper function to find regex matches
func findMatches(text, pattern string) []string {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}
	return re.FindAllString(text, -1)
}

func TestPromptTemplate_Documentation(t *testing.T) {
	templates := getDefaultPromptTemplates()

	for name, template := range templates {
		t.Run(name, func(t *testing.T) {
			// Description should be meaningful
			assert.True(t, len(template.Description) > 10,
				"Template %s should have meaningful description", name)

			// Description should not just repeat the name
			assert.NotEqual(t, strings.ToLower(template.Name), strings.ToLower(template.Description),
				"Template %s description should not just repeat the name", name)
		})
	}
}

func TestPromptTemplate_VariableNaming(t *testing.T) {
	templates := getDefaultPromptTemplates()

	for name, template := range templates {
		t.Run(name, func(t *testing.T) {
			for _, variable := range template.Variables {
				// Variable names should be valid identifiers
				assert.Regexp(t, `^[a-zA-Z][a-zA-Z0-9_]*$`, variable,
					"Variable %s in template %s should be a valid identifier", variable, name)

				// Should not be too short (avoid single letters)
				assert.True(t, len(variable) > 1,
					"Variable %s in template %s should be descriptive", variable, name)
			}
		})
	}
}
