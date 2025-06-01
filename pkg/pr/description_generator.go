package pr

import (
	"context"
	"fmt"
	"strings"

	"github.com/fumiya-kume/cca/pkg/analysis"
	"github.com/fumiya-kume/cca/pkg/commit"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var titleCaser = cases.Title(language.English)

// DescriptionGenerator generates pull request descriptions
type DescriptionGenerator struct {
	config DescriptionGeneratorConfig
}

// DescriptionGeneratorConfig configures the description generator
type DescriptionGeneratorConfig struct {
	IncludeAnalysis   bool
	IncludeChecklist  bool
	IncludeMetadata   bool
	IncludeImpact     bool
	IncludeReferences bool
	MaxLength         int
	Style             DescriptionStyle
	CustomSections    []string
}

// DescriptionStyle defines the style of the description
type DescriptionStyle int

const (
	DescriptionStyleConcise DescriptionStyle = iota
	DescriptionStyleDetailed
	DescriptionStyleTechnical
	DescriptionStyleBusiness
)

// NewDescriptionGenerator creates a new description generator
func NewDescriptionGenerator(config DescriptionGeneratorConfig) *DescriptionGenerator {
	// Set defaults
	if config.MaxLength == 0 {
		config.MaxLength = 2000 // GitHub's limit is much higher, but keep it reasonable
	}

	return &DescriptionGenerator{
		config: config,
	}
}

// GenerateDescription generates a comprehensive PR description
func (dg *DescriptionGenerator) GenerateDescription(ctx context.Context, commitPlan *commit.CommitPlan, analysisResult *analysis.AnalysisResult, metadata *PRMetadata) (string, error) {
	var description strings.Builder

	// Build description sections
	sections := dg.buildDescriptionSections(commitPlan, analysisResult, metadata)

	// Render sections
	for i, section := range sections {
		if i > 0 {
			description.WriteString("\n\n")
		}
		description.WriteString(section)
	}

	result := description.String()

	// Ensure description doesn't exceed max length
	if len(result) > dg.config.MaxLength {
		result = dg.truncateDescription(result)
	}

	return result, nil
}

// buildDescriptionSections builds the main sections of the description
func (dg *DescriptionGenerator) buildDescriptionSections(commitPlan *commit.CommitPlan, analysisResult *analysis.AnalysisResult, metadata *PRMetadata) []string {
	var sections []string

	// Summary section (always included)
	sections = append(sections, dg.generateSummarySection(commitPlan, metadata))

	// Changes section
	sections = append(sections, dg.generateChangesSection(commitPlan))

	// Impact section
	if dg.config.IncludeImpact {
		if impact := dg.generateImpactSection(commitPlan, analysisResult, metadata); impact != "" {
			sections = append(sections, impact)
		}
	}

	// Technical details section
	if dg.config.Style == DescriptionStyleTechnical || dg.config.Style == DescriptionStyleDetailed {
		if technical := dg.generateTechnicalSection(commitPlan, analysisResult); technical != "" {
			sections = append(sections, technical)
		}
	}

	// Testing section
	if testing := dg.generateTestingSection(commitPlan, metadata); testing != "" {
		sections = append(sections, testing)
	}

	// References section
	if dg.config.IncludeReferences {
		if references := dg.generateReferencesSection(metadata); references != "" {
			sections = append(sections, references)
		}
	}

	// Analysis section
	if dg.config.IncludeAnalysis && analysisResult != nil {
		if analysis := dg.generateAnalysisSection(analysisResult); analysis != "" {
			sections = append(sections, analysis)
		}
	}

	// Checklist section
	if dg.config.IncludeChecklist {
		sections = append(sections, dg.generateChecklistSection(commitPlan, metadata))
	}

	// Metadata section
	if dg.config.IncludeMetadata {
		sections = append(sections, dg.generateMetadataSection(metadata))
	}

	return sections
}

// generateSummarySection generates the summary section
func (dg *DescriptionGenerator) generateSummarySection(commitPlan *commit.CommitPlan, metadata *PRMetadata) string {
	var summary strings.Builder

	summary.WriteString("## Summary\n\n")

	if commitPlan == nil || len(commitPlan.Commits) == 0 {
		summary.WriteString("This pull request contains updates to the repository.\n")
		return summary.String()
	}

	// Generate summary based on style
	switch dg.config.Style {
	case DescriptionStyleConcise:
		summary.WriteString(dg.generateConciseSummary(commitPlan, metadata))
	case DescriptionStyleBusiness:
		summary.WriteString(dg.generateBusinessSummary(commitPlan, metadata))
	case DescriptionStyleTechnical:
		summary.WriteString(dg.generateTechnicalSummary(commitPlan, metadata))
	default:
		summary.WriteString(dg.generateDetailedSummary(commitPlan, metadata))
	}

	// Add impact indicators
	if metadata != nil {
		switch metadata.ChangeType {
		case ChangeTypeFeature:
			summary.WriteString("\n🚀 **New Feature**: This PR introduces new functionality.")
		case ChangeTypeBugfix:
			summary.WriteString("\n🐛 **Bug Fix**: This PR resolves reported issues.")
		}

		// Add breaking change warning
		for _, commit := range commitPlan.Commits {
			if commit.Breaking {
				summary.WriteString("\n\n⚠️ **BREAKING CHANGES**: This PR contains breaking changes that may require updates to dependent code.")
				break
			}
		}

		// Add priority indicator
		if metadata.Priority == PriorityCritical || metadata.Priority == PriorityUrgent {
			summary.WriteString(fmt.Sprintf("\n🚨 **%s Priority**: This change requires immediate attention.", metadata.Priority.String()))
		}
	}

	return summary.String()
}

// generateConciseSummary generates a concise summary
func (dg *DescriptionGenerator) generateConciseSummary(commitPlan *commit.CommitPlan, metadata *PRMetadata) string {
	if len(commitPlan.Commits) == 1 {
		commit := commitPlan.Commits[0]
		return fmt.Sprintf("This PR %s.", strings.ToLower(commit.Message.Subject))
	}

	return fmt.Sprintf("This PR contains %d commits with various improvements and changes.", len(commitPlan.Commits))
}

// generateBusinessSummary generates a business-focused summary
func (dg *DescriptionGenerator) generateBusinessSummary(commitPlan *commit.CommitPlan, metadata *PRMetadata) string {
	var summary strings.Builder

	// Focus on business value
	if metadata != nil {
		switch metadata.ChangeType {
		case ChangeTypeFeature:
			summary.WriteString("This update delivers new functionality that enhances user experience and system capabilities.")
		case ChangeTypeBugfix:
			summary.WriteString("This update resolves critical issues to improve system reliability and user satisfaction.")
		case ChangeTypeRefactor:
			summary.WriteString("This update optimizes system performance to deliver faster response times and better resource utilization.")
		default:
			summary.WriteString("This update improves the overall quality and maintainability of the system.")
		}
	}

	// Add business impact
	summary.WriteString(fmt.Sprintf("\n\n**Impact**: Affects %d files with %d total changes.",
		len(dg.getAllFiles(commitPlan)), commitPlan.TotalChanges))

	if metadata != nil && metadata.EstimatedReviewTime > 0 {
		summary.WriteString(fmt.Sprintf(" Estimated review time: %s.", metadata.EstimatedReviewTime.String()))
	}

	return summary.String()
}

// generateTechnicalSummary generates a technical summary
func (dg *DescriptionGenerator) generateTechnicalSummary(commitPlan *commit.CommitPlan, metadata *PRMetadata) string {
	var summary strings.Builder

	summary.WriteString("### Technical Overview\n\n")

	// List commits with technical details
	for i, commit := range commitPlan.Commits {
		summary.WriteString(fmt.Sprintf("%d. **%s**: %s\n",
			i+1, titleCaser.String(commit.Type.String()), commit.Message.Subject))

		if commit.Scope != "" {
			summary.WriteString(fmt.Sprintf("   - Scope: `%s`\n", commit.Scope))
		}

		summary.WriteString(fmt.Sprintf("   - Files: %d, Changes: %d additions, %d deletions\n",
			len(commit.Files),
			dg.countAdditions(commit.Changes),
			dg.countDeletions(commit.Changes)))

		if commit.Breaking {
			summary.WriteString("   - ⚠️ Contains breaking changes\n")
		}

		summary.WriteString("\n")
	}

	return summary.String()
}

// generateDetailedSummary generates a detailed summary
func (dg *DescriptionGenerator) generateDetailedSummary(commitPlan *commit.CommitPlan, metadata *PRMetadata) string {
	var summary strings.Builder

	if len(commitPlan.Commits) == 1 {
		commit := commitPlan.Commits[0]
		summary.WriteString(fmt.Sprintf("This pull request %s.\n\n", strings.ToLower(commit.Message.Subject)))

		if commit.Message.Body != "" {
			summary.WriteString("### Details\n\n")
			summary.WriteString(commit.Message.Body)
			summary.WriteString("\n\n")
		}
	} else {
		summary.WriteString(fmt.Sprintf("This pull request contains %d commits that work together to:\n\n", len(commitPlan.Commits)))

		// Group commits by type
		commitsByType := make(map[commit.CommitType][]commit.PlannedCommit)
		for _, c := range commitPlan.Commits {
			commitsByType[c.Type] = append(commitsByType[c.Type], c)
		}

		for commitType, commits := range commitsByType {
			summary.WriteString(fmt.Sprintf("**%s Changes:**\n", titleCaser.String(commitType.String())))
			for _, c := range commits {
				summary.WriteString(fmt.Sprintf("- %s\n", c.Message.Subject))
			}
			summary.WriteString("\n")
		}
	}

	return summary.String()
}

// generateChangesSection generates the changes section
func (dg *DescriptionGenerator) generateChangesSection(commitPlan *commit.CommitPlan) string {
	if commitPlan == nil {
		return ""
	}

	var changes strings.Builder
	changes.WriteString("## Changes\n\n")

	// Overall statistics
	allFiles := dg.getAllFiles(commitPlan)
	totalAdditions := 0
	totalDeletions := 0

	for _, commit := range commitPlan.Commits {
		totalAdditions += dg.countAdditions(commit.Changes)
		totalDeletions += dg.countDeletions(commit.Changes)
	}

	changes.WriteString(fmt.Sprintf("- **%d** files changed\n", len(allFiles)))
	changes.WriteString(fmt.Sprintf("- **%d** additions\n", totalAdditions))
	changes.WriteString(fmt.Sprintf("- **%d** deletions\n", totalDeletions))
	changes.WriteString(fmt.Sprintf("- **%d** commits\n\n", len(commitPlan.Commits)))

	// Categorize files
	sourceFiles, testFiles, docFiles, configFiles := dg.categorizeFiles(allFiles)

	if len(sourceFiles) > 0 {
		changes.WriteString("### Source Code\n")
		dg.writeFileList(&changes, sourceFiles, 5)
		changes.WriteString("\n")
	}

	if len(testFiles) > 0 {
		changes.WriteString("### Tests\n")
		dg.writeFileList(&changes, testFiles, 5)
		changes.WriteString("\n")
	}

	if len(docFiles) > 0 {
		changes.WriteString("### Documentation\n")
		dg.writeFileList(&changes, docFiles, 5)
		changes.WriteString("\n")
	}

	if len(configFiles) > 0 {
		changes.WriteString("### Configuration\n")
		dg.writeFileList(&changes, configFiles, 5)
		changes.WriteString("\n")
	}

	return changes.String()
}

// generateImpactSection generates the impact section
func (dg *DescriptionGenerator) generateImpactSection(commitPlan *commit.CommitPlan, analysisResult *analysis.AnalysisResult, metadata *PRMetadata) string {
	impacts := dg.analyzeImpacts(commitPlan, analysisResult, metadata)
	if len(impacts) == 0 {
		return ""
	}

	var impact strings.Builder
	impact.WriteString("## Impact Analysis\n\n")

	for _, imp := range impacts {
		impact.WriteString(fmt.Sprintf("- **%s**: %s\n", imp.Type, imp.Description))
	}

	return impact.String()
}

// generateTechnicalSection generates the technical details section
func (dg *DescriptionGenerator) generateTechnicalSection(commitPlan *commit.CommitPlan, analysisResult *analysis.AnalysisResult) string {
	var technical strings.Builder
	technical.WriteString("## Technical Details\n\n")

	// Add architecture information
	if analysisResult != nil && analysisResult.ProjectInfo != nil {
		technical.WriteString(fmt.Sprintf("- **Language**: %s\n", analysisResult.ProjectInfo.MainLanguage))
		technical.WriteString(fmt.Sprintf("- **Project Type**: %s\n", analysisResult.ProjectInfo.ProjectType.String()))

		if len(analysisResult.Frameworks) > 0 {
			frameworks := make([]string, len(analysisResult.Frameworks))
			for i, fw := range analysisResult.Frameworks {
				frameworks[i] = fw.Name
			}
			technical.WriteString(fmt.Sprintf("- **Frameworks**: %s\n", strings.Join(frameworks, ", ")))
		}
		technical.WriteString("\n")
	}

	// Add commit strategy information
	if commitPlan != nil {
		technical.WriteString("### Commit Strategy\n\n")
		technical.WriteString(fmt.Sprintf("- **Strategy**: %s\n", commitPlan.Strategy.String()))
		technical.WriteString(fmt.Sprintf("- **Total Commits**: %d\n", len(commitPlan.Commits)))

		if commitPlan.EstimatedTime > 0 {
			technical.WriteString(fmt.Sprintf("- **Estimated Time**: %s\n", commitPlan.EstimatedTime.String()))
		}
		technical.WriteString("\n")
	}

	return technical.String()
}

// generateTestingSection generates the testing section
func (dg *DescriptionGenerator) generateTestingSection(commitPlan *commit.CommitPlan, metadata *PRMetadata) string {
	if commitPlan == nil {
		return ""
	}

	allFiles := dg.getAllFiles(commitPlan)
	_, testFiles, _, _ := dg.categorizeFiles(allFiles)

	if len(testFiles) == 0 && (metadata == nil || metadata.TestCoverage == 0) {
		return ""
	}

	var testing strings.Builder
	testing.WriteString("## Testing\n\n")

	if len(testFiles) > 0 {
		testing.WriteString(fmt.Sprintf("- **Test files**: %d\n", len(testFiles)))
		for _, file := range testFiles {
			testing.WriteString(fmt.Sprintf("  - `%s`\n", file))
		}
		testing.WriteString("\n")
	}

	if metadata != nil && metadata.TestCoverage > 0 {
		testing.WriteString(fmt.Sprintf("- **Test coverage**: %.1f%%\n\n", metadata.TestCoverage*100))
	}

	testing.WriteString("### Testing Checklist\n\n")
	testing.WriteString("- [ ] All existing tests pass\n")
	testing.WriteString("- [ ] New tests added for new functionality\n")
	testing.WriteString("- [ ] Edge cases are covered\n")
	testing.WriteString("- [ ] Integration tests updated if needed\n")

	return testing.String()
}

// generateReferencesSection generates the references section
func (dg *DescriptionGenerator) generateReferencesSection(metadata *PRMetadata) string {
	if metadata == nil || len(metadata.IssueReferences) == 0 {
		return ""
	}

	var references strings.Builder
	references.WriteString("## Related Issues\n\n")

	for _, ref := range metadata.IssueReferences {
		references.WriteString(fmt.Sprintf("- Closes %s\n", ref))
	}

	return references.String()
}

// generateAnalysisSection generates the analysis section
func (dg *DescriptionGenerator) generateAnalysisSection(analysisResult *analysis.AnalysisResult) string {
	var analysisSection strings.Builder
	analysisSection.WriteString("## Code Analysis\n\n")

	if analysisResult.ProjectInfo != nil {
		analysisSection.WriteString(fmt.Sprintf("- **Project**: %s\n", analysisResult.ProjectInfo.Name))
		analysisSection.WriteString(fmt.Sprintf("- **Main Language**: %s\n", analysisResult.ProjectInfo.MainLanguage))
		analysisSection.WriteString(fmt.Sprintf("- **Type**: %s\n", analysisResult.ProjectInfo.ProjectType.String()))
	}

	if len(analysisResult.Languages) > 0 {
		var langs []string
		for _, lang := range analysisResult.Languages {
			langs = append(langs, fmt.Sprintf("%s (%.1f%%)", lang.Name, lang.Percentage*100))
		}
		analysisSection.WriteString(fmt.Sprintf("- **Languages**: %s\n", strings.Join(langs, ", ")))
	}

	if analysisResult.Dependencies != nil {
		analysisSection.WriteString(fmt.Sprintf("- **Dependencies**: %d direct\n",
			len(analysisResult.Dependencies.Direct)))
	}

	return analysisSection.String()
}

// generateChecklistSection generates the checklist section
func (dg *DescriptionGenerator) generateChecklistSection(commitPlan *commit.CommitPlan, metadata *PRMetadata) string {
	var checklist strings.Builder
	checklist.WriteString("## Review Checklist\n\n")

	// Standard checklist items
	checklist.WriteString("- [ ] Code follows project style guidelines\n")
	checklist.WriteString("- [ ] Self-review completed\n")
	checklist.WriteString("- [ ] Code is well-documented\n")
	checklist.WriteString("- [ ] No new warnings introduced\n")
	checklist.WriteString("- [ ] All tests pass\n")

	// Conditional checklist items
	if commitPlan != nil {
		allFiles := dg.getAllFiles(commitPlan)
		_, testFiles, docFiles, _ := dg.categorizeFiles(allFiles)

		if len(testFiles) > 0 {
			checklist.WriteString("- [ ] New tests cover added functionality\n")
		}

		if len(docFiles) > 0 {
			checklist.WriteString("- [ ] Documentation is up to date\n")
		}

		// Check for breaking changes
		hasBreaking := false
		for _, commit := range commitPlan.Commits {
			if commit.Breaking {
				hasBreaking = true
				break
			}
		}

		if hasBreaking {
			checklist.WriteString("- [ ] Breaking changes are documented\n")
			checklist.WriteString("- [ ] Migration guide provided\n")
		}

		// Check for database changes
		if dg.hasDatabaseChanges(allFiles) {
			checklist.WriteString("- [ ] Database migrations included\n")
			checklist.WriteString("- [ ] Backward compatibility verified\n")
		}

		// Check for security changes
		if dg.hasSecurityChanges(allFiles) {
			checklist.WriteString("- [ ] Security review completed\n")
			checklist.WriteString("- [ ] No sensitive data exposed\n")
		}
	}

	return checklist.String()
}

// generateMetadataSection generates the metadata section
func (dg *DescriptionGenerator) generateMetadataSection(metadata *PRMetadata) string {
	if metadata == nil {
		return ""
	}

	var metadataSection strings.Builder
	metadataSection.WriteString("## Metadata\n\n")

	metadataSection.WriteString(fmt.Sprintf("- **Change Type**: %s\n", metadata.ChangeType.String()))
	metadataSection.WriteString(fmt.Sprintf("- **Priority**: %s\n", metadata.Priority.String()))
	metadataSection.WriteString(fmt.Sprintf("- **Complexity**: %s\n", metadata.Complexity.String()))

	if metadata.EstimatedReviewTime > 0 {
		metadataSection.WriteString(fmt.Sprintf("- **Estimated Review Time**: %s\n", metadata.EstimatedReviewTime.String()))
	}

	if metadata.TestCoverage > 0 {
		metadataSection.WriteString(fmt.Sprintf("- **Test Coverage**: %.1f%%\n", metadata.TestCoverage*100))
	}

	metadataSection.WriteString(fmt.Sprintf("- **Generated By**: %s\n", metadata.GeneratedBy))
	metadataSection.WriteString(fmt.Sprintf("- **Automation Level**: %s\n", metadata.AutomationLevel.String()))

	if len(metadata.Tags) > 0 {
		metadataSection.WriteString(fmt.Sprintf("- **Tags**: %s\n", strings.Join(metadata.Tags, ", ")))
	}

	return metadataSection.String()
}

// Helper methods

func (dg *DescriptionGenerator) getAllFiles(commitPlan *commit.CommitPlan) []string {
	fileSet := make(map[string]bool)
	for _, commit := range commitPlan.Commits {
		for _, file := range commit.Files {
			fileSet[file] = true
		}
	}

	var files []string
	for file := range fileSet {
		files = append(files, file)
	}
	return files
}

func (dg *DescriptionGenerator) categorizeFiles(files []string) (source, test, doc, config []string) {
	for _, file := range files {
		switch {
		case dg.isTestFile(file):
			test = append(test, file)
		case dg.isDocFile(file):
			doc = append(doc, file)
		case dg.isConfigFile(file):
			config = append(config, file)
		default:
			source = append(source, file)
		}
	}
	return
}

func (dg *DescriptionGenerator) writeFileList(builder *strings.Builder, files []string, limit int) {
	for i, file := range files {
		if i >= limit {
			fmt.Fprintf(builder, "- ... and %d more files\n", len(files)-limit)
			break
		}
		fmt.Fprintf(builder, "- `%s`\n", file)
	}
}

func (dg *DescriptionGenerator) countAdditions(changes []commit.FileChange) int {
	total := 0
	for _, change := range changes {
		total += change.Additions
	}
	return total
}

func (dg *DescriptionGenerator) countDeletions(changes []commit.FileChange) int {
	total := 0
	for _, change := range changes {
		total += change.Deletions
	}
	return total
}

func (dg *DescriptionGenerator) truncateDescription(description string) string {
	if len(description) <= dg.config.MaxLength {
		return description
	}

	// Find a good truncation point (end of a line)
	truncateAt := dg.config.MaxLength - 100 // Leave room for truncation message
	for i := truncateAt; i > truncateAt-100 && i > 0; i-- {
		if description[i] == '\n' {
			truncateAt = i
			break
		}
	}

	truncated := description[:truncateAt]
	truncated += "\n\n*[Description truncated due to length. See commits for full details.]*"

	return truncated
}

type Impact struct {
	Type        string
	Description string
	Severity    string
}

func (dg *DescriptionGenerator) analyzeImpacts(commitPlan *commit.CommitPlan, analysisResult *analysis.AnalysisResult, metadata *PRMetadata) []Impact {
	var impacts []Impact

	if commitPlan == nil {
		return impacts
	}

	// Check for breaking changes
	for _, commit := range commitPlan.Commits {
		if commit.Breaking {
			impacts = append(impacts, Impact{
				Type:        "Breaking Changes",
				Description: "This PR contains breaking changes that require attention during deployment",
				Severity:    "High",
			})
			break
		}
	}

	// Check for database impacts
	allFiles := dg.getAllFiles(commitPlan)
	if dg.hasDatabaseChanges(allFiles) {
		impacts = append(impacts, Impact{
			Type:        "Database",
			Description: "Database schema or migration files have been modified",
			Severity:    "Medium",
		})
	}

	// Check for security impacts
	if dg.hasSecurityChanges(allFiles) {
		impacts = append(impacts, Impact{
			Type:        "Security",
			Description: "Security-related code has been modified",
			Severity:    "High",
		})
	}

	// Check for configuration impacts
	if dg.hasConfigChanges(allFiles) {
		impacts = append(impacts, Impact{
			Type:        "Configuration",
			Description: "Configuration files have been modified",
			Severity:    "Medium",
		})
	}

	// Check for performance impacts
	if metadata != nil && metadata.Complexity >= ComplexityLevelComplex {
		impacts = append(impacts, Impact{
			Type:        "Performance",
			Description: "Complex changes that may impact system performance",
			Severity:    "Medium",
		})
	}

	return impacts
}

func (dg *DescriptionGenerator) isTestFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, "test") ||
		strings.Contains(lower, "spec") ||
		strings.HasSuffix(lower, "_test.go") ||
		strings.HasSuffix(lower, ".test.js")
}

func (dg *DescriptionGenerator) isDocFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".md") ||
		strings.HasSuffix(lower, ".rst") ||
		strings.Contains(lower, "readme") ||
		strings.Contains(lower, "doc")
}

func (dg *DescriptionGenerator) isConfigFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".json") ||
		strings.HasSuffix(lower, ".yaml") ||
		strings.HasSuffix(lower, ".yml") ||
		strings.HasSuffix(lower, ".toml") ||
		strings.Contains(lower, "config") ||
		strings.Contains(lower, ".env")
}

func (dg *DescriptionGenerator) hasDatabaseChanges(files []string) bool {
	for _, file := range files {
		lower := strings.ToLower(file)
		if strings.Contains(lower, "migration") ||
			strings.Contains(lower, "schema") ||
			strings.HasSuffix(lower, ".sql") {
			return true
		}
	}
	return false
}

func (dg *DescriptionGenerator) hasSecurityChanges(files []string) bool {
	for _, file := range files {
		lower := strings.ToLower(file)
		if strings.Contains(lower, "auth") ||
			strings.Contains(lower, "security") ||
			strings.Contains(lower, "crypto") ||
			strings.Contains(lower, "encrypt") {
			return true
		}
	}
	return false
}

func (dg *DescriptionGenerator) hasConfigChanges(files []string) bool {
	for _, file := range files {
		if dg.isConfigFile(file) {
			return true
		}
	}
	return false
}

// String methods for enums

func (ds DescriptionStyle) String() string {
	switch ds {
	case DescriptionStyleConcise:
		return "concise"
	case DescriptionStyleDetailed:
		return "detailed"
	case DescriptionStyleTechnical:
		return "technical"
	case DescriptionStyleBusiness:
		return "business"
	default:
		return "unknown"
	}
}
