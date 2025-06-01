package pr

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/fumiya-kume/cca/pkg/analysis"
	"github.com/fumiya-kume/cca/pkg/commit"
)

// PR type constants
const (
	prTypeFeature = "feature"
)

// TemplateGenerator generates pull request templates and content
type TemplateGenerator struct {
	config TemplateGeneratorConfig
}

// TemplateGeneratorConfig configures the template generator
type TemplateGeneratorConfig struct {
	DefaultTemplate   string
	CustomLabels      map[string]string
	IncludeChecklist  bool
	IncludeMetadata   bool
	IncludeReferences bool
	CustomSections    []TemplateSection
	VariableMapping   map[string]string
}

// TemplateContext contains context for template generation
type TemplateContext struct {
	Title             string
	ChangeType        string
	Priority          string
	Complexity        string
	EstimatedTime     string
	CommitCount       int
	FilesChanged      int
	LinesAdded        int
	LinesDeleted      int
	TestCoverage      float64
	IssueReferences   []string
	BreakingChanges   bool
	SecurityImpact    bool
	PerformanceImpact bool
	DatabaseChanges   bool
	ConfigChanges     bool
	Commits           []CommitSummary
	ChangedFiles      []FileSummary
	TestFiles         []string
	DocFiles          []string
	AnalysisResults   AnalysisSummary
	CustomVariables   map[string]string
}

// CommitSummary represents a summary of a commit
type CommitSummary struct {
	Type      string
	Scope     string
	Subject   string
	Breaking  bool
	Files     []string
	Additions int
	Deletions int
}

// FileSummary represents a summary of changed files
type FileSummary struct {
	Path      string
	Status    string
	Additions int
	Deletions int
	Language  string
	Type      string // source, test, config, doc
}

// AnalysisSummary contains summary of analysis results
type AnalysisSummary struct {
	Languages       []string
	Frameworks      []string
	MainLanguage    string
	ProjectType     string
	Dependencies    int
	Vulnerabilities int
	QualityScore    float64
	ComplexityScore float64
}

// NewTemplateGenerator creates a new template generator
func NewTemplateGenerator(config TemplateGeneratorConfig) *TemplateGenerator {
	return &TemplateGenerator{
		config: config,
	}
}

// GenerateTemplate generates a pull request template
func (tg *TemplateGenerator) GenerateTemplate(ctx context.Context, commitPlan *commit.CommitPlan, analysisResult *analysis.AnalysisResult, metadata *PRMetadata) (*PRTemplate, error) {
	// Build template context
	context := tg.buildTemplateContext(commitPlan, analysisResult, metadata)

	// Select appropriate template
	templateName := tg.selectTemplate(context)

	// Generate template sections
	sections := tg.generateSections(context, templateName)

	template := &PRTemplate{
		Name:        templateName,
		Title:       context.Title,
		Description: tg.generateTemplateDescription(context),
		Variables:   context.CustomVariables,
		Sections:    sections,
		Conditions:  tg.generateConditions(context),
	}

	return template, nil
}

// GenerateFromTemplate generates content from a template
func (tg *TemplateGenerator) GenerateFromTemplate(ctx context.Context, template *PRTemplate, context *TemplateContext) (string, error) {
	// Build the complete template content
	var templateContent strings.Builder

	// Add title
	templateContent.WriteString("# ")
	templateContent.WriteString(template.Title)
	templateContent.WriteString("\n\n")

	// Add sections in order
	for _, section := range template.Sections {
		if tg.shouldIncludeSection(section, context) {
			content, err := tg.renderSection(section, context)
			if err != nil {
				return "", fmt.Errorf("failed to render section %s: %w", section.Name, err)
			}
			templateContent.WriteString(content)
			templateContent.WriteString("\n\n")
		}
	}

	return templateContent.String(), nil
}

// buildTemplateContext builds the context for template generation
func (tg *TemplateGenerator) buildTemplateContext(commitPlan *commit.CommitPlan, analysisResult *analysis.AnalysisResult, metadata *PRMetadata) *TemplateContext {
	context := &TemplateContext{
		CustomVariables: make(map[string]string),
		TestFiles:       []string{},
		DocFiles:        []string{},
		ChangedFiles:    []FileSummary{},
		Commits:         []CommitSummary{},
	}

	// Extract basic information
	if metadata != nil {
		context.ChangeType = metadata.ChangeType.String()
		context.Priority = metadata.Priority.String()
		context.Complexity = metadata.Complexity.String()
		context.EstimatedTime = metadata.EstimatedReviewTime.String()
		context.TestCoverage = metadata.TestCoverage
		context.IssueReferences = metadata.IssueReferences
	}

	// Extract commit information
	if commitPlan != nil {
		context.CommitCount = len(commitPlan.Commits)
		context.FilesChanged = commitPlan.TotalChanges

		for _, commit := range commitPlan.Commits {
			summary := CommitSummary{
				Type:     commit.Type.String(),
				Scope:    commit.Scope,
				Subject:  commit.Message.Subject,
				Breaking: commit.Breaking,
				Files:    commit.Files,
			}

			// Calculate additions/deletions
			for _, change := range commit.Changes {
				summary.Additions += change.Additions
				summary.Deletions += change.Deletions
			}

			context.Commits = append(context.Commits, summary)
			context.LinesAdded += summary.Additions
			context.LinesDeleted += summary.Deletions

			// Check for breaking changes
			if commit.Breaking {
				context.BreakingChanges = true
			}

			// Categorize files
			for _, file := range commit.Files {
				fileSummary := tg.createFileSummary(file, commit.Changes)
				context.ChangedFiles = append(context.ChangedFiles, fileSummary)

				if tg.isTestFile(file) {
					context.TestFiles = append(context.TestFiles, file)
				} else if tg.isDocFile(file) {
					context.DocFiles = append(context.DocFiles, file)
				}

				// Check for special file types
				if tg.isConfigFile(file) {
					context.ConfigChanges = true
				}
				if tg.isDatabaseFile(file) {
					context.DatabaseChanges = true
				}
			}
		}
	}

	// Extract analysis information
	if analysisResult != nil {
		context.AnalysisResults = tg.buildAnalysisSummary(analysisResult)
	}

	// Generate title if not provided
	if context.Title == "" {
		context.Title = tg.generateTitle(context)
	}

	return context
}

// selectTemplate selects the appropriate template based on context
func (tg *TemplateGenerator) selectTemplate(context *TemplateContext) string {
	// Select template based on change type and other factors
	switch context.ChangeType {
	case prTypeFeature:
		if context.BreakingChanges {
			return "breaking-feature"
		}
		if context.Complexity == "complex" || context.Complexity == "very-complex" {
			return "complex-feature"
		}
		return prTypeFeature
	case prTypeBugfix:
		if context.Priority == "critical" || context.Priority == "urgent" {
			return "hotfix"
		}
		return "bugfix"
	case prTypeDocumentation:
		return "documentation"
	case prTypeTest:
		return "test-update"
	case prTypeRefactor:
		return "refactor"
	default:
		return "default"
	}
}

// generateSections generates template sections
func (tg *TemplateGenerator) generateSections(context *TemplateContext, templateName string) []TemplateSection {
	var sections []TemplateSection

	// Add common sections
	sections = append(sections, tg.createSummarySection(context))
	sections = append(sections, tg.createChangesSection(context))

	// Add template-specific sections
	switch templateName {
	case prTypeFeature, "complex-feature":
		sections = append(sections, tg.createFeatureSection(context))
		if context.BreakingChanges {
			sections = append(sections, tg.createBreakingChangesSection(context))
		}
	case "bugfix", "hotfix":
		sections = append(sections, tg.createBugfixSection(context))
	case prTypeDocumentation:
		sections = append(sections, tg.createDocumentationSection(context))
	case "test-update":
		sections = append(sections, tg.createTestSection(context))
	case prTypeRefactor:
		sections = append(sections, tg.createRefactorSection(context))
	}

	// Add testing section if tests are included
	if len(context.TestFiles) > 0 {
		sections = append(sections, tg.createTestingSection(context))
	}

	// Add checklist section
	if tg.config.IncludeChecklist {
		sections = append(sections, tg.createChecklistSection(context))
	}

	// Add metadata section
	if tg.config.IncludeMetadata {
		sections = append(sections, tg.createMetadataSection(context))
	}

	// Add custom sections
	sections = append(sections, tg.config.CustomSections...)

	// Sort sections by order
	for i := range sections {
		if sections[i].Order == 0 {
			sections[i].Order = i + 1
		}
	}

	return sections
}

// createSummarySection creates the summary section
func (tg *TemplateGenerator) createSummarySection(context *TemplateContext) TemplateSection {
	content := "## Summary\n\n"

	if len(context.Commits) == 1 {
		content += fmt.Sprintf("This PR %s.\n\n", context.Commits[0].Subject)
	} else {
		content += fmt.Sprintf("This PR contains %d commits that:\n\n", context.CommitCount)
		for _, commit := range context.Commits {
			content += fmt.Sprintf("- %s: %s\n", commit.Type, commit.Subject)
		}
		content += "\n"
	}

	// Add impact information
	if context.BreakingChanges {
		content += "⚠️ **Breaking Changes**: This PR contains breaking changes.\n\n"
	}

	if context.SecurityImpact {
		content += "🔒 **Security Impact**: This PR affects security-related code.\n\n"
	}

	if context.PerformanceImpact {
		content += "⚡ **Performance Impact**: This PR may affect performance.\n\n"
	}

	return TemplateSection{
		Name:     "summary",
		Title:    "Summary",
		Content:  content,
		Required: true,
		Order:    1,
	}
}

// createChangesSection creates the changes section
func (tg *TemplateGenerator) createChangesSection(context *TemplateContext) TemplateSection {
	content := "## Changes\n\n"
	content += fmt.Sprintf("- **Files changed**: %d\n", context.FilesChanged)
	content += fmt.Sprintf("- **Lines added**: %d\n", context.LinesAdded)
	content += fmt.Sprintf("- **Lines deleted**: %d\n", context.LinesDeleted)
	content += fmt.Sprintf("- **Commits**: %d\n\n", context.CommitCount)

	// Group files by type
	sourceFiles := []string{}
	testFiles := []string{}
	docFiles := []string{}
	configFiles := []string{}

	for _, file := range context.ChangedFiles {
		switch file.Type {
		case "source":
			sourceFiles = append(sourceFiles, file.Path)
		case prTypeTest:
			testFiles = append(testFiles, file.Path)
		case "doc":
			docFiles = append(docFiles, file.Path)
		case "config":
			configFiles = append(configFiles, file.Path)
		}
	}

	if len(sourceFiles) > 0 {
		content += "### Source Files\n"
		for _, file := range sourceFiles {
			content += fmt.Sprintf("- `%s`\n", file)
		}
		content += "\n"
	}

	if len(testFiles) > 0 {
		content += "### Test Files\n"
		for _, file := range testFiles {
			content += fmt.Sprintf("- `%s`\n", file)
		}
		content += "\n"
	}

	if len(docFiles) > 0 {
		content += "### Documentation\n"
		for _, file := range docFiles {
			content += fmt.Sprintf("- `%s`\n", file)
		}
		content += "\n"
	}

	if len(configFiles) > 0 {
		content += "### Configuration\n"
		for _, file := range configFiles {
			content += fmt.Sprintf("- `%s`\n", file)
		}
		content += "\n"
	}

	return TemplateSection{
		Name:     "changes",
		Title:    "Changes",
		Content:  content,
		Required: true,
		Order:    2,
	}
}

// createFeatureSection creates the feature section
func (tg *TemplateGenerator) createFeatureSection(context *TemplateContext) TemplateSection {
	content := "## New Features\n\n"
	content += "### Description\n"
	content += "<!-- Describe the new features added in this PR -->\n\n"
	content += "### Use Cases\n"
	content += "<!-- Describe the use cases for these features -->\n\n"
	content += "### API Changes\n"
	content += "<!-- List any API changes -->\n\n"

	return TemplateSection{
		Name:     "features",
		Title:    "New Features",
		Content:  content,
		Required: false,
		Order:    3,
	}
}

// createBugfixSection creates the bugfix section
func (tg *TemplateGenerator) createBugfixSection(context *TemplateContext) TemplateSection {
	content := "## Bug Fixes\n\n"
	content += "### Issues Fixed\n"

	if len(context.IssueReferences) > 0 {
		for _, ref := range context.IssueReferences {
			content += fmt.Sprintf("- Fixes %s\n", ref)
		}
	} else {
		content += "<!-- List the issues fixed by this PR -->\n"
	}

	content += "\n### Root Cause\n"
	content += "<!-- Describe the root cause of the bug -->\n\n"
	content += "### Solution\n"
	content += "<!-- Describe how the bug was fixed -->\n\n"

	return TemplateSection{
		Name:     "bugfix",
		Title:    "Bug Fixes",
		Content:  content,
		Required: false,
		Order:    3,
	}
}

// createBreakingChangesSection creates the breaking changes section
func (tg *TemplateGenerator) createBreakingChangesSection(context *TemplateContext) TemplateSection {
	content := "## ⚠️ Breaking Changes\n\n"
	content += "### Changes\n"
	content += "<!-- List all breaking changes -->\n\n"
	content += "### Migration Guide\n"
	content += "<!-- Provide migration instructions -->\n\n"
	content += "### Deprecations\n"
	content += "<!-- List any deprecations -->\n\n"

	return TemplateSection{
		Name:     "breaking-changes",
		Title:    "Breaking Changes",
		Content:  content,
		Required: true,
		Order:    4,
	}
}

// createTestingSection creates the testing section
func (tg *TemplateGenerator) createTestingSection(context *TemplateContext) TemplateSection {
	content := "## Testing\n\n"
	content += "### Test Coverage\n"

	if context.TestCoverage > 0 {
		content += fmt.Sprintf("- Coverage: %.1f%%\n", context.TestCoverage*100)
	}

	content += fmt.Sprintf("- Test files: %d\n", len(context.TestFiles))
	content += "\n### Test Files\n"

	for _, file := range context.TestFiles {
		content += fmt.Sprintf("- `%s`\n", file)
	}

	content += "\n### Testing Instructions\n"
	content += "<!-- Describe how to test these changes -->\n\n"

	return TemplateSection{
		Name:     "testing",
		Title:    "Testing",
		Content:  content,
		Required: false,
		Order:    5,
	}
}

// createChecklistSection creates the checklist section
func (tg *TemplateGenerator) createChecklistSection(context *TemplateContext) TemplateSection {
	content := "## Checklist\n\n"
	content += "- [ ] Code follows the project's style guidelines\n"
	content += "- [ ] Self-review of the code has been performed\n"
	content += "- [ ] Code is well-commented, particularly in hard-to-understand areas\n"
	content += "- [ ] Corresponding changes to documentation have been made\n"
	content += "- [ ] No new warnings are introduced\n"
	content += "- [ ] All tests pass locally\n"

	if len(context.TestFiles) > 0 {
		content += "- [ ] New tests have been added for new functionality\n"
	}

	if context.BreakingChanges {
		content += "- [ ] Breaking changes are documented\n"
		content += "- [ ] Migration guide is provided\n"
	}

	if context.DatabaseChanges {
		content += "- [ ] Database migrations are included\n"
		content += "- [ ] Database changes are backward compatible\n"
	}

	content += "\n"

	return TemplateSection{
		Name:     "checklist",
		Title:    "Checklist",
		Content:  content,
		Required: false,
		Order:    8,
	}
}

// createMetadataSection creates the metadata section
func (tg *TemplateGenerator) createMetadataSection(context *TemplateContext) TemplateSection {
	content := "## Metadata\n\n"
	content += fmt.Sprintf("- **Change Type**: %s\n", context.ChangeType)
	content += fmt.Sprintf("- **Priority**: %s\n", context.Priority)
	content += fmt.Sprintf("- **Complexity**: %s\n", context.Complexity)
	content += fmt.Sprintf("- **Estimated Review Time**: %s\n", context.EstimatedTime)

	if context.AnalysisResults.MainLanguage != "" {
		content += fmt.Sprintf("- **Primary Language**: %s\n", context.AnalysisResults.MainLanguage)
	}

	if context.AnalysisResults.QualityScore > 0 {
		content += fmt.Sprintf("- **Code Quality Score**: %.1f/10\n", context.AnalysisResults.QualityScore*10)
	}

	content += "\n"

	return TemplateSection{
		Name:     "metadata",
		Title:    "Metadata",
		Content:  content,
		Required: false,
		Order:    9,
	}
}

// Helper methods

func (tg *TemplateGenerator) createDocumentationSection(context *TemplateContext) TemplateSection {
	content := "## Documentation Updates\n\n"
	content += "### Files Updated\n"
	for _, file := range context.DocFiles {
		content += fmt.Sprintf("- `%s`\n", file)
	}
	content += "\n### Changes\n"
	content += "<!-- Describe the documentation changes -->\n\n"

	return TemplateSection{
		Name:     "documentation",
		Title:    "Documentation",
		Content:  content,
		Required: false,
		Order:    3,
	}
}

func (tg *TemplateGenerator) createTestSection(context *TemplateContext) TemplateSection {
	content := "## Test Updates\n\n"
	content += "### Test Files Added/Modified\n"
	for _, file := range context.TestFiles {
		content += fmt.Sprintf("- `%s`\n", file)
	}
	content += "\n### Test Coverage Impact\n"
	if context.TestCoverage > 0 {
		content += fmt.Sprintf("- New coverage: %.1f%%\n", context.TestCoverage*100)
	}
	content += "\n"

	return TemplateSection{
		Name:     "test-update",
		Title:    "Test Updates",
		Content:  content,
		Required: false,
		Order:    3,
	}
}

func (tg *TemplateGenerator) createRefactorSection(context *TemplateContext) TemplateSection {
	content := "## Refactoring\n\n"
	content += "### What was refactored\n"
	content += "<!-- Describe what code was refactored -->\n\n"
	content += "### Why it was refactored\n"
	content += "<!-- Explain the motivation for refactoring -->\n\n"
	content += "### Impact\n"
	content += "<!-- Describe the impact of the refactoring -->\n\n"

	return TemplateSection{
		Name:     "refactor",
		Title:    "Refactoring",
		Content:  content,
		Required: false,
		Order:    3,
	}
}

func (tg *TemplateGenerator) generateTemplateDescription(context *TemplateContext) string {
	return fmt.Sprintf("Template for %s changes", context.ChangeType)
}

func (tg *TemplateGenerator) generateConditions(context *TemplateContext) []TemplateCondition {
	var conditions []TemplateCondition

	// Add conditions based on context
	if context.BreakingChanges {
		conditions = append(conditions, TemplateCondition{
			Field:    "breaking_changes",
			Operator: "equals",
			Value:    "true",
		})
	}

	return conditions
}

func (tg *TemplateGenerator) shouldIncludeSection(section TemplateSection, context *TemplateContext) bool {
	// Check section conditions
	switch section.Name {
	case "breaking-changes":
		return context.BreakingChanges
	case "testing":
		return len(context.TestFiles) > 0
	case prTypeDocumentation:
		return len(context.DocFiles) > 0
	default:
		return true
	}
}

func (tg *TemplateGenerator) renderSection(section TemplateSection, context *TemplateContext) (string, error) {
	// Parse and execute template
	tmpl, err := template.New(section.Name).Parse(section.Content)
	if err != nil {
		return "", fmt.Errorf("invalid template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, context); err != nil {
		return "", fmt.Errorf("template execution failed: %w", err)
	}

	return result.String(), nil
}

func (tg *TemplateGenerator) createFileSummary(path string, changes []commit.FileChange) FileSummary {
	summary := FileSummary{
		Path: path,
		Type: "source",
	}

	// Find the change for this file
	for _, change := range changes {
		if change.Path == path {
			summary.Status = change.Type.String()
			summary.Additions = change.Additions
			summary.Deletions = change.Deletions
			break
		}
	}

	// Determine file type
	if tg.isTestFile(path) {
		summary.Type = "test"
	} else if tg.isDocFile(path) {
		summary.Type = "doc"
	} else if tg.isConfigFile(path) {
		summary.Type = "config"
	}

	// Determine language
	summary.Language = tg.detectLanguage(path)

	return summary
}

func (tg *TemplateGenerator) buildAnalysisSummary(result *analysis.AnalysisResult) AnalysisSummary {
	summary := AnalysisSummary{}

	if result.ProjectInfo != nil {
		summary.MainLanguage = result.ProjectInfo.MainLanguage
		summary.ProjectType = result.ProjectInfo.ProjectType.String()
	}

	// Extract languages
	for _, lang := range result.Languages {
		summary.Languages = append(summary.Languages, lang.Name)
	}

	// Extract frameworks
	for _, framework := range result.Frameworks {
		summary.Frameworks = append(summary.Frameworks, framework.Name)
	}

	if result.Dependencies != nil {
		summary.Dependencies = len(result.Dependencies.Direct)
	}

	return summary
}

func (tg *TemplateGenerator) generateTitle(context *TemplateContext) string {
	if len(context.Commits) == 1 {
		return context.Commits[0].Subject
	}
	return fmt.Sprintf("%s: Multiple changes", titleCaser.String(context.ChangeType))
}

func (tg *TemplateGenerator) isTestFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, "test") ||
		strings.Contains(lower, "spec") ||
		strings.HasSuffix(lower, "_test.go") ||
		strings.HasSuffix(lower, ".test.js")
}

func (tg *TemplateGenerator) isDocFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".md") ||
		strings.HasSuffix(lower, ".rst") ||
		strings.Contains(lower, "readme") ||
		strings.Contains(lower, "doc")
}

func (tg *TemplateGenerator) isConfigFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".json") ||
		strings.HasSuffix(lower, ".yaml") ||
		strings.HasSuffix(lower, ".yml") ||
		strings.Contains(lower, "config")
}

func (tg *TemplateGenerator) isDatabaseFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, "migration") ||
		strings.Contains(lower, "schema") ||
		strings.Contains(lower, ".sql")
}

func (tg *TemplateGenerator) detectLanguage(path string) string {
	switch strings.ToLower(strings.TrimPrefix(strings.ToLower(path), ".")) {
	case "go":
		return "Go"
	case "js", "jsx":
		return "JavaScript"
	case "ts", "tsx":
		return "TypeScript"
	case "py":
		return "Python"
	case "java":
		return "Java"
	case "rs":
		return "Rust"
	case "cpp", "cc", "cxx":
		return "C++"
	case "c":
		return "C"
	default:
		return "Unknown"
	}
}
