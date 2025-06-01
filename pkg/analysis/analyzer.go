// Package analysis provides comprehensive codebase analysis capabilities for ccAgents
package analysis

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/internal/types"
)

// Analyzer provides comprehensive code analysis and context building
type Analyzer struct {
	config             AnalyzerConfig
	projectDetector    *ProjectDetector
	languageDetector   *LanguageDetector
	frameworkDetector  *FrameworkDetector
	dependencyAnalyzer *DependencyAnalyzer
	contextBuilder     *ContextBuilder
	cache              *AnalysisCache
}

// AnalyzerConfig configures the code analyzer
type AnalyzerConfig struct {
	ProjectRoot        string
	MaxFileSize        int64
	MaxFilesToAnalyze  int
	IgnorePatterns     []string
	IncludeHiddenFiles bool
	CacheEnabled       bool
	CacheExpiration    time.Duration
	ParallelAnalysis   bool
	MaxWorkers         int
}

// AnalysisResult contains comprehensive analysis results
type AnalysisResult struct {
	ProjectInfo   *ProjectInfo       `json:"project_info"`
	Languages     []LanguageInfo     `json:"languages"`
	Frameworks    []FrameworkInfo    `json:"frameworks"`
	Dependencies  *DependencyInfo    `json:"dependencies"`
	CodeContext   *types.CodeContext `json:"code_context"`
	FileStructure *FileStructure     `json:"file_structure"`
	Conventions   *CodingConventions `json:"conventions"`
	Metadata      *ProjectMetadata   `json:"metadata"`
	IssueContext  *IssueContext      `json:"issue_context,omitempty"`
	AnalysisTime  time.Duration      `json:"analysis_time"`
	Timestamp     time.Time          `json:"timestamp"`
}

// ProjectInfo contains basic project information
type ProjectInfo struct {
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	Version           string                 `json:"version"`
	License           string                 `json:"license"`
	Repository        string                 `json:"repository"`
	Homepage          string                 `json:"homepage"`
	MainLanguage      string                 `json:"main_language"`
	ProjectType       ProjectType            `json:"project_type"`
	BuildSystem       string                 `json:"build_system"`
	PackageManager    string                 `json:"package_manager"`
	ConfigFiles       []string               `json:"config_files"`
	EntryPoints       []string               `json:"entry_points"`
	SourceDirs        []string               `json:"source_dirs"`
	TestDirs          []string               `json:"test_dirs"`
	DocumentationDirs []string               `json:"documentation_dirs"`
	Tags              []string               `json:"tags"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// LanguageInfo contains language-specific information
type LanguageInfo struct {
	Name         string               `json:"name"`
	Version      string               `json:"version"`
	FileCount    int                  `json:"file_count"`
	LineCount    int                  `json:"line_count"`
	Percentage   float64              `json:"percentage"`
	Extensions   []string             `json:"extensions"`
	ConfigFiles  []string             `json:"config_files"`
	ToolingFiles []string             `json:"tooling_files"`
	Conventions  *LanguageConventions `json:"conventions"`
}

// FrameworkInfo contains framework-specific information
type FrameworkInfo struct {
	Name         string                `json:"name"`
	Version      string                `json:"version"`
	Type         FrameworkType         `json:"type"`
	ConfigFiles  []string              `json:"config_files"`
	Dependencies []string              `json:"dependencies"`
	Patterns     []string              `json:"patterns"`
	Conventions  *FrameworkConventions `json:"conventions"`
}

// DependencyInfo contains dependency analysis results
type DependencyInfo struct {
	Direct           []Dependency         `json:"direct"`
	Transitive       []Dependency         `json:"transitive"`
	DevDeps          []Dependency         `json:"dev_dependencies"`
	PeerDeps         []Dependency         `json:"peer_dependencies"`
	Conflicts        []DependencyConflict `json:"conflicts"`
	Security         []SecurityIssue      `json:"security_issues"`
	Licenses         map[string]int       `json:"licenses"`
	UpdatesAvailable []DependencyUpdate   `json:"updates_available"`
}

// FileStructure represents the project's file organization
type FileStructure struct {
	TotalFiles     int            `json:"total_files"`
	TotalLines     int            `json:"total_lines"`
	TotalSize      int64          `json:"total_size"`
	DirectoryTree  *DirectoryNode `json:"directory_tree"`
	FilesByType    map[string]int `json:"files_by_type"`
	LargestFiles   []FileInfo     `json:"largest_files"`
	RecentFiles    []FileInfo     `json:"recent_files"`
	ImportantFiles []FileInfo     `json:"important_files"`
}

// CodingConventions represents discovered coding standards
type CodingConventions struct {
	IndentationStyle  string                 `json:"indentation_style"`
	IndentationSize   int                    `json:"indentation_size"`
	LineEndings       string                 `json:"line_endings"`
	QuoteStyle        string                 `json:"quote_style"`
	NamingConventions map[string]string      `json:"naming_conventions"`
	FileNaming        string                 `json:"file_naming"`
	DirectoryNaming   string                 `json:"directory_naming"`
	CommentStyle      []string               `json:"comment_style"`
	FormattingRules   map[string]interface{} `json:"formatting_rules"`
}

// ProjectMetadata contains extracted metadata
type ProjectMetadata struct {
	CreationDate  time.Time           `json:"creation_date"`
	LastModified  time.Time           `json:"last_modified"`
	Contributors  []Contributor       `json:"contributors"`
	CommitCount   int                 `json:"commit_count"`
	BranchCount   int                 `json:"branch_count"`
	TagCount      int                 `json:"tag_count"`
	IssueCount    int                 `json:"issue_count"`
	PRCount       int                 `json:"pr_count"`
	Stars         int                 `json:"stars"`
	Forks         int                 `json:"forks"`
	Watchers      int                 `json:"watchers"`
	Topics        []string            `json:"topics"`
	README        *READMEInfo         `json:"readme"`
	Documentation []DocumentationFile `json:"documentation"`
}

// IssueContext contains issue-specific analysis context
type IssueContext struct {
	IssueNumber     int             `json:"issue_number"`
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	Labels          []string        `json:"labels"`
	RelatedFiles    []string        `json:"related_files"`
	AffectedModules []string        `json:"affected_modules"`
	Complexity      ComplexityLevel `json:"complexity"`
	EstimatedEffort time.Duration   `json:"estimated_effort"`
	Prerequisites   []string        `json:"prerequisites"`
	TestStrategy    string          `json:"test_strategy"`
	RiskFactors     []string        `json:"risk_factors"`
}

// Supporting types

type ProjectType int

const (
	ProjectTypeUnknown ProjectType = iota
	ProjectTypeLibrary
	ProjectTypeApplication
	ProjectTypeWebApp
	ProjectTypeMicroservice
	ProjectTypeCLI
	ProjectTypeFramework
	ProjectTypeAPI
	ProjectTypeMobile
	ProjectTypeDesktop
	ProjectTypeGame
	ProjectTypeScript
	ProjectTypeDocumentation
)

type FrameworkType int

const (
	FrameworkTypeUnknown FrameworkType = iota
	FrameworkTypeWeb
	FrameworkTypeMobile
	FrameworkTypeDesktop
	FrameworkTypeAPI
	FrameworkTypeORM
	FrameworkTypeTesting
	FrameworkTypeBuild
	FrameworkTypeLogging
	FrameworkTypeAuth
	FrameworkTypeUI
)

type ComplexityLevel int

const (
	ComplexityLow ComplexityLevel = iota
	ComplexityMedium
	ComplexityHigh
	ComplexityCritical
)

// String constants for common values
const (
	typeAPI      = "api"
	complexityLow = "low"
	complexityMedium = "medium"
	complexityHigh = "high"
	unknownType = "unknown"
)

type Dependency struct {
	Name            string                  `json:"name"`
	Version         string                  `json:"version"`
	Type            DependencyType          `json:"type"`
	Source          string                  `json:"source"`
	License         string                  `json:"license"`
	Description     string                  `json:"description"`
	Homepage        string                  `json:"homepage"`
	Repository      string                  `json:"repository"`
	Deprecated      bool                    `json:"deprecated"`
	Vulnerabilities []SecurityVulnerability `json:"vulnerabilities"`
}

type DependencyType int

const (
	DependencyTypeDirect DependencyType = iota
	DependencyTypeTransitive
	DependencyTypeDev
	DependencyTypePeer
	DependencyTypeOptional
	DependencyTypeBundled
)

type DependencyConflict struct {
	Package    string   `json:"package"`
	Versions   []string `json:"versions"`
	Reason     string   `json:"reason"`
	Severity   string   `json:"severity"`
	Resolution string   `json:"resolution"`
}

type SecurityIssue struct {
	Type        string   `json:"type"`
	Severity    string   `json:"severity"`
	Package     string   `json:"package"`
	Version     string   `json:"version"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	CVE         string   `json:"cve,omitempty"`
	References  []string `json:"references"`
	FixVersion  string   `json:"fix_version,omitempty"`
}

type SecurityVulnerability struct {
	ID          string   `json:"id"`
	Severity    string   `json:"severity"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	CVSSScore   float64  `json:"cvss_score"`
	References  []string `json:"references"`
	FixedIn     string   `json:"fixed_in,omitempty"`
}

type DependencyUpdate struct {
	Package        string `json:"package"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	UpdateType     string `json:"update_type"`
	ChangelogURL   string `json:"changelog_url,omitempty"`
	Breaking       bool   `json:"breaking"`
}

type DirectoryNode struct {
	Name     string           `json:"name"`
	Path     string           `json:"path"`
	Type     string           `json:"type"`
	Size     int64            `json:"size"`
	Children []*DirectoryNode `json:"children,omitempty"`
	Files    []FileInfo       `json:"files,omitempty"`
}

type FileInfo struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	Extension    string    `json:"extension"`
	Language     string    `json:"language"`
	LineCount    int       `json:"line_count"`
	LastModified time.Time `json:"last_modified"`
	Importance   float64   `json:"importance"`
	Type         FileType  `json:"type"`
}

type FileType int

const (
	FileTypeSource FileType = iota
	FileTypeTest
	FileTypeConfig
	FileTypeDocumentation
	FileTypeBuild
	FileTypeAsset
	FileTypeOther
)

type LanguageConventions struct {
	NamingStyle        string   `json:"naming_style"`
	FileStructure      string   `json:"file_structure"`
	ImportStyle        string   `json:"import_style"`
	ErrorHandling      string   `json:"error_handling"`
	TestingPattern     string   `json:"testing_pattern"`
	DocumentationStyle string   `json:"documentation_style"`
	CommonPatterns     []string `json:"common_patterns"`
}

type FrameworkConventions struct {
	DirectoryStructure []string          `json:"directory_structure"`
	NamingPatterns     map[string]string `json:"naming_patterns"`
	ConfigurationStyle string            `json:"configuration_style"`
	RoutingPattern     string            `json:"routing_pattern,omitempty"`
	ComponentStructure string            `json:"component_structure,omitempty"`
	StateManagement    string            `json:"state_management,omitempty"`
	TestingStrategy    string            `json:"testing_strategy"`
}

type Contributor struct {
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Commits      int       `json:"commits"`
	Additions    int       `json:"additions"`
	Deletions    int       `json:"deletions"`
	LastActivity time.Time `json:"last_activity"`
}

type READMEInfo struct {
	Path      string   `json:"path"`
	Format    string   `json:"format"`
	Sections  []string `json:"sections"`
	HasBadges bool     `json:"has_badges"`
	HasTOC    bool     `json:"has_toc"`
	WordCount int      `json:"word_count"`
	LineCount int      `json:"line_count"`
	Quality   float64  `json:"quality"`
}

type DocumentationFile struct {
	Path        string    `json:"path"`
	Type        string    `json:"type"`
	Format      string    `json:"format"`
	Topics      []string  `json:"topics"`
	WordCount   int       `json:"word_count"`
	LastUpdated time.Time `json:"last_updated"`
}

// Default configuration values
const (
	DefaultMaxFileSize       = 10 * 1024 * 1024 // 10MB
	DefaultMaxFilesToAnalyze = 10000
	DefaultCacheExpiration   = 24 * time.Hour
	DefaultMaxWorkers        = 10
)

// NewAnalyzer creates a new code analyzer
func NewAnalyzer(config AnalyzerConfig) (*Analyzer, error) {
	// Set defaults
	if config.MaxFileSize == 0 {
		config.MaxFileSize = DefaultMaxFileSize
	}
	if config.MaxFilesToAnalyze == 0 {
		config.MaxFilesToAnalyze = DefaultMaxFilesToAnalyze
	}
	if config.CacheExpiration == 0 {
		config.CacheExpiration = DefaultCacheExpiration
	}
	if config.MaxWorkers == 0 {
		config.MaxWorkers = DefaultMaxWorkers
	}
	if len(config.IgnorePatterns) == 0 {
		config.IgnorePatterns = getDefaultIgnorePatterns()
	}

	analyzer := &Analyzer{
		config:             config,
		projectDetector:    NewProjectDetector(config),
		languageDetector:   NewLanguageDetector(config),
		frameworkDetector:  NewFrameworkDetector(config),
		dependencyAnalyzer: NewDependencyAnalyzer(config),
		contextBuilder:     NewContextBuilder(config),
	}

	if config.CacheEnabled {
		analyzer.cache = NewAnalysisCache(config.CacheExpiration)
	}

	return analyzer, nil
}

// AnalyzeProject performs comprehensive project analysis
func (a *Analyzer) AnalyzeProject(ctx context.Context, projectRoot string) (*AnalysisResult, error) {
	startTime := time.Now()

	// Validate project root and check cache
	if err := a.validateProjectRoot(projectRoot); err != nil {
		return nil, err
	}

	if cached := a.getCachedResult(projectRoot); cached != nil {
		return cached, nil
	}

	// Initialize result
	result := &AnalysisResult{Timestamp: startTime}

	// Run concurrent analysis phases
	errors := a.runConcurrentPhases(ctx, projectRoot, result)
	if len(errors) > 0 {
		return nil, fmt.Errorf("analysis failed with %d errors: %v", len(errors), errors)
	}

	// Run sequential analysis phases
	errors = a.runSequentialPhases(ctx, projectRoot, result)

	// Finalize result
	a.finalizeResult(result, startTime, projectRoot, errors)

	return result, nil
}

// validateProjectRoot checks if the project root exists
func (a *Analyzer) validateProjectRoot(projectRoot string) error {
	if _, err := os.Stat(projectRoot); os.IsNotExist(err) {
		return fmt.Errorf("project root does not exist: %s", projectRoot)
	}
	return nil
}

// getCachedResult retrieves cached analysis result if available
func (a *Analyzer) getCachedResult(projectRoot string) *AnalysisResult {
	if a.cache != nil {
		return a.cache.Get(projectRoot)
	}
	return nil
}

// runConcurrentPhases executes analysis phases that can run in parallel
func (a *Analyzer) runConcurrentPhases(ctx context.Context, projectRoot string, result *AnalysisResult) []error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)

	// Phase 1: Project structure detection
	wg.Add(1)
	go a.detectProjectInfo(ctx, projectRoot, result, &wg, &mu, &errors)

	// Phase 2: Language detection
	wg.Add(1)
	go a.detectLanguages(ctx, projectRoot, result, &wg, &mu, &errors)

	// Phase 3: Framework detection
	wg.Add(1)
	go a.detectFrameworks(ctx, projectRoot, result, &wg, &mu, &errors)

	wg.Wait()
	return errors
}

// runSequentialPhases executes analysis phases that depend on previous results
func (a *Analyzer) runSequentialPhases(ctx context.Context, projectRoot string, result *AnalysisResult) []error {
	var errors []error

	// Phase 4: Dependency analysis (needs project info)
	errors = a.analyzeDependencies(ctx, projectRoot, result, errors)

	// Phase 5: File structure analysis
	errors = a.analyzeFileStructurePhase(ctx, projectRoot, result, errors)

	// Phase 6: Coding conventions
	errors = a.analyzeCodingConventionsPhase(ctx, projectRoot, result, errors)

	// Phase 7: Project metadata
	errors = a.extractProjectMetadataPhase(ctx, projectRoot, result, errors)

	// Phase 8: Build code context
	errors = a.buildCodeContextPhase(ctx, result, errors)

	return errors
}

// Helper methods for concurrent phases
func (a *Analyzer) detectProjectInfo(ctx context.Context, projectRoot string, result *AnalysisResult, wg *sync.WaitGroup, mu *sync.Mutex, errors *[]error) {
	defer wg.Done()
	if projectInfo, err := a.projectDetector.DetectProjectInfo(ctx, projectRoot); err != nil {
		mu.Lock()
		*errors = append(*errors, fmt.Errorf("project detection failed: %w", err))
		mu.Unlock()
	} else {
		mu.Lock()
		result.ProjectInfo = projectInfo
		mu.Unlock()
	}
}

func (a *Analyzer) detectLanguages(ctx context.Context, projectRoot string, result *AnalysisResult, wg *sync.WaitGroup, mu *sync.Mutex, errors *[]error) {
	defer wg.Done()
	if languages, err := a.languageDetector.DetectLanguages(ctx, projectRoot); err != nil {
		mu.Lock()
		*errors = append(*errors, fmt.Errorf("language detection failed: %w", err))
		mu.Unlock()
	} else {
		mu.Lock()
		result.Languages = languages
		mu.Unlock()
	}
}

func (a *Analyzer) detectFrameworks(ctx context.Context, projectRoot string, result *AnalysisResult, wg *sync.WaitGroup, mu *sync.Mutex, errors *[]error) {
	defer wg.Done()
	if frameworks, err := a.frameworkDetector.DetectFrameworks(ctx, projectRoot); err != nil {
		mu.Lock()
		*errors = append(*errors, fmt.Errorf("framework detection failed: %w", err))
		mu.Unlock()
	} else {
		mu.Lock()
		result.Frameworks = frameworks
		mu.Unlock()
	}
}

// Helper methods for sequential phases
func (a *Analyzer) analyzeDependencies(ctx context.Context, projectRoot string, result *AnalysisResult, errors []error) []error {
	if dependencyInfo, err := a.dependencyAnalyzer.AnalyzeDependencies(ctx, projectRoot, result.ProjectInfo); err != nil {
		errors = append(errors, fmt.Errorf("dependency analysis failed: %w", err))
	} else {
		result.Dependencies = dependencyInfo
	}
	return errors
}

func (a *Analyzer) analyzeFileStructurePhase(ctx context.Context, projectRoot string, result *AnalysisResult, errors []error) []error {
	if fileStructure, err := a.analyzeFileStructure(ctx, projectRoot); err != nil {
		errors = append(errors, fmt.Errorf("file structure analysis failed: %w", err))
	} else {
		result.FileStructure = fileStructure
	}
	return errors
}

func (a *Analyzer) analyzeCodingConventionsPhase(ctx context.Context, projectRoot string, result *AnalysisResult, errors []error) []error {
	if conventions, err := a.analyzeCodingConventions(ctx, projectRoot, result.Languages); err != nil {
		errors = append(errors, fmt.Errorf("conventions analysis failed: %w", err))
	} else {
		result.Conventions = conventions
	}
	return errors
}

func (a *Analyzer) extractProjectMetadataPhase(ctx context.Context, projectRoot string, result *AnalysisResult, errors []error) []error {
	if metadata, err := a.extractProjectMetadata(ctx, projectRoot); err != nil {
		errors = append(errors, fmt.Errorf("metadata extraction failed: %w", err))
	} else {
		result.Metadata = metadata
	}
	return errors
}

func (a *Analyzer) buildCodeContextPhase(ctx context.Context, result *AnalysisResult, errors []error) []error {
	if codeContext, err := a.contextBuilder.BuildContext(ctx, result); err != nil {
		errors = append(errors, fmt.Errorf("context building failed: %w", err))
	} else {
		result.CodeContext = codeContext
	}
	return errors
}

// finalizeResult completes the analysis result
func (a *Analyzer) finalizeResult(result *AnalysisResult, startTime time.Time, projectRoot string, errors []error) {
	result.AnalysisTime = time.Since(startTime)

	// Cache result if caching is enabled and no errors
	if a.cache != nil && len(errors) == 0 {
		a.cache.Set(projectRoot, result)
	}

	// Log warnings but don't fail the analysis
	if len(errors) > 0 {
		// TODO: Add proper logging of warnings
		_ = errors // Acknowledge that we have errors but continue
	}
}

// AnalyzeForIssue performs issue-specific analysis
func (a *Analyzer) AnalyzeForIssue(ctx context.Context, projectRoot string, issueData *types.IssueData) (*AnalysisResult, error) {
	// First perform general project analysis
	result, err := a.AnalyzeProject(ctx, projectRoot)
	if err != nil {
		return nil, fmt.Errorf("project analysis failed: %w", err)
	}

	// Add issue-specific context
	issueContext, err := a.analyzeIssueContext(ctx, projectRoot, issueData, result)
	if err != nil {
		return result, fmt.Errorf("issue context analysis failed: %w", err)
	}

	result.IssueContext = issueContext
	return result, nil
}

// Helper methods

func (a *Analyzer) analyzeFileStructure(ctx context.Context, projectRoot string) (*FileStructure, error) {
	// Implementation for file structure analysis
	return &FileStructure{
		TotalFiles: 0,
		TotalLines: 0,
		TotalSize:  0,
		DirectoryTree: &DirectoryNode{
			Name: filepath.Base(projectRoot),
			Path: projectRoot,
			Type: "directory",
		},
		FilesByType:    make(map[string]int),
		LargestFiles:   []FileInfo{},
		RecentFiles:    []FileInfo{},
		ImportantFiles: []FileInfo{},
	}, nil
}

func (a *Analyzer) analyzeCodingConventions(ctx context.Context, projectRoot string, languages []LanguageInfo) (*CodingConventions, error) {
	// Implementation for coding conventions analysis
	return &CodingConventions{
		IndentationStyle:  "spaces",
		IndentationSize:   2,
		LineEndings:       "lf",
		QuoteStyle:        "double",
		NamingConventions: make(map[string]string),
		FileNaming:        "kebab-case",
		DirectoryNaming:   "kebab-case",
		CommentStyle:      []string{"//", "/* */"},
		FormattingRules:   make(map[string]interface{}),
	}, nil
}

func (a *Analyzer) extractProjectMetadata(ctx context.Context, projectRoot string) (*ProjectMetadata, error) {
	// Implementation for project metadata extraction
	return &ProjectMetadata{
		CreationDate:  time.Now(),
		LastModified:  time.Now(),
		Contributors:  []Contributor{},
		CommitCount:   0,
		BranchCount:   0,
		TagCount:      0,
		IssueCount:    0,
		PRCount:       0,
		Stars:         0,
		Forks:         0,
		Watchers:      0,
		Topics:        []string{},
		Documentation: []DocumentationFile{},
	}, nil
}

func (a *Analyzer) analyzeIssueContext(ctx context.Context, projectRoot string, issueData *types.IssueData, projectResult *AnalysisResult) (*IssueContext, error) {
	// Implementation for issue-specific context analysis
	return &IssueContext{
		IssueNumber:     issueData.Number,
		Title:           issueData.Title,
		Description:     issueData.Body,
		Labels:          issueData.Labels,
		RelatedFiles:    []string{},
		AffectedModules: []string{},
		Complexity:      ComplexityMedium,
		EstimatedEffort: 2 * time.Hour,
		Prerequisites:   []string{},
		TestStrategy:    "unit-tests",
		RiskFactors:     []string{},
	}, nil
}

func getDefaultIgnorePatterns() []string {
	return []string{
		"node_modules/**",
		".git/**",
		".svn/**",
		".hg/**",
		"*.log",
		"*.tmp",
		"*.temp",
		".DS_Store",
		"Thumbs.db",
		"__pycache__/**",
		"*.pyc",
		"*.pyo",
		"target/**",
		"build/**",
		"dist/**",
		"out/**",
		"*.class",
		"*.jar",
		"*.war",
		"*.ear",
		"vendor/**",
		"coverage/**",
		".nyc_output/**",
		"*.min.js",
		"*.min.css",
		"*.map",
	}
}

func (pt ProjectType) String() string {
	switch pt {
	case ProjectTypeLibrary:
		return "library"
	case ProjectTypeApplication:
		return "application"
	case ProjectTypeWebApp:
		return "web-app"
	case ProjectTypeMicroservice:
		return "microservice"
	case ProjectTypeCLI:
		return "cli"
	case ProjectTypeFramework:
		return "framework"
	case ProjectTypeAPI:
		return typeAPI
	case ProjectTypeMobile:
		return "mobile"
	case ProjectTypeDesktop:
		return "desktop"
	case ProjectTypeGame:
		return "game"
	case ProjectTypeScript:
		return "script"
	case ProjectTypeDocumentation:
		return "documentation"
	default:
		return unknownType
	}
}

func (ft FrameworkType) String() string {
	switch ft {
	case FrameworkTypeWeb:
		return "web"
	case FrameworkTypeMobile:
		return "mobile"
	case FrameworkTypeDesktop:
		return "desktop"
	case FrameworkTypeAPI:
		return typeAPI
	case FrameworkTypeORM:
		return "orm"
	case FrameworkTypeTesting:
		return "testing"
	case FrameworkTypeBuild:
		return "build"
	case FrameworkTypeLogging:
		return "logging"
	case FrameworkTypeAuth:
		return "auth"
	case FrameworkTypeUI:
		return "ui"
	default:
		return unknownType
	}
}

func (cl ComplexityLevel) String() string {
	switch cl {
	case ComplexityLow:
		return complexityLow
	case ComplexityMedium:
		return complexityMedium
	case ComplexityHigh:
		return complexityHigh
	case ComplexityCritical:
		return "critical"
	default:
		return unknownType
	}
}
