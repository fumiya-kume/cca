// Package analysis provides comprehensive codebase analysis capabilities for ccAgents
package analysis

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/fumiya-kume/cca/internal/types"
)

// String constants
const (
	unknownValue = "Unknown"
)

// ContextBuilder builds comprehensive code context for AI interactions
type ContextBuilder struct {
	config AnalyzerConfig
}

// NewContextBuilder creates a new context builder
func NewContextBuilder(config AnalyzerConfig) *ContextBuilder {
	return &ContextBuilder{
		config: config,
	}
}

// BuildContext builds comprehensive code context from analysis results
func (cb *ContextBuilder) BuildContext(ctx context.Context, analysisResult *AnalysisResult) (*types.CodeContext, error) {
	codeContext := &types.CodeContext{
		ProjectSummary:    cb.buildProjectSummary(analysisResult),
		TechnicalStack:    cb.buildTechnicalStack(analysisResult),
		Architecture:      cb.buildArchitectureInfo(analysisResult),
		CodeConventions:   cb.buildCodeConventions(analysisResult),
		RelevantFiles:     cb.identifyRelevantFiles(analysisResult),
		Dependencies:      cb.buildDependencyContext(analysisResult),
		TestingStrategy:   cb.buildTestingStrategy(analysisResult),
		BuildInformation:  cb.buildBuildInformation(analysisResult),
		DocumentationRefs: cb.buildDocumentationRefs(analysisResult),
		IssueContext:      cb.buildIssueContext(analysisResult),
		Metadata:          cb.buildContextMetadata(analysisResult),
	}

	return codeContext, nil
}

// BuildContextForIssue builds issue-specific code context
func (cb *ContextBuilder) BuildContextForIssue(ctx context.Context, analysisResult *AnalysisResult, issueData *types.IssueData) (*types.CodeContext, error) {
	// First build general context
	codeContext, err := cb.BuildContext(ctx, analysisResult)
	if err != nil {
		return nil, err
	}

	// Enhance with issue-specific context
	codeContext.IssueContext = &types.IssueContextInfo{
		IssueNumber:         issueData.Number,
		Title:               issueData.Title,
		Description:         issueData.Body,
		Labels:              issueData.Labels,
		Requirements:        cb.extractRequirements(issueData.Body),
		AcceptanceCriteria:  cb.extractAcceptanceCriteria(issueData.Body),
		MentionedFiles:      cb.extractMentionedFiles(issueData.Body, analysisResult.ProjectInfo.Name),
		RelatedIssues:       cb.extractRelatedIssues(issueData.Body),
		Complexity:          cb.estimateComplexity(issueData),
		EstimatedEffort:     cb.estimateEffort(issueData),
		RiskFactors:         cb.identifyRiskFactors(issueData, analysisResult),
		TestStrategy:        cb.suggestTestStrategy(issueData, analysisResult),
		ImplementationHints: cb.generateImplementationHints(issueData, analysisResult),
	}

	// Filter relevant files based on issue context
	codeContext.RelevantFiles = cb.filterRelevantFilesForIssue(codeContext.RelevantFiles, codeContext.IssueContext)

	return codeContext, nil
}

// buildProjectSummary creates a comprehensive project summary
func (cb *ContextBuilder) buildProjectSummary(analysis *AnalysisResult) *types.ProjectSummaryInfo {
	summary := &types.ProjectSummaryInfo{
		Name:           analysis.ProjectInfo.Name,
		Description:    analysis.ProjectInfo.Description,
		Version:        analysis.ProjectInfo.Version,
		License:        analysis.ProjectInfo.License,
		MainLanguage:   analysis.ProjectInfo.MainLanguage,
		ProjectType:    analysis.ProjectInfo.ProjectType.String(),
		BuildSystem:    analysis.ProjectInfo.BuildSystem,
		PackageManager: analysis.ProjectInfo.PackageManager,
		Tags:           analysis.ProjectInfo.Tags,
	}

	// Add language distribution
	if len(analysis.Languages) > 0 {
		summary.LanguageDistribution = make(map[string]float64)
		for _, lang := range analysis.Languages {
			summary.LanguageDistribution[lang.Name] = lang.Percentage
		}
	}

	// Add framework information
	if len(analysis.Frameworks) > 0 {
		summary.PrimaryFrameworks = make([]string, 0, len(analysis.Frameworks))
		for _, framework := range analysis.Frameworks {
			summary.PrimaryFrameworks = append(summary.PrimaryFrameworks, framework.Name)
		}
	}

	// Add basic metrics
	if analysis.FileStructure != nil {
		summary.TotalFiles = analysis.FileStructure.TotalFiles
		summary.TotalLines = analysis.FileStructure.TotalLines
		summary.ProjectSize = analysis.FileStructure.TotalSize
	}

	return summary
}

// buildTechnicalStack creates detailed technical stack information
func (cb *ContextBuilder) buildTechnicalStack(analysis *AnalysisResult) *types.TechnicalStackInfo {
	stack := &types.TechnicalStackInfo{
		Languages:        make([]types.LanguageInfo, 0, len(analysis.Languages)),
		Frameworks:       make([]types.FrameworkInfo, 0, len(analysis.Frameworks)),
		BuildTools:       []string{},
		TestingTools:     []string{},
		DevelopmentTools: []string{},
		Databases:        []string{},
		CloudServices:    []string{},
	}

	// Convert languages
	for _, lang := range analysis.Languages {
		stackLang := types.LanguageInfo{
			Name:        lang.Name,
			Version:     lang.Version,
			Percentage:  lang.Percentage,
			FileCount:   lang.FileCount,
			LineCount:   lang.LineCount,
			Extensions:  lang.Extensions,
			ConfigFiles: lang.ConfigFiles,
		}
		stack.Languages = append(stack.Languages, stackLang)
	}

	// Convert frameworks
	for _, framework := range analysis.Frameworks {
		stackFramework := types.FrameworkInfo{
			Name:    framework.Name,
			Version: framework.Version,
			Type:    framework.Type.String(),
			Purpose: cb.getFrameworkPurpose(framework.Name),
		}
		stack.Frameworks = append(stack.Frameworks, stackFramework)

		// Categorize frameworks
		switch framework.Type {
		case FrameworkTypeTesting:
			stack.TestingTools = append(stack.TestingTools, framework.Name)
		case FrameworkTypeBuild:
			stack.BuildTools = append(stack.BuildTools, framework.Name)
		}
	}

	// Add build system
	if analysis.ProjectInfo.BuildSystem != "" {
		stack.BuildTools = append(stack.BuildTools, analysis.ProjectInfo.BuildSystem)
	}

	// Detect additional tools from config files
	cb.detectToolsFromConfigFiles(analysis.ProjectInfo.ConfigFiles, stack)

	return stack
}

// buildArchitectureInfo creates architecture information
func (cb *ContextBuilder) buildArchitectureInfo(analysis *AnalysisResult) *types.ArchitectureInfo {
	arch := &types.ArchitectureInfo{
		ProjectStructure: cb.buildProjectStructure(analysis),
		DesignPatterns:   cb.identifyDesignPatterns(analysis),
		DatabaseSchema:   cb.identifyDatabaseSchema(analysis),
		APIEndpoints:     cb.identifyAPIEndpoints(analysis),
		ConfigStructure:  cb.buildConfigStructure(analysis),
		DeploymentInfo:   cb.buildDeploymentInfo(analysis),
	}

	return arch
}

// buildCodeConventions creates code conventions information
func (cb *ContextBuilder) buildCodeConventions(analysis *AnalysisResult) *types.CodeConventionsInfo {
	conventions := &types.CodeConventionsInfo{
		NamingConventions:  make(map[string]string),
		FormattingRules:    make(map[string]interface{}),
		ImportStyle:        "",
		CommentStyle:       []string{},
		FileOrganization:   "",
		TestingPatterns:    []string{},
		ErrorHandling:      "",
		DocumentationStyle: "",
	}

	if analysis.Conventions != nil {
		conventions.IndentationStyle = analysis.Conventions.IndentationStyle
		conventions.IndentationSize = analysis.Conventions.IndentationSize
		conventions.LineEndings = analysis.Conventions.LineEndings
		conventions.QuoteStyle = analysis.Conventions.QuoteStyle
		conventions.NamingConventions = analysis.Conventions.NamingConventions
		conventions.FileNaming = analysis.Conventions.FileNaming
		conventions.DirectoryNaming = analysis.Conventions.DirectoryNaming
		conventions.CommentStyle = analysis.Conventions.CommentStyle
		conventions.FormattingRules = analysis.Conventions.FormattingRules
	}

	// Extract conventions from language info
	for _, lang := range analysis.Languages {
		if lang.Conventions != nil {
			if conventions.ImportStyle == "" {
				conventions.ImportStyle = lang.Conventions.ImportStyle
			}
			if conventions.ErrorHandling == "" {
				conventions.ErrorHandling = lang.Conventions.ErrorHandling
			}
			if conventions.DocumentationStyle == "" {
				conventions.DocumentationStyle = lang.Conventions.DocumentationStyle
			}
			conventions.TestingPatterns = append(conventions.TestingPatterns, lang.Conventions.TestingPattern)
		}
	}

	return conventions
}

// identifyRelevantFiles identifies files relevant for the current context
func (cb *ContextBuilder) identifyRelevantFiles(analysis *AnalysisResult) []types.RelevantFileInfo {
	var relevantFiles []types.RelevantFileInfo

	if analysis.FileStructure != nil {
		// Add important files
		for _, file := range analysis.FileStructure.ImportantFiles {
			relevantFile := types.RelevantFileInfo{
				Path:         file.Path,
				Purpose:      cb.identifyFilePurpose(file),
				Language:     file.Language,
				Size:         file.Size,
				LastModified: file.LastModified,
				Importance:   file.Importance,
				Summary:      cb.generateFileSummary(file),
			}
			relevantFiles = append(relevantFiles, relevantFile)
		}

		// Add configuration files
		for _, configFile := range analysis.ProjectInfo.ConfigFiles {
			relevantFile := types.RelevantFileInfo{
				Path:       configFile,
				Purpose:    "Configuration",
				Language:   cb.detectConfigLanguage(configFile),
				Importance: 0.8,
				Summary:    cb.generateConfigFileSummary(configFile),
			}
			relevantFiles = append(relevantFiles, relevantFile)
		}
	}

	// Sort by importance
	sort.Slice(relevantFiles, func(i, j int) bool {
		return relevantFiles[i].Importance > relevantFiles[j].Importance
	})

	return relevantFiles
}

// buildDependencyContext creates dependency context
func (cb *ContextBuilder) buildDependencyContext(analysis *AnalysisResult) *types.DependencyContextInfo {
	if analysis.Dependencies == nil {
		return &types.DependencyContextInfo{}
	}

	depContext := &types.DependencyContextInfo{
		DirectDependencies:  len(analysis.Dependencies.Direct),
		DevDependencies:     len(analysis.Dependencies.DevDeps),
		TotalDependencies:   len(analysis.Dependencies.Direct) + len(analysis.Dependencies.Transitive),
		SecurityIssues:      len(analysis.Dependencies.Security),
		LicenseDistribution: analysis.Dependencies.Licenses,
		UpdatesAvailable:    len(analysis.Dependencies.UpdatesAvailable),
		KeyDependencies:     cb.identifyKeyDependencies(analysis.Dependencies),
		DependencyConflicts: len(analysis.Dependencies.Conflicts),
	}

	return depContext
}

// buildTestingStrategy creates testing strategy information
func (cb *ContextBuilder) buildTestingStrategy(analysis *AnalysisResult) *types.TestingStrategyInfo {
	strategy := &types.TestingStrategyInfo{
		TestFrameworks:  []string{},
		TestDirectories: analysis.ProjectInfo.TestDirs,
		CoverageTools:   []string{},
		TestPatterns:    []string{},
		TestingApproach: "",
		MockingStrategy: "",
		E2ETestingTools: []string{},
	}

	// Extract testing information from frameworks
	for _, framework := range analysis.Frameworks {
		if framework.Type == FrameworkTypeTesting {
			strategy.TestFrameworks = append(strategy.TestFrameworks, framework.Name)
		}
	}

	// Extract testing patterns from languages
	for _, lang := range analysis.Languages {
		if lang.Conventions != nil && lang.Conventions.TestingPattern != "" {
			strategy.TestPatterns = append(strategy.TestPatterns, lang.Conventions.TestingPattern)
		}
	}

	return strategy
}

// buildBuildInformation creates build information
func (cb *ContextBuilder) buildBuildInformation(analysis *AnalysisResult) *types.BuildInformationInfo {
	buildInfo := &types.BuildInformationInfo{
		BuildSystem:       analysis.ProjectInfo.BuildSystem,
		BuildFiles:        []string{},
		BuildTargets:      []string{},
		ArtifactTypes:     []string{},
		BuildScripts:      make(map[string]string),
		CompilerFlags:     []string{},
		OptimizationLevel: "",
		OutputDirectories: []string{},
	}

	// Extract build files from config files
	for _, configFile := range analysis.ProjectInfo.ConfigFiles {
		if cb.isBuildFile(configFile) {
			buildInfo.BuildFiles = append(buildInfo.BuildFiles, configFile)
		}
	}

	// Identify build targets based on project type
	buildInfo.BuildTargets = cb.identifyBuildTargets(analysis.ProjectInfo)
	buildInfo.ArtifactTypes = cb.identifyArtifactTypes(analysis.ProjectInfo)

	return buildInfo
}

// buildDocumentationRefs creates documentation references
func (cb *ContextBuilder) buildDocumentationRefs(analysis *AnalysisResult) *types.DocumentationRefsInfo {
	docRefs := &types.DocumentationRefsInfo{
		ReadmeFile:       cb.findReadmeFile(analysis),
		DocDirectories:   analysis.ProjectInfo.DocumentationDirs,
		InlineDocStyle:   "",
		APIDocumentation: []string{},
		Guides:           []string{},
		Examples:         []string{},
		ChangelogFile:    cb.findChangelogFile(analysis),
		ContributingFile: cb.findContributingFile(analysis),
	}

	return docRefs
}

// buildIssueContext creates issue context (will be overridden in BuildContextForIssue)
func (cb *ContextBuilder) buildIssueContext(analysis *AnalysisResult) *types.IssueContextInfo {
	return nil // Will be populated in BuildContextForIssue
}

// buildContextMetadata creates context metadata
func (cb *ContextBuilder) buildContextMetadata(analysis *AnalysisResult) *types.ContextMetadata {
	return &types.ContextMetadata{
		GeneratedAt:      time.Now(),
		AnalysisVersion:  "1.0.0",
		ContextVersion:   "1.0.0",
		AnalysisDuration: analysis.AnalysisTime,
		FilesAnalyzed:    cb.countAnalyzedFiles(analysis),
		DataSources:      cb.identifyDataSources(analysis),
		Confidence:       cb.calculateConfidence(analysis),
		Completeness:     cb.calculateCompleteness(analysis),
	}
}

// Helper methods for issue-specific context

func (cb *ContextBuilder) extractRequirements(issueBody string) []string {
	var requirements []string

	// Look for requirement patterns
	patterns := []string{
		`(?i)requirements?:\s*\n((?:[-*]\s*.+\n?)+)`,
		`(?i)must:\s*\n((?:[-*]\s*.+\n?)+)`,
		`(?i)should:\s*\n((?:[-*]\s*.+\n?)+)`,
		`(?i)needs? to:\s*\n((?:[-*]\s*.+\n?)+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(issueBody, -1)
		for _, match := range matches {
			if len(match) > 1 {
				lines := strings.Split(match[1], "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line != "" && (strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*")) {
						requirements = append(requirements, strings.TrimSpace(line[1:]))
					}
				}
			}
		}
	}

	return requirements
}

func (cb *ContextBuilder) extractAcceptanceCriteria(issueBody string) []string {
	var criteria []string

	// Look for acceptance criteria patterns
	patterns := []string{
		`(?i)acceptance criteria:\s*\n((?:[-*]\s*.+\n?)+)`,
		`(?i)AC:\s*\n((?:[-*]\s*.+\n?)+)`,
		`(?i)criteria:\s*\n((?:[-*]\s*.+\n?)+)`,
		`(?i)definition of done:\s*\n((?:[-*]\s*.+\n?)+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(issueBody, -1)
		for _, match := range matches {
			if len(match) > 1 {
				lines := strings.Split(match[1], "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line != "" && (strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*")) {
						criteria = append(criteria, strings.TrimSpace(line[1:]))
					}
				}
			}
		}
	}

	return criteria
}

func (cb *ContextBuilder) extractMentionedFiles(issueBody, projectName string) []string {
	var files []string

	// Look for file patterns
	patterns := []string{
		`[a-zA-Z0-9_\-./]+\.[a-zA-Z0-9]+`,              // file.ext
		`(?i)in\s+([a-zA-Z0-9_\-./]+\.[a-zA-Z0-9]+)`,   // "in file.ext"
		`(?i)file\s+([a-zA-Z0-9_\-./]+\.[a-zA-Z0-9]+)`, // "file file.ext"
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(issueBody, -1)
		for _, match := range matches {
			file := match[0]
			if len(match) > 1 {
				file = match[1]
			}

			// Filter out URLs and common false positives
			if !strings.Contains(file, "http") && !strings.Contains(file, "www") {
				files = append(files, file)
			}
		}
	}

	return files
}

func (cb *ContextBuilder) extractRelatedIssues(issueBody string) []int {
	var issues []int

	// Look for issue references
	re := regexp.MustCompile(`(?i)(?:issue|#)(\d+)`)
	matches := re.FindAllStringSubmatch(issueBody, -1)

	for _, match := range matches {
		if len(match) > 1 {
			if issueNum := parseInt(match[1]); issueNum > 0 {
				issues = append(issues, issueNum)
			}
		}
	}

	return issues
}

// Helper methods for various analysis tasks

func (cb *ContextBuilder) buildProjectStructure(analysis *AnalysisResult) map[string]interface{} {
	structure := make(map[string]interface{})

	structure["source_directories"] = analysis.ProjectInfo.SourceDirs
	structure["test_directories"] = analysis.ProjectInfo.TestDirs
	structure["documentation_directories"] = analysis.ProjectInfo.DocumentationDirs
	structure["entry_points"] = analysis.ProjectInfo.EntryPoints

	return structure
}

func (cb *ContextBuilder) identifyDesignPatterns(analysis *AnalysisResult) []string {
	var patterns []string

	// Identify patterns based on frameworks and project type
	for _, framework := range analysis.Frameworks {
		switch framework.Name {
		case "React":
			patterns = append(patterns, "Component Pattern", "Hook Pattern")
		case "Angular":
			patterns = append(patterns, "MVC", "Dependency Injection")
		case "Django":
			patterns = append(patterns, "MVT", "ORM Pattern")
		case "Spring Boot":
			patterns = append(patterns, "MVC", "Dependency Injection", "Bean Pattern")
		}
	}

	return patterns
}

func (cb *ContextBuilder) identifyDatabaseSchema(analysis *AnalysisResult) map[string]interface{} {
	schema := make(map[string]interface{})

	// Look for database-related files and dependencies
	if analysis.Dependencies != nil {
		for _, dep := range analysis.Dependencies.Direct {
			if cb.isDatabaseDependency(dep.Name) {
				schema["database_type"] = cb.getDatabaseType(dep.Name)
				break
			}
		}
	}

	return schema
}

func (cb *ContextBuilder) identifyAPIEndpoints(analysis *AnalysisResult) []map[string]interface{} {
	var endpoints []map[string]interface{}

	// This would require parsing source files for route definitions
	// Placeholder implementation

	return endpoints
}

func (cb *ContextBuilder) buildConfigStructure(analysis *AnalysisResult) map[string]interface{} {
	config := make(map[string]interface{})

	if analysis.ProjectInfo != nil {
		config["files"] = analysis.ProjectInfo.ConfigFiles
	}
	config["environment_variables"] = cb.identifyEnvironmentVariables(analysis)

	return config
}

func (cb *ContextBuilder) buildDeploymentInfo(analysis *AnalysisResult) map[string]interface{} {
	deployment := make(map[string]interface{})

	// Check for deployment-related files
	if analysis.ProjectInfo != nil {
		for _, configFile := range analysis.ProjectInfo.ConfigFiles {
			if cb.isDeploymentFile(configFile) {
				deployment["deployment_type"] = cb.getDeploymentType(configFile)
				deployment["deployment_files"] = []string{configFile}
			}
		}
	}

	return deployment
}

// Additional helper methods

func (cb *ContextBuilder) getFrameworkPurpose(frameworkName string) string {
	purposes := map[string]string{
		"React":       "Frontend UI Library",
		"Vue.js":      "Frontend Framework",
		"Angular":     "Full-stack Framework",
		"Express.js":  "Backend Web Framework",
		"Django":      "Full-stack Web Framework",
		"Spring Boot": "Enterprise Application Framework",
		"Jest":        "JavaScript Testing Framework",
		"pytest":      "Python Testing Framework",
	}

	if purpose, exists := purposes[frameworkName]; exists {
		return purpose
	}
	return unknownValue
}

func (cb *ContextBuilder) detectToolsFromConfigFiles(configFiles []string, stack *types.TechnicalStackInfo) {
	for _, file := range configFiles {
		switch {
		case strings.Contains(file, "jest"):
			stack.TestingTools = appendUnique(stack.TestingTools, "Jest")
		case strings.Contains(file, "webpack"):
			stack.BuildTools = appendUnique(stack.BuildTools, "Webpack")
		case strings.Contains(file, "eslint"):
			stack.DevelopmentTools = appendUnique(stack.DevelopmentTools, "ESLint")
		case strings.Contains(file, "prettier"):
			stack.DevelopmentTools = appendUnique(stack.DevelopmentTools, "Prettier")
		case strings.Contains(file, "docker"):
			stack.DevelopmentTools = appendUnique(stack.DevelopmentTools, "Docker")
		}
	}
}

func (cb *ContextBuilder) identifyFilePurpose(file FileInfo) string {
	switch file.Type {
	case FileTypeSource:
		return "Source Code"
	case FileTypeTest:
		return "Test"
	case FileTypeConfig:
		return "Configuration"
	case FileTypeDocumentation:
		return "Documentation"
	case FileTypeBuild:
		return "Build Script"
	default:
		return "Other"
	}
}

func (cb *ContextBuilder) generateFileSummary(file FileInfo) string {
	return fmt.Sprintf("%s file with %d lines", file.Language, file.LineCount)
}

func (cb *ContextBuilder) detectConfigLanguage(configFile string) string {
	ext := strings.ToLower(filepath.Ext(configFile))
	switch ext {
	case ".json":
		return "JSON"
	case ".yaml", ".yml":
		return "YAML"
	case ".toml":
		return "TOML"
	case ".xml":
		return "XML"
	case ".js":
		return langJavaScript
	default:
		return "Configuration"
	}
}

func (cb *ContextBuilder) generateConfigFileSummary(configFile string) string {
	return fmt.Sprintf("Configuration file for %s", cb.getConfigPurpose(configFile))
}

func (cb *ContextBuilder) getConfigPurpose(configFile string) string {
	switch {
	case strings.Contains(configFile, "package.json"):
		return "Node.js package management"
	case strings.Contains(configFile, "go.mod"):
		return "Go module management"
	case strings.Contains(configFile, "Cargo.toml"):
		return "Rust package management"
	case strings.Contains(configFile, "pom.xml"):
		return "Maven build configuration"
	default:
		return "project configuration"
	}
}

func (cb *ContextBuilder) identifyKeyDependencies(deps *DependencyInfo) []string {
	var keyDeps []string

	// Identify key dependencies based on usage patterns
	for _, dep := range deps.Direct {
		if cb.isKeyDependency(dep.Name) {
			keyDeps = append(keyDeps, dep.Name)
		}
	}

	return keyDeps
}

func (cb *ContextBuilder) isKeyDependency(name string) bool {
	keyPatterns := []string{
		"react", "vue", "angular", "express", "django", "spring",
		"lodash", "underscore", "moment", "axios", "jquery",
	}

	for _, pattern := range keyPatterns {
		if strings.Contains(strings.ToLower(name), pattern) {
			return true
		}
	}
	return false
}

func (cb *ContextBuilder) isBuildFile(filename string) bool {
	buildFiles := []string{
		"Makefile", "build.gradle", "pom.xml", "webpack.config.js",
		"rollup.config.js", "vite.config.js", "gulpfile.js",
	}

	for _, buildFile := range buildFiles {
		if strings.Contains(filename, buildFile) {
			return true
		}
	}
	return false
}

func (cb *ContextBuilder) identifyBuildTargets(projectInfo *ProjectInfo) []string {
	var targets []string

	switch projectInfo.ProjectType {
	case ProjectTypeWebApp:
		targets = []string{"build", "dev", "prod"}
	case ProjectTypeLibrary:
		targets = []string{"build", "test", "publish"}
	case ProjectTypeCLI:
		targets = []string{"build", "install"}
	default:
		targets = []string{"build", "test"}
	}

	return targets
}

func (cb *ContextBuilder) identifyArtifactTypes(projectInfo *ProjectInfo) []string {
	var artifacts []string

	switch projectInfo.MainLanguage {
	case "JavaScript", "TypeScript":
		artifacts = []string{"bundle.js", "package"}
	case "Go":
		artifacts = []string{"binary"}
	case "Java":
		artifacts = []string{"jar", "war"}
	case "Rust":
		artifacts = []string{"binary", "crate"}
	case "Python":
		artifacts = []string{"wheel", "sdist"}
	}

	return artifacts
}

func (cb *ContextBuilder) findReadmeFile(analysis *AnalysisResult) string {
	if analysis.Metadata != nil && analysis.Metadata.README != nil {
		return analysis.Metadata.README.Path
	}
	return ""
}

func (cb *ContextBuilder) findChangelogFile(analysis *AnalysisResult) string {
	// Look for changelog files in common locations
	changelogFiles := []string{"CHANGELOG.md", "CHANGELOG.txt", "HISTORY.md"}
	for _, file := range changelogFiles {
		// This would check if the file exists in the project
		return file
	}
	return ""
}

func (cb *ContextBuilder) findContributingFile(analysis *AnalysisResult) string {
	// Look for contributing files
	contributingFiles := []string{"CONTRIBUTING.md", "CONTRIBUTING.txt"}
	for _, file := range contributingFiles {
		// This would check if the file exists in the project
		return file
	}
	return ""
}

func (cb *ContextBuilder) countAnalyzedFiles(analysis *AnalysisResult) int {
	if analysis.FileStructure != nil {
		return analysis.FileStructure.TotalFiles
	}
	return 0
}

func (cb *ContextBuilder) identifyDataSources(analysis *AnalysisResult) []string {
	sources := []string{"file_system", "git_metadata"}

	if len(analysis.ProjectInfo.ConfigFiles) > 0 {
		sources = append(sources, "configuration_files")
	}
	if analysis.Dependencies != nil {
		sources = append(sources, "dependency_manifests")
	}

	return sources
}

func (cb *ContextBuilder) calculateConfidence(analysis *AnalysisResult) float64 {
	// Calculate confidence based on available data
	confidence := 0.5 // Base confidence

	if analysis.ProjectInfo != nil {
		confidence += 0.2
	}
	if len(analysis.Languages) > 0 {
		confidence += 0.1
	}
	if len(analysis.Frameworks) > 0 {
		confidence += 0.1
	}
	if analysis.Dependencies != nil {
		confidence += 0.1
	}

	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

func (cb *ContextBuilder) calculateCompleteness(analysis *AnalysisResult) float64 {
	// Calculate completeness based on expected vs actual data
	expectedComponents := 7.0 // ProjectInfo, Languages, Frameworks, Dependencies, FileStructure, Conventions, Metadata
	actualComponents := 0.0

	if analysis.ProjectInfo != nil {
		actualComponents++
	}
	if len(analysis.Languages) > 0 {
		actualComponents++
	}
	if len(analysis.Frameworks) > 0 {
		actualComponents++
	}
	if analysis.Dependencies != nil {
		actualComponents++
	}
	if analysis.FileStructure != nil {
		actualComponents++
	}
	if analysis.Conventions != nil {
		actualComponents++
	}
	if analysis.Metadata != nil {
		actualComponents++
	}

	return actualComponents / expectedComponents
}

// Issue-specific helper methods

func (cb *ContextBuilder) estimateComplexity(issueData *types.IssueData) string {
	// Simple complexity estimation based on labels and description length
	for _, label := range issueData.Labels {
		switch strings.ToLower(label) {
		case "critical", "major", "breaking":
			return complexityHigh
		case "simple", "trivial", "documentation":
			return complexityLow
		}
	}

	// Estimate based on description length
	if len(issueData.Body) > 1000 {
		return complexityHigh
	} else if len(issueData.Body) > 300 {
		return "medium"
	}

	return complexityLow
}

func (cb *ContextBuilder) estimateEffort(issueData *types.IssueData) time.Duration {
	complexity := cb.estimateComplexity(issueData)

	switch complexity {
	case "low":
		return 2 * time.Hour
	case "medium":
		return 8 * time.Hour
	case "high":
		return 24 * time.Hour
	default:
		return 4 * time.Hour
	}
}

func (cb *ContextBuilder) identifyRiskFactors(issueData *types.IssueData, analysis *AnalysisResult) []string {
	var risks []string

	// Check for risky labels
	for _, label := range issueData.Labels {
		switch strings.ToLower(label) {
		case "breaking", "breaking-change":
			risks = append(risks, "Breaking change")
		case "security":
			risks = append(risks, "Security implications")
		case "performance":
			risks = append(risks, "Performance impact")
		case "database", "migration":
			risks = append(risks, "Database changes")
		}
	}

	// Check for risky keywords in description
	riskKeywords := []string{"delete", "remove", "breaking", "migration", "schema", "security"}
	for _, keyword := range riskKeywords {
		if strings.Contains(strings.ToLower(issueData.Body), keyword) {
			risks = append(risks, fmt.Sprintf("Mentions %s", keyword))
		}
	}

	return risks
}

func (cb *ContextBuilder) suggestTestStrategy(issueData *types.IssueData, analysis *AnalysisResult) string {
	// Suggest test strategy based on issue type and project
	for _, label := range issueData.Labels {
		switch strings.ToLower(label) {
		case "api", "backend":
			return "API integration tests + unit tests"
		case "frontend", "ui":
			return "Component tests + E2E tests"
		case "bug", "fix":
			return "Regression tests + existing test updates"
		case "feature", "enhancement":
			return "Comprehensive unit tests + integration tests"
		}
	}

	return "Unit tests + integration tests"
}

func (cb *ContextBuilder) generateImplementationHints(issueData *types.IssueData, analysis *AnalysisResult) []string {
	var hints []string

	// Generate hints based on project type and frameworks
	if analysis.ProjectInfo.ProjectType == ProjectTypeWebApp {
		hints = append(hints, "Consider responsive design")
		hints = append(hints, "Ensure accessibility compliance")
	}

	for _, framework := range analysis.Frameworks {
		switch framework.Name {
		case "React":
			hints = append(hints, "Use functional components and hooks")
			hints = append(hints, "Consider state management needs")
		case "Django":
			hints = append(hints, "Follow Django conventions")
			hints = append(hints, "Consider database migrations")
		case "Express.js":
			hints = append(hints, "Implement proper error handling")
			hints = append(hints, "Consider middleware patterns")
		}
	}

	return hints
}

func (cb *ContextBuilder) filterRelevantFilesForIssue(files []types.RelevantFileInfo, issueContext *types.IssueContextInfo) []types.RelevantFileInfo {
	if issueContext == nil {
		return files
	}

	// Boost importance of mentioned files
	mentionedFilesMap := make(map[string]bool)
	for _, file := range issueContext.MentionedFiles {
		mentionedFilesMap[file] = true
	}

	for i := range files {
		if mentionedFilesMap[files[i].Path] {
			files[i].Importance += 0.3
		}
	}

	// Re-sort by importance
	sort.Slice(files, func(i, j int) bool {
		return files[i].Importance > files[j].Importance
	})

	return files
}

// Utility functions

func (cb *ContextBuilder) isDatabaseDependency(name string) bool {
	dbDeps := []string{"mysql", "postgres", "mongodb", "redis", "sqlite"}
	for _, db := range dbDeps {
		if strings.Contains(strings.ToLower(name), db) {
			return true
		}
	}
	return false
}

func (cb *ContextBuilder) getDatabaseType(depName string) string {
	if strings.Contains(strings.ToLower(depName), "mysql") {
		return "MySQL"
	}
	if strings.Contains(strings.ToLower(depName), "postgres") {
		return "PostgreSQL"
	}
	if strings.Contains(strings.ToLower(depName), "mongodb") {
		return "MongoDB"
	}
	return unknownValue
}

func (cb *ContextBuilder) identifyEnvironmentVariables(analysis *AnalysisResult) []string {
	// This would scan source files for environment variable usage
	return []string{}
}

func (cb *ContextBuilder) isDeploymentFile(filename string) bool {
	deploymentFiles := []string{"Dockerfile", "docker-compose", ".yml", "k8s", "deployment"}
	for _, file := range deploymentFiles {
		if strings.Contains(filename, file) {
			return true
		}
	}
	return false
}

func (cb *ContextBuilder) getDeploymentType(filename string) string {
	if strings.Contains(filename, "docker") {
		return "Docker"
	}
	if strings.Contains(filename, "k8s") || strings.Contains(filename, "kubernetes") {
		return "Kubernetes"
	}
	return unknownValue
}

func appendUnique(slice []string, item string) []string {
	for _, existing := range slice {
		if existing == item {
			return slice
		}
	}
	return append(slice, item)
}

func parseInt(s string) int {
	// Simple integer parsing, ignoring errors
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
