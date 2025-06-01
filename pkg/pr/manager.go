// Package pr provides pull request management and automation functionality for ccAgents
package pr

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fumiya-kume/cca/pkg/analysis"
	"github.com/fumiya-kume/cca/pkg/commit"
	"github.com/fumiya-kume/cca/pkg/review"
)

// Status constants
const (
	statusFailure     = "failure"
	statusUnknown     = "unknown"
)

// PR type constants
const (
	prTypeBugfix       = "bugfix"
	prTypeHotfix       = "hotfix"
	prTypeRefactor     = "refactor"
	prTypeDocumentation = "documentation"
	prTypeTest         = "test"
)

// PRManager handles pull request automation and lifecycle management
type PRManager struct {
	config               PRConfig
	templateGenerator    *TemplateGenerator
	descriptionGenerator *DescriptionGenerator
	checksMonitor        *ChecksMonitor
	failureAnalyzer      *FailureAnalyzer
	client               GitHubClient
}

// PRConfig configures the PR manager
type PRConfig struct {
	AutoCreate             bool
	AutoMerge              bool
	RequireReviews         bool
	MinReviewers           int
	AutoLabeling           bool
	AutoAssignment         bool
	ChecksRequired         []string
	BranchProtection       bool
	ConflictResolution     ConflictStrategy
	FailureRetries         int
	Template               string
	CustomLabels           map[string]string
	NotificationSettings   NotificationSettings
	DraftMode              bool
	AutoUpdateBranch       bool
	SquashOnMerge          bool
	DeleteBranchAfterMerge bool
}

// PullRequest represents a pull request
type PullRequest struct {
	ID            int                  `json:"id"`
	Number        int                  `json:"number"`
	Title         string               `json:"title"`
	Description   string               `json:"description"`
	State         PRState              `json:"state"`
	Branch        string               `json:"branch"`
	BaseBranch    string               `json:"base_branch"`
	Author        string               `json:"author"`
	Reviewers     []string             `json:"reviewers"`
	Labels        []string             `json:"labels"`
	Assignees     []string             `json:"assignees"`
	Checks        []CheckStatus        `json:"checks"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	MergedAt      *time.Time           `json:"merged_at,omitempty"`
	Metadata      PRMetadata           `json:"metadata"`
	Conflicts     []ConflictInfo       `json:"conflicts"`
	Reviews       []ReviewStatus       `json:"reviews"`
	CommitPlan    *commit.CommitPlan   `json:"commit_plan,omitempty"`
	ReviewResults *review.ReviewResult `json:"review_results,omitempty"`
}

// PRMetadata contains metadata about the pull request
type PRMetadata struct {
	IssueReferences     []string                 `json:"issue_references"`
	ChangeType          ChangeType               `json:"change_type"`
	Priority            Priority                 `json:"priority"`
	EstimatedReviewTime time.Duration            `json:"estimated_review_time"`
	Complexity          ComplexityLevel          `json:"complexity"`
	TestCoverage        float64                  `json:"test_coverage"`
	AnalysisResult      *analysis.AnalysisResult `json:"analysis_result,omitempty"`
	GeneratedBy         string                   `json:"generated_by"`
	AutomationLevel     AutomationLevel          `json:"automation_level"`
	Tags                []string                 `json:"tags"`
}

// CheckStatus represents the status of a CI/CD check
type CheckStatus struct {
	Name        string      `json:"name"`
	Status      CheckState  `json:"status"`
	Conclusion  string      `json:"conclusion"`
	DetailsURL  string      `json:"details_url"`
	StartedAt   time.Time   `json:"started_at"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
	Output      CheckOutput `json:"output"`
}

// CheckOutput contains the output of a check
type CheckOutput struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Text    string `json:"text"`
}

// ConflictInfo represents a merge conflict
type ConflictInfo struct {
	File        string           `json:"file"`
	Lines       []int            `json:"lines"`
	Type        ConflictType     `json:"type"`
	Severity    ConflictSeverity `json:"severity"`
	Suggestion  string           `json:"suggestion"`
	AutoFixable bool             `json:"auto_fixable"`
}

// ReviewStatus represents the status of a code review
type ReviewStatus struct {
	Reviewer    string          `json:"reviewer"`
	State       ReviewState     `json:"state"`
	SubmittedAt time.Time       `json:"submitted_at"`
	Comments    []ReviewComment `json:"comments"`
}

// ReviewComment represents a review comment
type ReviewComment struct {
	ID       int         `json:"id"`
	File     string      `json:"file"`
	Line     int         `json:"line"`
	Body     string      `json:"body"`
	Type     CommentType `json:"type"`
	Resolved bool        `json:"resolved"`
}

// NotificationSettings configures notifications
type NotificationSettings struct {
	OnCreate   bool     `json:"on_create"`
	OnUpdate   bool     `json:"on_update"`
	OnReview   bool     `json:"on_review"`
	OnFailure  bool     `json:"on_failure"`
	OnMerge    bool     `json:"on_merge"`
	Recipients []string `json:"recipients"`
	Channels   []string `json:"channels"`
}

// PRTemplate represents a pull request template
type PRTemplate struct {
	Name        string              `json:"name"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Variables   map[string]string   `json:"variables"`
	Sections    []TemplateSection   `json:"sections"`
	Conditions  []TemplateCondition `json:"conditions"`
}

// TemplateSection represents a section in a PR template
type TemplateSection struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	Required bool   `json:"required"`
	Order    int    `json:"order"`
}

// TemplateCondition represents a condition for template usage
type TemplateCondition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// Enums and constants

type PRState int

const (
	PRStateOpen PRState = iota
	PRStateClosed
	PRStateMerged
	PRStateDraft
)

type CheckState int

const (
	CheckStatePending CheckState = iota
	CheckStateQueued
	CheckStateInProgress
	CheckStateCompleted
)

type ConflictType int

const (
	ConflictTypeContent ConflictType = iota
	ConflictTypeRename
	ConflictTypeDelete
	ConflictTypeBinary
)

type ConflictSeverity int

const (
	ConflictSeverityLow ConflictSeverity = iota
	ConflictSeverityMedium
	ConflictSeverityHigh
	ConflictSeverityCritical
)

type ConflictStrategy int

const (
	ConflictStrategyManual ConflictStrategy = iota
	ConflictStrategyAutoResolve
	ConflictStrategyRebase
	ConflictStrategyMerge
)

type ReviewState int

const (
	ReviewStatePending ReviewState = iota
	ReviewStateApproved
	ReviewStateChangesRequested
	ReviewStateCommented
	ReviewStateDismissed
)

type CommentType int

const (
	CommentTypeGeneral CommentType = iota
	CommentTypeSuggestion
	CommentTypeNitpick
	CommentTypeBlocking
	CommentTypePraise
)

type ChangeType int

const (
	ChangeTypeFeature ChangeType = iota
	ChangeTypeBugfix
	ChangeTypeHotfix
	ChangeTypeRefactor
	ChangeTypeDocumentation
	ChangeTypeConfiguration
	ChangeTypeTest
	ChangeTypeChore
)

type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
	PriorityUrgent
)

type ComplexityLevel int

const (
	ComplexityLevelSimple ComplexityLevel = iota
	ComplexityLevelModerate
	ComplexityLevelComplex
	ComplexityLevelVeryComplex
)

type AutomationLevel int

const (
	AutomationLevelManual AutomationLevel = iota
	AutomationLevelPartial
	AutomationLevelFull
	AutomationLevelAI
)

// GitHubClient interface for GitHub API operations
type GitHubClient interface {
	CreatePullRequest(ctx context.Context, pr *PullRequest) (*PullRequest, error)
	UpdatePullRequest(ctx context.Context, pr *PullRequest) error
	GetPullRequest(ctx context.Context, number int) (*PullRequest, error)
	ListPullRequests(ctx context.Context, filters PRFilters) ([]*PullRequest, error)
	MergePullRequest(ctx context.Context, number int, method MergeMethod) error
	GetChecks(ctx context.Context, ref string) ([]CheckStatus, error)
	AddLabels(ctx context.Context, number int, labels []string) error
	RequestReviews(ctx context.Context, number int, reviewers []string) error
	CreateComment(ctx context.Context, number int, comment string) error
}

type PRFilters struct {
	State  PRState
	Author string
	Labels []string
	Branch string
}

type MergeMethod int

const (
	MergeMethodMerge MergeMethod = iota
	MergeMethodSquash
	MergeMethodRebase
)

// NewPRManager creates a new PR manager
func NewPRManager(config PRConfig, client GitHubClient) *PRManager {
	// Set defaults
	if config.MinReviewers == 0 {
		config.MinReviewers = 1
	}
	if config.FailureRetries == 0 {
		config.FailureRetries = 3
	}

	pm := &PRManager{
		config: config,
		client: client,
	}

	// Initialize sub-components
	pm.templateGenerator = NewTemplateGenerator(TemplateGeneratorConfig{
		DefaultTemplate: config.Template,
		CustomLabels:    config.CustomLabels,
	})

	pm.descriptionGenerator = NewDescriptionGenerator(DescriptionGeneratorConfig{
		IncludeAnalysis:  true,
		IncludeChecklist: true,
		IncludeMetadata:  true,
	})

	pm.checksMonitor = NewChecksMonitor(ChecksMonitorConfig{
		RequiredChecks: config.ChecksRequired,
		RetryAttempts:  config.FailureRetries,
	})

	pm.failureAnalyzer = NewFailureAnalyzer(FailureAnalyzerConfig{
		AutoFix:         true,
		SuggestFixes:    true,
		AnalyzePatterns: true,
	})

	return pm
}

// CreatePullRequest creates a new pull request
func (pm *PRManager) CreatePullRequest(ctx context.Context, commitPlan *commit.CommitPlan, analysisResult *analysis.AnalysisResult) (*PullRequest, error) {
	// Generate PR metadata
	metadata := pm.generatePRMetadata(commitPlan, analysisResult)

	// Generate title and description
	title := pm.generateTitle(commitPlan, metadata)
	description, err := pm.descriptionGenerator.GenerateDescription(ctx, commitPlan, analysisResult, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to generate description: %w", err)
	}

	// Create PR object
	pr := &PullRequest{
		Title:       title,
		Description: description,
		State:       PRStateOpen,
		Branch:      pm.getCurrentBranch(ctx),
		BaseBranch:  "main", // TODO: Make configurable
		Author:      pm.getCurrentUser(ctx),
		Labels:      pm.generateLabels(metadata),
		Reviewers:   pm.selectReviewers(metadata),
		Assignees:   pm.selectAssignees(metadata),
		CreatedAt:   time.Now(),
		Metadata:    *metadata,
		CommitPlan:  commitPlan,
	}

	// Set as draft if configured
	if pm.config.DraftMode {
		pr.State = PRStateDraft
	}

	// Create the PR via GitHub API
	createdPR, err := pm.client.CreatePullRequest(ctx, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	// Add labels if auto-labeling is enabled
	if pm.config.AutoLabeling && len(pr.Labels) > 0 {
		if err := pm.client.AddLabels(ctx, createdPR.Number, pr.Labels); err != nil {
			// Log warning but don't fail
			fmt.Printf("Warning: failed to add labels: %v\n", err)
		}
	}

	// Request reviews if auto-assignment is enabled
	if pm.config.AutoAssignment && len(pr.Reviewers) > 0 {
		if err := pm.client.RequestReviews(ctx, createdPR.Number, pr.Reviewers); err != nil {
			// Log warning but don't fail
			fmt.Printf("Warning: failed to request reviews: %v\n", err)
		}
	}

	return createdPR, nil
}

// UpdatePullRequest updates an existing pull request
func (pm *PRManager) UpdatePullRequest(ctx context.Context, pr *PullRequest, commitPlan *commit.CommitPlan) error {
	// Update metadata
	if commitPlan != nil {
		analysisResult := pr.Metadata.AnalysisResult
		metadata := pm.generatePRMetadata(commitPlan, analysisResult)
		pr.Metadata = *metadata
		pr.CommitPlan = commitPlan
	}

	// Regenerate description if needed
	if pm.shouldRegenerateDescription(pr) {
		description, err := pm.descriptionGenerator.GenerateDescription(ctx, pr.CommitPlan, pr.Metadata.AnalysisResult, &pr.Metadata)
		if err != nil {
			return fmt.Errorf("failed to regenerate description: %w", err)
		}
		pr.Description = description
	}

	// Update via GitHub API
	return pm.client.UpdatePullRequest(ctx, pr)
}

// MonitorPullRequest monitors a pull request for check status and updates
func (pm *PRManager) MonitorPullRequest(ctx context.Context, prNumber int) error {
	pr, err := pm.client.GetPullRequest(ctx, prNumber)
	if err != nil {
		return fmt.Errorf("failed to get pull request: %w", err)
	}

	// Monitor checks
	if err := pm.checksMonitor.MonitorChecks(ctx, pr, pm.client); err != nil {
		return fmt.Errorf("failed to monitor checks: %w", err)
	}

	// Analyze failures if any
	if pm.hasFailedChecks(pr) {
		failures, err := pm.failureAnalyzer.AnalyzeFailures(ctx, pr, pm.client)
		if err != nil {
			return fmt.Errorf("failed to analyze failures: %w", err)
		}

		// Generate fixes if possible
		fixes := pm.failureAnalyzer.GenerateFixes(ctx, failures)
		if len(fixes) > 0 {
			if err := pm.applyAutomaticFixes(ctx, pr, fixes); err != nil {
				fmt.Printf("Warning: failed to apply automatic fixes: %v\n", err)
			}
		}
	}

	// Auto-merge if all conditions are met
	if pm.config.AutoMerge && pm.canAutoMerge(pr) {
		return pm.mergePullRequest(ctx, pr)
	}

	return nil
}

// MergePullRequest merges a pull request
func (pm *PRManager) MergePullRequest(ctx context.Context, prNumber int) error {
	pr, err := pm.client.GetPullRequest(ctx, prNumber)
	if err != nil {
		return fmt.Errorf("failed to get pull request: %w", err)
	}

	return pm.mergePullRequest(ctx, pr)
}

// mergePullRequest performs the actual merge
func (pm *PRManager) mergePullRequest(ctx context.Context, pr *PullRequest) error {
	// Check merge conditions
	if !pm.canMerge(pr) {
		return fmt.Errorf("pull request cannot be merged: conditions not met")
	}

	// Determine merge method
	method := MergeMethodMerge
	if pm.config.SquashOnMerge {
		method = MergeMethodSquash
	}

	// Perform the merge
	if err := pm.client.MergePullRequest(ctx, pr.Number, method); err != nil {
		return fmt.Errorf("failed to merge pull request: %w", err)
	}

	// Update PR state
	pr.State = PRStateMerged
	now := time.Now()
	pr.MergedAt = &now

	return nil
}

// Helper methods

func (pm *PRManager) generatePRMetadata(commitPlan *commit.CommitPlan, analysisResult *analysis.AnalysisResult) *PRMetadata {
	metadata := &PRMetadata{
		IssueReferences:     pm.extractIssueReferences(commitPlan),
		ChangeType:          pm.determineChangeType(commitPlan),
		Priority:            pm.determinePriority(commitPlan),
		EstimatedReviewTime: pm.estimateReviewTime(commitPlan),
		Complexity:          pm.assessComplexity(commitPlan),
		AnalysisResult:      analysisResult,
		GeneratedBy:         "cca-ai",
		AutomationLevel:     AutomationLevelAI,
		Tags:                pm.generateTags(commitPlan),
	}

	if commitPlan != nil && commitPlan.ValidationResult != nil {
		metadata.TestCoverage = pm.extractTestCoverage(commitPlan)
	}

	return metadata
}

func (pm *PRManager) generateTitle(commitPlan *commit.CommitPlan, metadata *PRMetadata) string {
	if commitPlan == nil || len(commitPlan.Commits) == 0 {
		return "Update repository"
	}

	// Use the primary commit's message as the title
	primaryCommit := commitPlan.Commits[0]
	title := primaryCommit.Message.Subject

	// Add prefix based on change type
	switch metadata.ChangeType {
	case ChangeTypeFeature:
		if !strings.HasPrefix(title, "feat") {
			title = "feat: " + title
		}
	case ChangeTypeBugfix:
		if !strings.HasPrefix(title, "fix") {
			title = "fix: " + title
		}
	case ChangeTypeHotfix:
		title = "hotfix: " + title
	}

	// Add priority indicator for urgent changes
	if metadata.Priority == PriorityUrgent || metadata.Priority == PriorityCritical {
		title = "🚨 " + title
	}

	return title
}

func (pm *PRManager) generateLabels(metadata *PRMetadata) []string {
	var labels []string

	// Add change type label
	switch metadata.ChangeType {
	case ChangeTypeFeature:
		labels = append(labels, "enhancement")
	case ChangeTypeBugfix:
		labels = append(labels, "bug")
	case ChangeTypeHotfix:
		labels = append(labels, "hotfix")
	case ChangeTypeRefactor:
		labels = append(labels, "refactor")
	case ChangeTypeDocumentation:
		labels = append(labels, "documentation")
	case ChangeTypeTest:
		labels = append(labels, "test")
	}

	// Add priority label
	switch metadata.Priority {
	case PriorityUrgent:
		labels = append(labels, "urgent")
	case PriorityCritical:
		labels = append(labels, "critical")
	case PriorityHigh:
		labels = append(labels, "high-priority")
	}

	// Add complexity label
	switch metadata.Complexity {
	case ComplexityLevelComplex, ComplexityLevelVeryComplex:
		labels = append(labels, "complex")
	}

	// Add automation label
	labels = append(labels, "automated")

	return labels
}

func (pm *PRManager) selectReviewers(metadata *PRMetadata) []string {
	// This would implement intelligent reviewer selection
	// based on CODEOWNERS, past reviews, expertise, etc.
	return []string{} // Placeholder
}

func (pm *PRManager) selectAssignees(metadata *PRMetadata) []string {
	// This would implement intelligent assignee selection
	return []string{} // Placeholder
}

func (pm *PRManager) getCurrentBranch(ctx context.Context) string {
	// This would get the current git branch
	return "feature/automated-changes" // Placeholder
}

func (pm *PRManager) getCurrentUser(ctx context.Context) string {
	// This would get the current git user
	return "cca-bot" // Placeholder
}

func (pm *PRManager) shouldRegenerateDescription(pr *PullRequest) bool {
	// Check if description should be regenerated based on changes
	return false // Placeholder
}

func (pm *PRManager) hasFailedChecks(pr *PullRequest) bool {
	for _, check := range pr.Checks {
		if check.Status == CheckStateCompleted && check.Conclusion == statusFailure {
			return true
		}
	}
	return false
}

func (pm *PRManager) canAutoMerge(pr *PullRequest) bool {
	return pm.canMerge(pr) && pm.config.AutoMerge
}

func (pm *PRManager) canMerge(pr *PullRequest) bool {
	// Check all merge conditions
	if pr.State != PRStateOpen {
		return false
	}

	// Check required reviews
	if pm.config.RequireReviews {
		approvedCount := 0
		for _, review := range pr.Reviews {
			if review.State == ReviewStateApproved {
				approvedCount++
			}
		}
		if approvedCount < pm.config.MinReviewers {
			return false
		}
	}

	// Check required checks
	for _, requiredCheck := range pm.config.ChecksRequired {
		found := false
		for _, check := range pr.Checks {
			if check.Name == requiredCheck && check.Status == CheckStateCompleted && check.Conclusion == "success" {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check for conflicts
	if len(pr.Conflicts) > 0 {
		return false
	}

	return true
}

func (pm *PRManager) applyAutomaticFixes(ctx context.Context, pr *PullRequest, fixes []AutomaticFix) error {
	// This would apply automatic fixes to the PR
	return nil // Placeholder
}

func (pm *PRManager) extractIssueReferences(commitPlan *commit.CommitPlan) []string {
	var references []string
	if commitPlan == nil {
		return references
	}

	for _, c := range commitPlan.Commits {
		// Extract issue references from commit messages
		// This would use regex to find patterns like "fixes #123", "closes #456"
		_ = c // TODO: Implement issue reference extraction
	}

	return references
}

func (pm *PRManager) determineChangeType(commitPlan *commit.CommitPlan) ChangeType {
	if commitPlan == nil || len(commitPlan.Commits) == 0 {
		return ChangeTypeChore
	}

	// Analyze commit types to determine overall change type
	primaryCommit := commitPlan.Commits[0]
	switch primaryCommit.Type {
	case commit.CommitTypeFeat:
		return ChangeTypeFeature
	case commit.CommitTypeFix:
		return ChangeTypeBugfix
	case commit.CommitTypeDocs:
		return ChangeTypeDocumentation
	case commit.CommitTypeTest:
		return ChangeTypeTest
	case commit.CommitTypeRefactor:
		return ChangeTypeRefactor
	default:
		return ChangeTypeChore
	}
}

func (pm *PRManager) determinePriority(commitPlan *commit.CommitPlan) Priority {
	if commitPlan == nil || len(commitPlan.Commits) == 0 {
		return PriorityMedium
	}

	// Determine priority based on commit metadata
	maxPriority := PriorityLow
	for _, commit := range commitPlan.Commits {
		if commit.Breaking {
			return PriorityCritical
		}
		if commit.Priority > int(maxPriority) {
			maxPriority = Priority(commit.Priority)
		}
	}

	return maxPriority
}

func (pm *PRManager) estimateReviewTime(commitPlan *commit.CommitPlan) time.Duration {
	if commitPlan == nil {
		return time.Hour
	}

	// Base time + time per commit + time based on complexity
	baseTime := time.Minute * 30
	commitTime := time.Duration(len(commitPlan.Commits)) * time.Minute * 15

	return baseTime + commitTime
}

func (pm *PRManager) assessComplexity(commitPlan *commit.CommitPlan) ComplexityLevel {
	if commitPlan == nil || len(commitPlan.Commits) == 0 {
		return ComplexityLevelSimple
	}

	totalChanges := commitPlan.TotalChanges
	commitCount := len(commitPlan.Commits)

	if totalChanges > 500 || commitCount > 10 {
		return ComplexityLevelVeryComplex
	}
	if totalChanges > 200 || commitCount > 5 {
		return ComplexityLevelComplex
	}
	if totalChanges > 50 || commitCount > 2 {
		return ComplexityLevelModerate
	}

	return ComplexityLevelSimple
}

func (pm *PRManager) extractTestCoverage(commitPlan *commit.CommitPlan) float64 {
	// Extract test coverage information from validation results
	return 0.0 // Placeholder
}

func (pm *PRManager) generateTags(commitPlan *commit.CommitPlan) []string {
	var tags []string

	if commitPlan == nil {
		return tags
	}

	// Generate tags based on commit plan
	if commitPlan.Strategy == commit.CommitStrategyFeature {
		tags = append(tags, "feature")
	}

	return tags
}

// AutomaticFix represents an automatic fix for a failure
type AutomaticFix struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	File        string  `json:"file"`
	Line        int     `json:"line"`
	Fix         string  `json:"fix"`
	Confidence  float64 `json:"confidence"`
}

// String methods for enums

func (ps PRState) String() string {
	switch ps {
	case PRStateOpen:
		return "open"
	case PRStateClosed:
		return "closed"
	case PRStateMerged:
		return "merged"
	case PRStateDraft:
		return "draft"
	default:
		return statusUnknown
	}
}

func (ct ChangeType) String() string {
	switch ct {
	case ChangeTypeFeature:
		return "feature"
	case ChangeTypeBugfix:
		return prTypeBugfix
	case ChangeTypeHotfix:
		return prTypeHotfix
	case ChangeTypeRefactor:
		return prTypeRefactor
	case ChangeTypeDocumentation:
		return prTypeDocumentation
	case ChangeTypeConfiguration:
		return "configuration"
	case ChangeTypeTest:
		return prTypeTest
	case ChangeTypeChore:
		return "chore"
	default:
		return statusUnknown
	}
}

func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityMedium:
		return "medium"
	case PriorityHigh:
		return "high"
	case PriorityCritical:
		return "critical"
	case PriorityUrgent:
		return "urgent"
	default:
		return statusUnknown
	}
}

func (cl ComplexityLevel) String() string {
	switch cl {
	case ComplexityLevelSimple:
		return "simple"
	case ComplexityLevelModerate:
		return "moderate"
	case ComplexityLevelComplex:
		return "complex"
	case ComplexityLevelVeryComplex:
		return "very-complex"
	default:
		return statusUnknown
	}
}

func (al AutomationLevel) String() string {
	switch al {
	case AutomationLevelManual:
		return "manual"
	case AutomationLevelPartial:
		return "partial"
	case AutomationLevelFull:
		return "full"
	case AutomationLevelAI:
		return "ai"
	default:
		return statusUnknown
	}
}
