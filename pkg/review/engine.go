package review

import (
	"context"
	"fmt"

	"strings"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/pkg/analysis"
)

// Severity constants
const (
	severityCritical = "critical"
	severityHigh     = "high"
	severityMedium   = "medium"
	severityLow      = "low"
	severityUnknown  = "unknown"
)

// ReviewEngine provides comprehensive code review capabilities
type ReviewEngine struct {
	config          ReviewConfig
	staticAnalyzer  *StaticAnalyzer
	securityScanner *SecurityScanner
	qualityAnalyzer *QualityAnalyzer
	aiReviewer      *AIReviewer
	mutex           sync.RWMutex
}

// ReviewConfig configures the review engine
type ReviewConfig struct {
	EnableStaticAnalysis bool
	EnableSecurityScan   bool
	EnableQualityCheck   bool
	EnableAIReview       bool
	MaxReviewIterations  int
	ParallelReviews      bool
	MaxWorkers           int
	ReviewTimeout        time.Duration
	MinQualityScore      float64
	SecurityLevel        SecurityLevel
	IgnorePatterns       []string
}

// ReviewResult contains comprehensive review results
type ReviewResult struct {
	OverallScore     float64               `json:"overall_score"`
	Status           ReviewStatus          `json:"status"`
	Issues           []ReviewIssue         `json:"issues"`
	Suggestions      []ReviewSuggestion    `json:"suggestions"`
	SecurityFindings []SecurityFinding     `json:"security_findings"`
	QualityMetrics   *QualityMetrics       `json:"quality_metrics"`
	StaticAnalysis   *StaticAnalysisResult `json:"static_analysis"`
	AIReviewResult   *AIReviewResult       `json:"ai_review"`
	FixedIssues      []FixedIssue          `json:"fixed_issues"`
	Iterations       int                   `json:"iterations"`
	ReviewTime       time.Duration         `json:"review_time"`
	Timestamp        time.Time             `json:"timestamp"`
}

// ReviewIssue represents a code review issue
type ReviewIssue struct {
	ID          string        `json:"id"`
	Type        IssueType     `json:"type"`
	Severity    IssueSeverity `json:"severity"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	File        string        `json:"file"`
	Line        int           `json:"line"`
	Column      int           `json:"column"`
	Rule        string        `json:"rule"`
	Category    string        `json:"category"`
	Fixable     bool          `json:"fixable"`
	Suggestion  string        `json:"suggestion,omitempty"`
	Context     []string      `json:"context"`
	Tags        []string      `json:"tags"`
}

// ReviewSuggestion represents an improvement suggestion
type ReviewSuggestion struct {
	ID          string             `json:"id"`
	Type        SuggestionType     `json:"type"`
	Priority    SuggestionPriority `json:"priority"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	File        string             `json:"file,omitempty"`
	Line        int                `json:"line,omitempty"`
	Before      string             `json:"before,omitempty"`
	After       string             `json:"after,omitempty"`
	Rationale   string             `json:"rationale"`
	Impact      string             `json:"impact"`
	Effort      string             `json:"effort"`
}

// SecurityFinding represents a security issue
type SecurityFinding struct {
	ID          string           `json:"id"`
	Severity    SecuritySeverity `json:"severity"`
	Type        string           `json:"type"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	File        string           `json:"file"`
	Line        int              `json:"line"`
	CWE         string           `json:"cwe,omitempty"`
	OWASP       string           `json:"owasp,omitempty"`
	Confidence  float64          `json:"confidence"`
	Impact      string           `json:"impact"`
	Remediation string           `json:"remediation"`
	References  []string         `json:"references"`
}

// QualityMetrics contains code quality measurements
type QualityMetrics struct {
	OverallScore    float64             `json:"overall_score"`
	Maintainability float64             `json:"maintainability"`
	Readability     float64             `json:"readability"`
	Complexity      *ComplexityMetrics  `json:"complexity"`
	Coverage        *CoverageMetrics    `json:"coverage"`
	Duplication     *DuplicationMetrics `json:"duplication"`
	Documentation   float64             `json:"documentation"`
	TestQuality     float64             `json:"test_quality"`
	Performance     float64             `json:"performance"`
	TechnicalDebt   time.Duration       `json:"technical_debt"`
}

// ComplexityMetrics contains complexity measurements
type ComplexityMetrics struct {
	Cyclomatic    float64 `json:"cyclomatic"`
	Cognitive     float64 `json:"cognitive"`
	Halstead      float64 `json:"halstead"`
	LinesOfCode   int     `json:"lines_of_code"`
	Functions     int     `json:"functions"`
	Classes       int     `json:"classes"`
	AverageMethod float64 `json:"average_method"`
}

// CoverageMetrics contains test coverage information
type CoverageMetrics struct {
	Line      float64  `json:"line"`
	Branch    float64  `json:"branch"`
	Function  float64  `json:"function"`
	Statement float64  `json:"statement"`
	Uncovered []string `json:"uncovered"`
}

// DuplicationMetrics contains code duplication information
type DuplicationMetrics struct {
	Percentage   float64            `json:"percentage"`
	Lines        int                `json:"lines"`
	Blocks       int                `json:"blocks"`
	Files        int                `json:"files"`
	Duplications []DuplicationBlock `json:"duplications"`
}

// DuplicationBlock represents a duplicated code block
type DuplicationBlock struct {
	Files     []string `json:"files"`
	Lines     int      `json:"lines"`
	Tokens    int      `json:"tokens"`
	StartLine int      `json:"start_line"`
	EndLine   int      `json:"end_line"`
}

// StaticAnalysisResult contains static analysis results
type StaticAnalysisResult struct {
	LinterResults    []LinterResult    `json:"linter_results"`
	TypeCheckResults []TypeCheckResult `json:"type_check_results"`
	ImportAnalysis   *ImportAnalysis   `json:"import_analysis"`
	DeadCodeReport   *DeadCodeReport   `json:"dead_code_report"`
	PerformanceHints []PerformanceHint `json:"performance_hints"`
}

// LinterResult contains linter output
type LinterResult struct {
	Tool     string        `json:"tool"`
	Version  string        `json:"version"`
	Issues   []ReviewIssue `json:"issues"`
	Warnings []ReviewIssue `json:"warnings"`
	Errors   []ReviewIssue `json:"errors"`
	Stats    LinterStats   `json:"stats"`
}

// LinterStats contains linter statistics
type LinterStats struct {
	FilesScanned int `json:"files_scanned"`
	TotalIssues  int `json:"total_issues"`
	Errors       int `json:"errors"`
	Warnings     int `json:"warnings"`
	InfoMessages int `json:"info_messages"`
}

// TypeCheckResult contains type checking results
type TypeCheckResult struct {
	Tool     string        `json:"tool"`
	Issues   []ReviewIssue `json:"issues"`
	Coverage float64       `json:"coverage"`
}

// ImportAnalysis contains import/dependency analysis
type ImportAnalysis struct {
	UnusedImports   []string `json:"unused_imports"`
	CircularImports []string `json:"circular_imports"`
	MissingImports  []string `json:"missing_imports"`
	ShadowedImports []string `json:"shadowed_imports"`
}

// DeadCodeReport contains dead code analysis
type DeadCodeReport struct {
	UnusedFunctions []string `json:"unused_functions"`
	UnusedVariables []string `json:"unused_variables"`
	UnusedTypes     []string `json:"unused_types"`
	UnreachableCode []string `json:"unreachable_code"`
}

// PerformanceHint contains performance optimization suggestions
type PerformanceHint struct {
	Type        string `json:"type"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
	Impact      string `json:"impact"`
}

// AIReviewResult contains AI-powered review results
type AIReviewResult struct {
	OverallFeedback       string              `json:"overall_feedback"`
	Recommendations       []AIRecommendation  `json:"recommendations"`
	BestPractices         []BestPracticeIssue `json:"best_practices"`
	ArchitecturalFeedback string              `json:"architectural_feedback"`
	StyleFeedback         string              `json:"style_feedback"`
	TestingFeedback       string              `json:"testing_feedback"`
	SecurityFeedback      string              `json:"security_feedback"`
	PerformanceFeedback   string              `json:"performance_feedback"`
	Confidence            float64             `json:"confidence"`
}

// AIRecommendation contains AI-generated recommendations
type AIRecommendation struct {
	Type        string `json:"type"`
	Priority    string `json:"priority"`
	Description string `json:"description"`
	Rationale   string `json:"rationale"`
	Example     string `json:"example,omitempty"`
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
}

// BestPracticeIssue represents best practice violations
type BestPracticeIssue struct {
	Practice   string `json:"practice"`
	Violation  string `json:"violation"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	Suggestion string `json:"suggestion"`
	Reference  string `json:"reference,omitempty"`
}

// FixedIssue represents an automatically fixed issue
type FixedIssue struct {
	OriginalIssue ReviewIssue `json:"original_issue"`
	Fix           AppliedFix  `json:"fix"`
	Success       bool        `json:"success"`
	Error         string      `json:"error,omitempty"`
}

// AppliedFix represents a fix that was applied
type AppliedFix struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	File        string `json:"file"`
	Before      string `json:"before"`
	After       string `json:"after"`
	LineStart   int    `json:"line_start"`
	LineEnd     int    `json:"line_end"`
}

// Enums and constants

type ReviewStatus int

const (
	ReviewStatusPending ReviewStatus = iota
	ReviewStatusInProgress
	ReviewStatusCompleted
	ReviewStatusFailed
	ReviewStatusPartial
)

type IssueType int

const (
	IssueTypeSyntax IssueType = iota
	IssueTypeLogic
	IssueTypeSecurity
	IssueTypePerformance
	IssueTypeStyle
	IssueTypeMaintainability
	IssueTypeDocumentation
	IssueTypeTesting
	IssueTypeAccessibility
)

type IssueSeverity int

const (
	IssueSeverityInfo IssueSeverity = iota
	IssueSeverityLow
	IssueSeverityMedium
	IssueSeverityHigh
	IssueSeverityCritical
)

type SuggestionType int

const (
	SuggestionTypeRefactor SuggestionType = iota
	SuggestionTypeOptimization
	SuggestionTypeArchitecture
	SuggestionTypeSecurity
	SuggestionTypeStyle
	SuggestionTypeDocumentation
	SuggestionTypeTesting
)

type SuggestionPriority int

const (
	SuggestionPriorityLow SuggestionPriority = iota
	SuggestionPriorityMedium
	SuggestionPriorityHigh
	SuggestionPriorityCritical
)

type SecuritySeverity int

const (
	SecuritySeverityInfo SecuritySeverity = iota
	SecuritySeverityLow
	SecuritySeverityMedium
	SecuritySeverityHigh
	SecuritySeverityCritical
)

type SecurityLevel int

const (
	SecurityLevelBasic SecurityLevel = iota
	SecurityLevelStandard
	SecurityLevelStrict
	SecurityLevelPentesting
)

// NewReviewEngine creates a new review engine
func NewReviewEngine(config ReviewConfig) (*ReviewEngine, error) {
	// Set defaults
	if config.MaxReviewIterations == 0 {
		config.MaxReviewIterations = 3
	}
	if config.MaxWorkers == 0 {
		config.MaxWorkers = 4
	}
	if config.ReviewTimeout == 0 {
		config.ReviewTimeout = 30 * time.Minute
	}
	if config.MinQualityScore == 0 {
		config.MinQualityScore = 0.7
	}

	engine := &ReviewEngine{
		config: config,
		mutex:  sync.RWMutex{},
	}

	// Initialize sub-components
	if config.EnableStaticAnalysis {
		engine.staticAnalyzer = NewStaticAnalyzer(StaticAnalyzerConfig{
			IgnorePatterns: config.IgnorePatterns,
		})
	}

	if config.EnableSecurityScan {
		engine.securityScanner = NewSecurityScanner(SecurityScannerConfig{
			SecurityLevel: config.SecurityLevel,
		})
	}

	if config.EnableQualityCheck {
		engine.qualityAnalyzer = NewQualityAnalyzer(QualityAnalyzerConfig{
			MinScore: config.MinQualityScore,
		})
	}

	if config.EnableAIReview {
		engine.aiReviewer = NewAIReviewer(AIReviewerConfig{
			MaxIterations: config.MaxReviewIterations,
		})
	}

	return engine, nil
}

// ReviewChanges performs comprehensive code review on changes
func (re *ReviewEngine) ReviewChanges(ctx context.Context, changes []string, analysisResult *analysis.AnalysisResult) (*ReviewResult, error) {
	startTime := time.Now()

	result := &ReviewResult{
		Status:           ReviewStatusInProgress,
		Issues:           []ReviewIssue{},
		Suggestions:      []ReviewSuggestion{},
		SecurityFindings: []SecurityFinding{},
		FixedIssues:      []FixedIssue{},
		Timestamp:        startTime,
	}

	// Perform iterative review
	for iteration := 0; iteration < re.config.MaxReviewIterations; iteration++ {
		result.Iterations = iteration + 1

		// Run all review phases
		if err := re.runReviewPhase(ctx, changes, analysisResult, result); err != nil {
			result.Status = ReviewStatusFailed
			return result, fmt.Errorf("review phase %d failed: %w", iteration+1, err)
		}

		// Check if we've reached acceptable quality
		if result.OverallScore >= re.config.MinQualityScore && len(re.getCriticalIssues(result.Issues)) == 0 {
			break
		}

		// Apply automatic fixes if available
		if err := re.applyAutomaticFixes(ctx, result); err != nil {
			// Log error but continue
			fmt.Printf("Warning: Could not apply automatic fixes: %v\n", err)
		}
	}

	// Calculate final scores
	re.calculateFinalScores(result)

	result.ReviewTime = time.Since(startTime)
	result.Status = ReviewStatusCompleted

	return result, nil
}

// runReviewPhase executes a single review phase
func (re *ReviewEngine) runReviewPhase(ctx context.Context, changes []string, analysisResult *analysis.AnalysisResult, result *ReviewResult) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)

	// Static analysis
	if re.config.EnableStaticAnalysis && re.staticAnalyzer != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if staticResult, err := re.staticAnalyzer.Analyze(ctx, changes); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("static analysis failed: %w", err))
				mu.Unlock()
			} else {
				mu.Lock()
				result.StaticAnalysis = staticResult
				result.Issues = append(result.Issues, re.convertStaticAnalysisIssues(staticResult)...)
				mu.Unlock()
			}
		}()
	}

	// Security scanning
	if re.config.EnableSecurityScan && re.securityScanner != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if securityFindings, err := re.securityScanner.Scan(ctx, changes); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("security scan failed: %w", err))
				mu.Unlock()
			} else {
				mu.Lock()
				result.SecurityFindings = append(result.SecurityFindings, securityFindings...)
				mu.Unlock()
			}
		}()
	}

	// Quality analysis
	if re.config.EnableQualityCheck && re.qualityAnalyzer != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if qualityMetrics, err := re.qualityAnalyzer.Analyze(ctx, changes, analysisResult); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("quality analysis failed: %w", err))
				mu.Unlock()
			} else {
				mu.Lock()
				result.QualityMetrics = qualityMetrics
				mu.Unlock()
			}
		}()
	}

	// AI review
	if re.config.EnableAIReview && re.aiReviewer != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if aiResult, err := re.aiReviewer.Review(ctx, changes, analysisResult, result); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("AI review failed: %w", err))
				mu.Unlock()
			} else {
				mu.Lock()
				result.AIReviewResult = aiResult
				result.Suggestions = append(result.Suggestions, re.convertAIRecommendations(aiResult.Recommendations)...)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("review phase failed with %d errors: %v", len(errors), errors)
	}

	return nil
}

// Helper methods

func (re *ReviewEngine) getCriticalIssues(issues []ReviewIssue) []ReviewIssue {
	var critical []ReviewIssue
	for _, issue := range issues {
		if issue.Severity == IssueSeverityCritical {
			critical = append(critical, issue)
		}
	}
	return critical
}

func (re *ReviewEngine) applyAutomaticFixes(ctx context.Context, result *ReviewResult) error {
	// Apply automatic fixes for fixable issues
	for _, issue := range result.Issues {
		if issue.Fixable {
			if fix, err := re.generateFix(issue); err == nil {
				if appliedFix, err := re.applyFix(fix); err == nil {
					result.FixedIssues = append(result.FixedIssues, FixedIssue{
						OriginalIssue: issue,
						Fix:           appliedFix,
						Success:       true,
					})
				} else {
					result.FixedIssues = append(result.FixedIssues, FixedIssue{
						OriginalIssue: issue,
						Success:       false,
						Error:         err.Error(),
					})
				}
			}
		}
	}
	return nil
}

func (re *ReviewEngine) generateFix(issue ReviewIssue) (Fix, error) {
	// Generate appropriate fix based on issue type
	return Fix{}, nil // Placeholder
}

func (re *ReviewEngine) applyFix(fix Fix) (AppliedFix, error) {
	// Apply the fix to the codebase
	return AppliedFix{}, nil // Placeholder
}

func (re *ReviewEngine) convertStaticAnalysisIssues(result *StaticAnalysisResult) []ReviewIssue {
	var issues []ReviewIssue
	for _, linterResult := range result.LinterResults {
		issues = append(issues, linterResult.Issues...)
		issues = append(issues, linterResult.Warnings...)
		issues = append(issues, linterResult.Errors...)
	}
	return issues
}

func (re *ReviewEngine) convertAIRecommendations(recommendations []AIRecommendation) []ReviewSuggestion {
	var suggestions []ReviewSuggestion
	for _, rec := range recommendations {
		suggestion := ReviewSuggestion{
			ID:          generateID(),
			Type:        re.mapAIRecommendationType(rec.Type),
			Priority:    re.mapAIRecommendationPriority(rec.Priority),
			Title:       rec.Description,
			Description: rec.Rationale,
			File:        rec.File,
			Line:        rec.Line,
			Rationale:   rec.Rationale,
		}
		suggestions = append(suggestions, suggestion)
	}
	return suggestions
}

func (re *ReviewEngine) mapAIRecommendationType(aiType string) SuggestionType {
	switch strings.ToLower(aiType) {
	case "refactor":
		return SuggestionTypeRefactor
	case "optimization":
		return SuggestionTypeOptimization
	case "architecture":
		return SuggestionTypeArchitecture
	case "security":
		return SuggestionTypeSecurity
	case "style":
		return SuggestionTypeStyle
	case "documentation":
		return SuggestionTypeDocumentation
	case "testing":
		return SuggestionTypeTesting
	default:
		return SuggestionTypeRefactor
	}
}

func (re *ReviewEngine) mapAIRecommendationPriority(aiPriority string) SuggestionPriority {
	switch strings.ToLower(aiPriority) {
	case severityCritical:
		return SuggestionPriorityCritical
	case severityHigh:
		return SuggestionPriorityHigh
	case severityMedium:
		return SuggestionPriorityMedium
	case severityLow:
		return SuggestionPriorityLow
	default:
		return SuggestionPriorityMedium
	}
}

func (re *ReviewEngine) calculateFinalScores(result *ReviewResult) {
	// Calculate overall score based on various factors
	scores := []float64{}

	if result.QualityMetrics != nil {
		scores = append(scores, result.QualityMetrics.OverallScore)
	}

	if result.AIReviewResult != nil {
		scores = append(scores, result.AIReviewResult.Confidence)
	}

	// Factor in security findings
	securityScore := 1.0
	for _, finding := range result.SecurityFindings {
		switch finding.Severity {
		case SecuritySeverityCritical:
			securityScore -= 0.3
		case SecuritySeverityHigh:
			securityScore -= 0.2
		case SecuritySeverityMedium:
			securityScore -= 0.1
		case SecuritySeverityLow:
			securityScore -= 0.05
		}
	}
	if securityScore < 0 {
		securityScore = 0
	}
	scores = append(scores, securityScore)

	// Factor in critical issues
	criticalIssues := re.getCriticalIssues(result.Issues)
	issueScore := 1.0 - (float64(len(criticalIssues)) * 0.2)
	if issueScore < 0 {
		issueScore = 0
	}
	scores = append(scores, issueScore)

	// Calculate weighted average
	if len(scores) > 0 {
		sum := 0.0
		for _, score := range scores {
			sum += score
		}
		result.OverallScore = sum / float64(len(scores))
	}
}

// Utility functions

func generateID() string {
	// Generate a unique ID for issues/suggestions
	return fmt.Sprintf("review_%d", time.Now().UnixNano())
}

// Fix represents a code fix
type Fix struct {
	Type    string
	File    string
	Content string
	Line    int
}

// String methods for enums

func (rs ReviewStatus) String() string {
	switch rs {
	case ReviewStatusPending:
		return "pending"
	case ReviewStatusInProgress:
		return "in_progress"
	case ReviewStatusCompleted:
		return "completed"
	case ReviewStatusFailed:
		return "failed"
	case ReviewStatusPartial:
		return "partial"
	default:
		return severityUnknown
	}
}

func (it IssueType) String() string {
	switch it {
	case IssueTypeSyntax:
		return "syntax"
	case IssueTypeLogic:
		return "logic"
	case IssueTypeSecurity:
		return "security"
	case IssueTypePerformance:
		return "performance"
	case IssueTypeStyle:
		return "style"
	case IssueTypeMaintainability:
		return "maintainability"
	case IssueTypeDocumentation:
		return "documentation"
	case IssueTypeTesting:
		return "testing"
	case IssueTypeAccessibility:
		return "accessibility"
	default:
		return severityUnknown
	}
}

func (is IssueSeverity) String() string {
	switch is {
	case IssueSeverityInfo:
		return "info"
	case IssueSeverityLow:
		return severityLow
	case IssueSeverityMedium:
		return severityMedium
	case IssueSeverityHigh:
		return severityHigh
	case IssueSeverityCritical:
		return severityCritical
	default:
		return severityUnknown
	}
}
