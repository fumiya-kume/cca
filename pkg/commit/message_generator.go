package commit

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/fumiya-kume/cca/pkg/analysis"
)

// MessageGenerator generates commit messages based on changes and configuration
type MessageGenerator struct {
	config MessageGeneratorConfig
}

// MessageGeneratorConfig configures the message generator
type MessageGeneratorConfig struct {
	Style               MessageStyle
	MaxLength           int
	RequiredScopes      []string
	Template            string
	ConventionalCommits bool
	IncludeCoAuthors    bool
	IncludeReferences   bool
	AutoDetectType      bool
	AutoDetectScope     bool
	CustomTemplates     map[string]string
}

// CommitTemplate represents a template for generating commit messages
type CommitTemplate struct {
	Name        string
	Pattern     string
	Template    string
	Description string
	Variables   []string
}

// MessageContext contains context for generating commit messages
type MessageContext struct {
	Changes        []FileChange
	AnalysisResult *analysis.AnalysisResult
	CommitType     CommitType
	Scope          string
	Breaking       bool
	FilesAdded     []string
	FilesModified  []string
	FilesDeleted   []string
	TotalAdditions int
	TotalDeletions int
	MainLanguage   string
	ProjectType    string
	Features       []string
	Fixes          []string
}

// NewMessageGenerator creates a new message generator
func NewMessageGenerator(config MessageGeneratorConfig) *MessageGenerator {
	// Set defaults
	if config.MaxLength == 0 {
		config.MaxLength = 50
	}

	return &MessageGenerator{
		config: config,
	}
}

// GenerateMessage generates a commit message for the given changes
func (mg *MessageGenerator) GenerateMessage(ctx context.Context, changes []FileChange, analysisResult *analysis.AnalysisResult) (*ConventionalMessage, error) {
	if len(changes) == 0 {
		return nil, fmt.Errorf("no changes provided")
	}

	// Build message context
	messageContext := mg.buildMessageContext(changes, analysisResult)

	// Generate message based on style
	switch mg.config.Style {
	case MessageStyleConventional:
		return mg.generateConventionalMessage(messageContext)
	case MessageStyleTraditional:
		return mg.generateTraditionalMessage(messageContext)
	case MessageStyleCustom:
		return mg.generateCustomMessage(messageContext)
	default:
		return mg.generateConventionalMessage(messageContext)
	}
}

// generateConventionalMessage generates a conventional commit message
func (mg *MessageGenerator) generateConventionalMessage(ctx *MessageContext) (*ConventionalMessage, error) {
	// Determine commit type
	commitType := mg.determineCommitType(ctx)

	// Determine scope
	scope := mg.determineScope(ctx)

	// Check for breaking changes
	breaking := mg.isBreakingChange(ctx)

	// Generate subject
	subject := mg.generateSubject(ctx, commitType, scope)

	// Generate body
	body := mg.generateBody(ctx)

	// Generate footer
	footer := mg.generateFooter(ctx)

	// Build the full message
	fullMessage := mg.buildFullMessage(commitType.String(), scope, subject, body, footer, breaking)

	message := &ConventionalMessage{
		Type:     commitType.String(),
		Scope:    scope,
		Subject:  subject,
		Body:     body,
		Footer:   footer,
		Breaking: breaking,
		Full:     fullMessage,
	}

	// Add co-authors if configured
	if mg.config.IncludeCoAuthors {
		coAuthors := mg.extractCoAuthors(ctx)
		message.CoAuthors = coAuthors
	}

	// Add references if configured
	if mg.config.IncludeReferences {
		references := mg.extractReferences(ctx)
		message.References = references
	}

	return message, nil
}

// generateTraditionalMessage generates a traditional git commit message
func (mg *MessageGenerator) generateTraditionalMessage(ctx *MessageContext) (*ConventionalMessage, error) {
	subject := mg.generateTraditionalSubject(ctx)
	body := mg.generateTraditionalBody(ctx)

	message := &ConventionalMessage{
		Subject: subject,
		Body:    body,
		Full:    mg.buildTraditionalMessage(subject, body),
	}

	return message, nil
}

// generateCustomMessage generates a custom commit message using templates
func (mg *MessageGenerator) generateCustomMessage(ctx *MessageContext) (*ConventionalMessage, error) {
	if mg.config.Template == "" {
		return mg.generateConventionalMessage(ctx)
	}

	// Parse and execute template
	tmpl, err := template.New("commit").Parse(mg.config.Template)
	if err != nil {
		return nil, fmt.Errorf("invalid template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, ctx); err != nil {
		return nil, fmt.Errorf("template execution failed: %w", err)
	}

	message := &ConventionalMessage{
		Full: strings.TrimSpace(result.String()),
	}

	// Parse the generated message to extract components
	mg.parseGeneratedMessage(message)

	return message, nil
}

// buildMessageContext builds the context for message generation
func (mg *MessageGenerator) buildMessageContext(changes []FileChange, analysisResult *analysis.AnalysisResult) *MessageContext {
	ctx := &MessageContext{
		Changes:        changes,
		AnalysisResult: analysisResult,
		FilesAdded:     []string{},
		FilesModified:  []string{},
		FilesDeleted:   []string{},
		Features:       []string{},
		Fixes:          []string{},
	}

	// Categorize changes
	for _, change := range changes {
		switch change.Type {
		case ChangeTypeAdd:
			ctx.FilesAdded = append(ctx.FilesAdded, change.Path)
		case ChangeTypeModify:
			ctx.FilesModified = append(ctx.FilesModified, change.Path)
		case ChangeTypeDelete:
			ctx.FilesDeleted = append(ctx.FilesDeleted, change.Path)
		}

		ctx.TotalAdditions += change.Additions
		ctx.TotalDeletions += change.Deletions
	}

	// Extract project information
	if analysisResult != nil && analysisResult.ProjectInfo != nil {
		ctx.MainLanguage = analysisResult.ProjectInfo.MainLanguage
		ctx.ProjectType = analysisResult.ProjectInfo.ProjectType.String()
	}

	// Detect features and fixes
	ctx.Features = mg.detectFeatures(changes)
	ctx.Fixes = mg.detectFixes(changes)

	return ctx
}

// determineCommitType determines the commit type based on changes
func (mg *MessageGenerator) determineCommitType(ctx *MessageContext) CommitType {
	if mg.config.AutoDetectType {
		return mg.autoDetectCommitType(ctx)
	}

	// Default type determination logic
	if len(ctx.Fixes) > 0 {
		return CommitTypeFix
	}
	if len(ctx.FilesAdded) > 0 {
		return CommitTypeFeat
	}
	if mg.hasTestFiles(ctx.Changes) {
		return CommitTypeTest
	}
	if mg.hasDocFiles(ctx.Changes) {
		return CommitTypeDocs
	}
	if mg.hasConfigFiles(ctx.Changes) {
		return CommitTypeChore
	}

	return CommitTypeChore
}

// autoDetectCommitType automatically detects commit type using advanced heuristics
func (mg *MessageGenerator) autoDetectCommitType(ctx *MessageContext) CommitType {
	scores := make(map[CommitType]int)

	// Apply context-based scoring
	mg.scoreByContext(ctx, scores)

	// Score each file change
	for _, change := range ctx.Changes {
		mg.scoreByFileChange(change, scores)
	}

	// Return the highest scoring type
	return mg.findHighestScoringType(scores)
}

// scoreByContext applies scoring based on the message context
func (mg *MessageGenerator) scoreByContext(ctx *MessageContext, scores map[CommitType]int) {
	// Check for fixes first (highest priority)
	if len(ctx.Fixes) > 0 {
		scores[CommitTypeFix] += 20
	}

	// Check for added files (feature indicator)
	if len(ctx.FilesAdded) > 0 {
		scores[CommitTypeFeat] += 15
	}
}

// scoreByFileChange applies scoring based on individual file changes
func (mg *MessageGenerator) scoreByFileChange(change FileChange, scores map[CommitType]int) {
	path := strings.ToLower(change.Path)

	// Apply file type scoring
	mg.scoreByFileType(path, scores)

	// Apply pattern-based scoring
	mg.scoreByPatterns(path, scores)

	// Apply change type scoring
	if change.Type == ChangeTypeAdd {
		scores[CommitTypeFeat] += 12
	}
}

// scoreByFileType applies scoring based on file types
func (mg *MessageGenerator) scoreByFileType(path string, scores map[CommitType]int) {
	switch {
	case mg.isTestFile(path):
		scores[CommitTypeTest] += 10
	case mg.isDocFile(path):
		scores[CommitTypeDocs] += 10
	case mg.isConfigFile(path):
		scores[CommitTypeChore] += 8
	case mg.isBuildFile(path):
		scores[CommitTypeBuild] += 10
	case mg.isStyleFile(path):
		scores[CommitTypeStyle] += 8
	}
}

// scoreByPatterns applies scoring based on path patterns
func (mg *MessageGenerator) scoreByPatterns(path string, scores map[CommitType]int) {
	patternScores := map[string]map[CommitType]int{
		"perf":        {CommitTypePerf: 10},
		"performance": {CommitTypePerf: 10},
		"fix":         {CommitTypeFix: 15},
		"bug":         {CommitTypeFix: 15},
		"refactor":    {CommitTypeRefactor: 10},
		"rename":      {CommitTypeRefactor: 10},
	}

	for pattern, typeScores := range patternScores {
		if strings.Contains(path, pattern) {
			for commitType, score := range typeScores {
				scores[commitType] += score
			}
		}
	}
}

// findHighestScoringType returns the commit type with the highest score
func (mg *MessageGenerator) findHighestScoringType(scores map[CommitType]int) CommitType {
	var maxScore int
	detectedType := CommitTypeChore

	for commitType, score := range scores {
		if score > maxScore {
			maxScore = score
			detectedType = commitType
		}
	}

	return detectedType
}

// determineScope determines the scope based on changes and configuration
func (mg *MessageGenerator) determineScope(ctx *MessageContext) string {
	if mg.config.AutoDetectScope {
		return mg.autoDetectScope(ctx)
	}

	// Check required scopes
	if len(mg.config.RequiredScopes) > 0 {
		for _, scope := range mg.config.RequiredScopes {
			for _, change := range ctx.Changes {
				if strings.Contains(change.Path, scope) {
					return scope
				}
			}
		}
		return mg.config.RequiredScopes[0] // Default to first required scope
	}

	return ""
}

// autoDetectScope automatically detects scope based on file patterns
func (mg *MessageGenerator) autoDetectScope(ctx *MessageContext) string {
	scopeCounts := make(map[string]int)

	for _, change := range ctx.Changes {
		mg.countScopeFromDirectories(change.Path, scopeCounts)
		mg.countScopeFromPatterns(change.Path, scopeCounts)
	}

	return mg.getMostCommonScope(scopeCounts)
}

// countScopeFromDirectories extracts scope from directory structure
func (mg *MessageGenerator) countScopeFromDirectories(path string, scopeCounts map[string]int) {
	parts := strings.Split(path, "/")
	if len(parts) <= 1 {
		return
	}

	scope := parts[0]
	if scope == "." || scope == ".." {
		return
	}

	// Normalize common directory names to standard scope names
	normalizedScope := mg.normalizeScope(scope)
	scopeCounts[normalizedScope]++
}

// normalizeScope converts directory names to standard scope names
func (mg *MessageGenerator) normalizeScope(scope string) string {
	normalizationMap := map[string]string{
		"database":      "db",
		"frontend":      "ui",
		"ui":            "ui",
		"backend":       "backend",
		"server":        "backend",
		"authentication": "auth",
		"configuration": "config",
	}

	if normalized, exists := normalizationMap[scope]; exists {
		return normalized
	}
	return scope
}

// countScopeFromPatterns detects scope from path patterns
func (mg *MessageGenerator) countScopeFromPatterns(path string, scopeCounts map[string]int) {
	lowerPath := strings.ToLower(path)

	patternScopes := map[string]string{
		"api":      "api",
		"ui":       "ui",
		"frontend": "ui",
		"backend":  "backend",
		"server":   "backend",
		"db":       "db",
		"database": "db",
		"auth":     "auth",
		"config":   "config",
	}

	for pattern, scope := range patternScopes {
		if strings.Contains(lowerPath, pattern) {
			scopeCounts[scope]++
		}
	}
}

// getMostCommonScope returns the scope with the highest count
func (mg *MessageGenerator) getMostCommonScope(scopeCounts map[string]int) string {
	var maxCount int
	var detectedScope string

	for scope, count := range scopeCounts {
		if count > maxCount {
			maxCount = count
			detectedScope = scope
		}
	}

	return detectedScope
}

// isBreakingChange determines if the changes constitute a breaking change
func (mg *MessageGenerator) isBreakingChange(ctx *MessageContext) bool {
	for _, change := range ctx.Changes {
		// Deletion of files might be breaking
		if change.Type == ChangeTypeDelete {
			return true
		}

		// Changes to API files might be breaking
		if strings.Contains(strings.ToLower(change.Path), "api") {
			return true
		}

		// Large changes might be breaking
		if change.Additions+change.Deletions > 100 {
			return true
		}
	}

	return false
}

// generateSubject generates the commit subject line
func (mg *MessageGenerator) generateSubject(ctx *MessageContext, commitType CommitType, scope string) string {
	var subject string

	switch commitType {
	case CommitTypeFeat:
		subject = mg.generateFeatureSubject(ctx)
	case CommitTypeFix:
		subject = mg.generateFixSubject(ctx)
	case CommitTypeDocs:
		subject = mg.generateDocsSubject(ctx)
	case CommitTypeTest:
		subject = mg.generateTestSubject(ctx)
	case CommitTypeRefactor:
		subject = mg.generateRefactorSubject(ctx)
	case CommitTypeStyle:
		subject = mg.generateStyleSubject(ctx)
	case CommitTypePerf:
		subject = mg.generatePerfSubject(ctx)
	case CommitTypeBuild:
		subject = mg.generateBuildSubject(ctx)
	case CommitTypeCI:
		subject = mg.generateCISubject(ctx)
	default:
		subject = mg.generateGenericSubject(ctx)
	}

	// Ensure subject doesn't exceed max length
	if len(subject) > mg.config.MaxLength {
		subject = subject[:mg.config.MaxLength-3] + "..."
	}

	return subject
}

// generateFeatureSubject generates subject for feature commits
func (mg *MessageGenerator) generateFeatureSubject(ctx *MessageContext) string {
	if len(ctx.Features) > 0 {
		return fmt.Sprintf("add %s", ctx.Features[0])
	}

	if len(ctx.FilesAdded) == 1 {
		fileName := filepath.Base(ctx.FilesAdded[0])
		return fmt.Sprintf("add %s", fileName)
	}

	if len(ctx.FilesAdded) > 1 {
		return fmt.Sprintf("add %d new files", len(ctx.FilesAdded))
	}

	return "add new functionality"
}

// generateFixSubject generates subject for fix commits
func (mg *MessageGenerator) generateFixSubject(ctx *MessageContext) string {
	if len(ctx.Fixes) > 0 {
		return fmt.Sprintf("fix %s", ctx.Fixes[0])
	}

	if len(ctx.Changes) == 1 {
		fileName := filepath.Base(ctx.Changes[0].Path)
		return fmt.Sprintf("fix issue in %s", fileName)
	}

	return "fix bugs and issues"
}

// generateDocsSubject generates subject for documentation commits
func (mg *MessageGenerator) generateDocsSubject(ctx *MessageContext) string {
	if len(ctx.Changes) == 1 {
		fileName := filepath.Base(ctx.Changes[0].Path)
		return fmt.Sprintf("update %s", fileName)
	}

	return "update documentation"
}

// generateTestSubject generates subject for test commits
func (mg *MessageGenerator) generateTestSubject(ctx *MessageContext) string {
	if len(ctx.FilesAdded) > 0 {
		return "add tests"
	}

	return "update tests"
}

// generateRefactorSubject generates subject for refactor commits
func (mg *MessageGenerator) generateRefactorSubject(ctx *MessageContext) string {
	if len(ctx.Changes) == 1 {
		fileName := filepath.Base(ctx.Changes[0].Path)
		return fmt.Sprintf("refactor %s", fileName)
	}

	return "refactor code"
}

// generateStyleSubject generates subject for style commits
func (mg *MessageGenerator) generateStyleSubject(ctx *MessageContext) string {
	return "improve code style"
}

// generatePerfSubject generates subject for performance commits
func (mg *MessageGenerator) generatePerfSubject(ctx *MessageContext) string {
	return "improve performance"
}

// generateBuildSubject generates subject for build commits
func (mg *MessageGenerator) generateBuildSubject(ctx *MessageContext) string {
	return "update build configuration"
}

// generateCISubject generates subject for CI commits
func (mg *MessageGenerator) generateCISubject(ctx *MessageContext) string {
	return "update CI configuration"
}

// generateGenericSubject generates a generic subject
func (mg *MessageGenerator) generateGenericSubject(ctx *MessageContext) string {
	if len(ctx.Changes) == 1 {
		fileName := filepath.Base(ctx.Changes[0].Path)
		return fmt.Sprintf("update %s", fileName)
	}

	return fmt.Sprintf("update %d files", len(ctx.Changes))
}

// generateBody generates the commit body
func (mg *MessageGenerator) generateBody(ctx *MessageContext) string {
	var bodyParts []string

	// Add summary of changes
	if len(ctx.FilesAdded) > 0 {
		bodyParts = append(bodyParts, fmt.Sprintf("Added: %s", strings.Join(mg.limitFiles(ctx.FilesAdded, 3), ", ")))
	}
	if len(ctx.FilesModified) > 0 {
		bodyParts = append(bodyParts, fmt.Sprintf("Modified: %s", strings.Join(mg.limitFiles(ctx.FilesModified, 3), ", ")))
	}
	if len(ctx.FilesDeleted) > 0 {
		bodyParts = append(bodyParts, fmt.Sprintf("Deleted: %s", strings.Join(mg.limitFiles(ctx.FilesDeleted, 3), ", ")))
	}

	// Add statistics
	if ctx.TotalAdditions > 0 || ctx.TotalDeletions > 0 {
		bodyParts = append(bodyParts, fmt.Sprintf("Changes: +%d -%d", ctx.TotalAdditions, ctx.TotalDeletions))
	}

	return strings.Join(bodyParts, "\n")
}

// generateFooter generates the commit footer
func (mg *MessageGenerator) generateFooter(ctx *MessageContext) string {
	var footerParts []string

	// Add breaking change notice
	if mg.isBreakingChange(ctx) {
		footerParts = append(footerParts, "BREAKING CHANGE: This commit contains breaking changes")
	}

	return strings.Join(footerParts, "\n")
}

// buildFullMessage builds the complete conventional commit message
func (mg *MessageGenerator) buildFullMessage(commitType, scope, subject, body, footer string, breaking bool) string {
	var parts []string

	// Build header
	header := commitType
	if scope != "" {
		header += fmt.Sprintf("(%s)", scope)
	}
	if breaking {
		header += "!"
	}
	header += ": " + subject

	parts = append(parts, header)

	// Add body if present
	if body != "" {
		parts = append(parts, "", body)
	}

	// Add footer if present
	if footer != "" {
		parts = append(parts, "", footer)
	}

	return strings.Join(parts, "\n")
}

// generateTraditionalSubject generates a traditional commit subject
func (mg *MessageGenerator) generateTraditionalSubject(ctx *MessageContext) string {
	if len(ctx.Changes) == 1 {
		change := ctx.Changes[0]
		fileName := filepath.Base(change.Path)

		switch change.Type {
		case ChangeTypeAdd:
			return fmt.Sprintf("Add %s", fileName)
		case ChangeTypeDelete:
			return fmt.Sprintf("Remove %s", fileName)
		case ChangeTypeModify:
			return fmt.Sprintf("Update %s", fileName)
		case ChangeTypeRename:
			return fmt.Sprintf("Rename %s", fileName)
		default:
			return fmt.Sprintf("Change %s", fileName)
		}
	}

	// Multiple files
	addCount := len(ctx.FilesAdded)
	modCount := len(ctx.FilesModified)
	delCount := len(ctx.FilesDeleted)

	if addCount > 0 && modCount == 0 && delCount == 0 {
		return fmt.Sprintf("Add %d files", addCount)
	}
	if modCount > 0 && addCount == 0 && delCount == 0 {
		return fmt.Sprintf("Update %d files", modCount)
	}
	if delCount > 0 && addCount == 0 && modCount == 0 {
		return fmt.Sprintf("Remove %d files", delCount)
	}

	return fmt.Sprintf("Update %d files", len(ctx.Changes))
}

// generateTraditionalBody generates a traditional commit body
func (mg *MessageGenerator) generateTraditionalBody(ctx *MessageContext) string {
	var lines []string

	if len(ctx.Changes) <= 5 {
		for _, change := range ctx.Changes {
			var action string
			switch change.Type {
			case ChangeTypeAdd:
				action = "+"
			case ChangeTypeDelete:
				action = "-"
			case ChangeTypeModify:
				action = "M"
			case ChangeTypeRename:
				action = "R"
			default:
				action = "M"
			}
			lines = append(lines, fmt.Sprintf("%s %s", action, change.Path))
		}
	}

	return strings.Join(lines, "\n")
}

// buildTraditionalMessage builds a traditional commit message
func (mg *MessageGenerator) buildTraditionalMessage(subject, body string) string {
	if body == "" {
		return subject
	}
	return subject + "\n\n" + body
}

// parseGeneratedMessage parses a generated message to extract components
func (mg *MessageGenerator) parseGeneratedMessage(message *ConventionalMessage) {
	lines := strings.Split(message.Full, "\n")
	if len(lines) > 0 {
		// Parse conventional commit header
		header := lines[0]
		re := regexp.MustCompile(`^(\w+)(\(([^)]+)\))?(!?):\s*(.+)$`)
		matches := re.FindStringSubmatch(header)

		if len(matches) >= 6 {
			message.Type = matches[1]
			message.Scope = matches[3]
			message.Breaking = matches[4] == "!"
			message.Subject = matches[5]
		} else {
			message.Subject = header
		}

		// Extract body and footer
		if len(lines) > 2 {
			bodyStart := 2
			footerStart := len(lines)

			// Find footer (lines starting with BREAKING CHANGE:, Closes:, etc.)
			for i := len(lines) - 1; i >= bodyStart; i-- {
				if strings.HasPrefix(lines[i], "BREAKING CHANGE:") ||
					strings.HasPrefix(lines[i], "Closes:") ||
					strings.HasPrefix(lines[i], "Fixes:") {
					footerStart = i
					break
				}
			}

			if footerStart > bodyStart {
				message.Body = strings.Join(lines[bodyStart:footerStart], "\n")
			}
			if footerStart < len(lines) {
				message.Footer = strings.Join(lines[footerStart:], "\n")
			}
		}
	}
}

// Helper methods

func (mg *MessageGenerator) detectFeatures(changes []FileChange) []string {
	var features []string

	for _, change := range changes {
		if change.Type == ChangeTypeAdd {
			features = append(features, mg.extractFeatureName(change.Path))
		}
	}

	return mg.deduplicate(features)
}

func (mg *MessageGenerator) detectFixes(changes []FileChange) []string {
	var fixes []string

	for _, change := range changes {
		path := strings.ToLower(change.Path)
		if strings.Contains(path, "fix") || strings.Contains(path, "bug") {
			fixes = append(fixes, mg.extractFixName(change.Path))
		}
	}

	return mg.deduplicate(fixes)
}

func (mg *MessageGenerator) extractFeatureName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Clean up the name
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")

	return name
}

func (mg *MessageGenerator) extractFixName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Extract issue/bug from filename
	if strings.Contains(name, "fix") {
		name = strings.TrimPrefix(name, "fix_")
	}
	if strings.Contains(name, "bug") {
		name = strings.TrimPrefix(name, "bug_")
	}

	// Convert underscores to spaces for readability
	return strings.ReplaceAll(name, "_", " ")
}

func (mg *MessageGenerator) extractCoAuthors(ctx *MessageContext) []string {
	// This would be implemented based on git log analysis or configuration
	return []string{}
}

func (mg *MessageGenerator) extractReferences(ctx *MessageContext) []string {
	// This would extract issue references from file patterns or content
	return []string{}
}

func (mg *MessageGenerator) hasTestFiles(changes []FileChange) bool {
	for _, change := range changes {
		if mg.isTestFile(change.Path) {
			return true
		}
	}
	return false
}

func (mg *MessageGenerator) hasDocFiles(changes []FileChange) bool {
	for _, change := range changes {
		if mg.isDocFile(change.Path) {
			return true
		}
	}
	return false
}

func (mg *MessageGenerator) hasConfigFiles(changes []FileChange) bool {
	for _, change := range changes {
		if mg.isConfigFile(change.Path) {
			return true
		}
	}
	return false
}

func (mg *MessageGenerator) isTestFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, "test") ||
		strings.Contains(lower, "spec") ||
		strings.HasSuffix(lower, "_test.go") ||
		strings.HasSuffix(lower, ".test.js") ||
		strings.HasSuffix(lower, ".spec.js")
}

func (mg *MessageGenerator) isDocFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".md") ||
		strings.HasSuffix(lower, ".rst") ||
		strings.HasSuffix(lower, ".txt") ||
		strings.Contains(lower, "readme") ||
		strings.Contains(lower, "doc")
}

func (mg *MessageGenerator) isConfigFile(path string) bool {
	lower := strings.ToLower(path)
	configExts := []string{".json", ".yaml", ".yml", ".toml", ".ini", ".conf"}

	for _, ext := range configExts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}

	return strings.Contains(lower, "config") ||
		strings.Contains(lower, ".env") ||
		strings.Contains(lower, "dockerfile")
}

func (mg *MessageGenerator) isBuildFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, "makefile") ||
		strings.Contains(lower, "build") ||
		strings.Contains(lower, "webpack") ||
		strings.Contains(lower, "package.json") ||
		strings.Contains(lower, "go.mod") ||
		strings.Contains(lower, "cargo.toml")
}

func (mg *MessageGenerator) isStyleFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".css") ||
		strings.HasSuffix(lower, ".scss") ||
		strings.HasSuffix(lower, ".sass") ||
		strings.HasSuffix(lower, ".less")
}

func (mg *MessageGenerator) limitFiles(files []string, limit int) []string {
	if len(files) <= limit {
		return files
	}

	result := make([]string, limit)
	for i := 0; i < limit; i++ {
		result[i] = filepath.Base(files[i])
	}

	return result
}

func (mg *MessageGenerator) deduplicate(items []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0) // Initialize with empty slice instead of nil

	for _, item := range items {
		if !seen[item] && item != "" {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
