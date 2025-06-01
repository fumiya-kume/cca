// Package commit provides commit message generation and management functionality for ccAgents
package commit

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fumiya-kume/cca/pkg/analysis"
)

// CommitManager handles intelligent commit generation and management
type CommitManager struct {
	config           CommitConfig
	changeAnalyzer   *ChangeAnalyzer
	messageGenerator *MessageGenerator
	validator        *CommitValidator
	workingDir       string
}

// CommitConfig configures the commit manager
type CommitConfig struct {
	ConventionalCommits  bool
	MaxCommitSize        int
	AtomicCommits        bool
	AutoSquash           bool
	ValidateBeforeCommit bool
	AllowAmending        bool
	GenerateCoAuthor     bool
	SignCommits          bool
	CommitTemplate       string
	MessageStyle         MessageStyle
	MaxMessageLength     int
	RequiredScopes       []string
	DryRun               bool
}

// CommitPlan represents a planned commit strategy
type CommitPlan struct {
	Commits          []PlannedCommit     `json:"commits"`
	Strategy         CommitStrategy      `json:"strategy"`
	Dependencies     map[string][]string `json:"dependencies"`
	TotalChanges     int                 `json:"total_changes"`
	EstimatedTime    time.Duration       `json:"estimated_time"`
	ValidationResult *ValidationResult   `json:"validation_result"`
	Metadata         CommitMetadata      `json:"metadata"`
}

// PlannedCommit represents a single planned commit
type PlannedCommit struct {
	ID           string              `json:"id"`
	Message      ConventionalMessage `json:"message"`
	Files        []string            `json:"files"`
	Changes      []FileChange        `json:"changes"`
	Dependencies []string            `json:"dependencies"`
	Type         CommitType          `json:"type"`
	Scope        string              `json:"scope"`
	Breaking     bool                `json:"breaking"`
	Size         CommitSize          `json:"size"`
	Priority     int                 `json:"priority"`
	Metadata     map[string]string   `json:"metadata"`
}

// ConventionalMessage represents a conventional commit message
type ConventionalMessage struct {
	Type       string   `json:"type"`
	Scope      string   `json:"scope,omitempty"`
	Subject    string   `json:"subject"`
	Body       string   `json:"body,omitempty"`
	Footer     string   `json:"footer,omitempty"`
	Breaking   bool     `json:"breaking"`
	CoAuthors  []string `json:"co_authors,omitempty"`
	References []string `json:"references,omitempty"`
	Full       string   `json:"full"`
}

// FileChange represents a change to a file
type FileChange struct {
	Path      string     `json:"path"`
	Type      ChangeType `json:"type"`
	Additions int        `json:"additions"`
	Deletions int        `json:"deletions"`
	Content   string     `json:"content,omitempty"`
	Binary    bool       `json:"binary"`
	Renamed   bool       `json:"renamed"`
	OldPath   string     `json:"old_path,omitempty"`
}

// CommitMetadata contains metadata about commits
type CommitMetadata struct {
	Author        string                `json:"author"`
	CoAuthors     []string              `json:"co_authors"`
	Timestamp     time.Time             `json:"timestamp"`
	Branch        string                `json:"branch"`
	Tags          []string              `json:"tags"`
	Issues        []string              `json:"issues"`
	ProjectInfo   *analysis.ProjectInfo `json:"project_info"`
	ReviewResults map[string]string     `json:"review_results"`
}

// ValidationResult contains commit validation results
type ValidationResult struct {
	Valid       bool                `json:"valid"`
	Errors      []ValidationError   `json:"errors"`
	Warnings    []ValidationWarning `json:"warnings"`
	Score       float64             `json:"score"`
	Suggestions []string            `json:"suggestions"`
}

// ValidationError represents a commit validation error
type ValidationError struct {
	Type     string        `json:"type"`
	Message  string        `json:"message"`
	File     string        `json:"file,omitempty"`
	Line     int           `json:"line,omitempty"`
	Severity ErrorSeverity `json:"severity"`
}

// ValidationWarning represents a commit validation warning
type ValidationWarning struct {
	Type       string `json:"type"`
	Message    string `json:"message"`
	File       string `json:"file,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

// Enums and constants

type MessageStyle int

const (
	MessageStyleConventional MessageStyle = iota
	MessageStyleTraditional
	MessageStyleCustom
)

type CommitStrategy int

const (
	CommitStrategyAtomic CommitStrategy = iota
	CommitStrategyLogical
	CommitStrategyFeature
	CommitStrategyFile
	CommitStrategyTemporal
)

type CommitType int

const (
	CommitTypeFeat CommitType = iota
	CommitTypeFix
	CommitTypeDocs
	CommitTypeStyle
	CommitTypeRefactor
	CommitTypePerf
	CommitTypeTest
	CommitTypeBuild
	CommitTypeCI
	CommitTypeChore
	CommitTypeRevert
)

type ChangeType int

const (
	ChangeTypeAdd ChangeType = iota
	ChangeTypeModify
	ChangeTypeDelete
	ChangeTypeRename
	ChangeTypeCopy
	ChangeTypeUntracked
)

type CommitSize int

const (
	CommitSizeSmall CommitSize = iota
	CommitSizeMedium
	CommitSizeLarge
	CommitSizeHuge
)

type ErrorSeverity int

const (
	ErrorSeverityInfo ErrorSeverity = iota
	ErrorSeverityWarning
	ErrorSeverityError
	ErrorSeverityCritical
)

// NewCommitManager creates a new commit manager
func NewCommitManager(config CommitConfig, workingDir string) (*CommitManager, error) {
	// Set defaults
	if config.MaxCommitSize == 0 {
		config.MaxCommitSize = 100 // max files per commit
	}
	if config.MaxMessageLength == 0 {
		config.MaxMessageLength = 50 // conventional commit subject length
	}

	cm := &CommitManager{
		config:     config,
		workingDir: workingDir,
	}

	// Initialize sub-components
	cm.changeAnalyzer = NewChangeAnalyzer(ChangeAnalyzerConfig{
		AtomicChanges: config.AtomicCommits,
		MaxCommitSize: config.MaxCommitSize,
	})

	cm.messageGenerator = NewMessageGenerator(MessageGeneratorConfig{
		Style:               config.MessageStyle,
		MaxLength:           config.MaxMessageLength,
		RequiredScopes:      config.RequiredScopes,
		Template:            config.CommitTemplate,
		ConventionalCommits: config.ConventionalCommits,
	})

	cm.validator = NewCommitValidator(CommitValidatorConfig{
		ConventionalCommits: config.ConventionalCommits,
		MaxMessageLength:    config.MaxMessageLength,
		RequiredScopes:      config.RequiredScopes,
		AllowBreaking:       true,
	})

	return cm, nil
}

// CreateCommitPlan analyzes changes and creates an intelligent commit plan
func (cm *CommitManager) CreateCommitPlan(ctx context.Context, analysisResult *analysis.AnalysisResult) (*CommitPlan, error) {
	// Analyze current changes
	changes, err := cm.analyzeWorkingDirectory(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze working directory: %w", err)
	}

	if len(changes) == 0 {
		return &CommitPlan{
			Commits:      []PlannedCommit{},
			Strategy:     CommitStrategyAtomic,
			TotalChanges: 0,
		}, nil
	}

	// Group changes into logical commits
	commitGroups, err := cm.changeAnalyzer.GroupChanges(ctx, changes, analysisResult)
	if err != nil {
		return nil, fmt.Errorf("failed to group changes: %w", err)
	}

	// Generate commit messages and plans
	var plannedCommits []PlannedCommit
	dependencies := make(map[string][]string)

	for i, group := range commitGroups {
		commitID := fmt.Sprintf("commit_%d", i+1)

		// Generate conventional commit message
		message, err := cm.messageGenerator.GenerateMessage(ctx, group, analysisResult)
		if err != nil {
			return nil, fmt.Errorf("failed to generate message for commit %s: %w", commitID, err)
		}

		// Determine commit properties
		commitType := cm.determineCommitType(group)
		scope := cm.determineScope(group, analysisResult)
		isBreaking := cm.isBreakingChange(group)
		size := cm.calculateCommitSize(group)

		plannedCommit := PlannedCommit{
			ID:       commitID,
			Message:  *message,
			Files:    cm.extractFilePaths(group),
			Changes:  group,
			Type:     commitType,
			Scope:    scope,
			Breaking: isBreaking,
			Size:     size,
			Priority: cm.calculatePriority(commitType, size, isBreaking),
			Metadata: make(map[string]string),
		}

		// Analyze dependencies
		deps := cm.analyzeDependencies(plannedCommit, plannedCommits)
		plannedCommit.Dependencies = deps
		dependencies[commitID] = deps

		plannedCommits = append(plannedCommits, plannedCommit)
	}

	// Sort commits by priority and dependencies
	sortedCommits := cm.sortCommitsByDependencies(plannedCommits, dependencies)

	// Create metadata
	metadata, err := cm.createCommitMetadata(ctx, analysisResult)
	if err != nil {
		return nil, fmt.Errorf("failed to create commit metadata: %w", err)
	}

	plan := &CommitPlan{
		Commits:       sortedCommits,
		Strategy:      cm.determineStrategy(sortedCommits),
		Dependencies:  dependencies,
		TotalChanges:  len(changes),
		EstimatedTime: cm.estimateCommitTime(sortedCommits),
		Metadata:      *metadata,
	}

	// Validate the plan
	if cm.config.ValidateBeforeCommit {
		validationResult, err := cm.validator.ValidatePlan(ctx, plan)
		if err != nil {
			return plan, fmt.Errorf("validation failed: %w", err)
		}
		plan.ValidationResult = validationResult
	}

	return plan, nil
}

// ExecuteCommitPlan executes the planned commits
func (cm *CommitManager) ExecuteCommitPlan(ctx context.Context, plan *CommitPlan) error {
	if cm.config.DryRun {
		return cm.previewCommitPlan(plan)
	}

	for _, commit := range plan.Commits {
		if err := cm.executeCommit(ctx, commit); err != nil {
			return fmt.Errorf("failed to execute commit %s: %w", commit.ID, err)
		}
	}

	return nil
}

// executeCommit executes a single commit
func (cm *CommitManager) executeCommit(ctx context.Context, commit PlannedCommit) error {
	// Stage files
	for _, file := range commit.Files {
		if err := cm.stageFile(ctx, file); err != nil {
			return fmt.Errorf("failed to stage file %s: %w", file, err)
		}
	}

	// Create commit
	args := []string{"commit", "-m", commit.Message.Full}

	if cm.config.SignCommits {
		args = append(args, "-S")
	}

	// #nosec G204 - git args are constructed from trusted configuration data
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = cm.workingDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	return nil
}

// stageFile stages a file for commit
func (cm *CommitManager) stageFile(ctx context.Context, file string) error {
	// #nosec G204 - git add with validated file path
	cmd := exec.CommandContext(ctx, "git", "add", file)
	cmd.Dir = cm.workingDir
	return cmd.Run()
}

// analyzeWorkingDirectory analyzes the current working directory for changes
func (cm *CommitManager) analyzeWorkingDirectory(ctx context.Context) ([]FileChange, error) {
	var changes []FileChange

	// Get git status
	// #nosec G204 - git status command with fixed arguments
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = cm.workingDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get git status: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		change, err := cm.parseGitStatusLine(line)
		if err != nil {
			continue // Skip invalid lines
		}

		changes = append(changes, *change)
	}

	return changes, nil
}

// parseGitStatusLine parses a git status line into a FileChange
func (cm *CommitManager) parseGitStatusLine(line string) (*FileChange, error) {
	if err := cm.validateStatusLine(line); err != nil {
		return nil, err
	}

	indexStatus, workStatus, filePath := cm.extractStatusComponents(line)
	changeType := cm.determineChangeType(indexStatus, workStatus)

	change := &FileChange{
		Path: filePath,
		Type: changeType,
	}

	cm.addChangeStatistics(change)

	return change, nil
}

// validateStatusLine validates that the git status line has the correct format
func (cm *CommitManager) validateStatusLine(line string) error {
	if len(line) < 3 {
		return fmt.Errorf("invalid status line: %s", line)
	}
	return nil
}

// extractStatusComponents extracts index status, work status and file path from status line
func (cm *CommitManager) extractStatusComponents(line string) (byte, byte, string) {
	indexStatus := line[0]
	workStatus := line[1]
	filePath := strings.TrimSpace(line[3:])
	return indexStatus, workStatus, filePath
}

// determineChangeType determines the change type based on status characters
func (cm *CommitManager) determineChangeType(indexStatus, workStatus byte) ChangeType {
	if cm.hasStatus('A', indexStatus, workStatus) {
		return ChangeTypeAdd
	}
	if cm.hasStatus('M', indexStatus, workStatus) {
		return ChangeTypeModify
	}
	if cm.hasStatus('D', indexStatus, workStatus) {
		return ChangeTypeDelete
	}
	if cm.hasStatus('R', indexStatus, workStatus) {
		return ChangeTypeRename
	}
	if cm.hasStatus('C', indexStatus, workStatus) {
		return ChangeTypeCopy
	}
	if cm.hasStatus('?', indexStatus, workStatus) {
		return ChangeTypeUntracked
	}
	return ChangeTypeModify // Default case
}

// hasStatus checks if either index or work status matches the given status character
func (cm *CommitManager) hasStatus(status byte, indexStatus, workStatus byte) bool {
	return indexStatus == status || workStatus == status
}

// addChangeStatistics adds change statistics to the FileChange if applicable
func (cm *CommitManager) addChangeStatistics(change *FileChange) {
	if change.Type == ChangeTypeUntracked {
		return
	}

	stats, err := cm.getChangeStats(change.Path)
	if err == nil {
		change.Additions = stats.Additions
		change.Deletions = stats.Deletions
	}
}

// getChangeStats gets the change statistics for a file
func (cm *CommitManager) getChangeStats(filePath string) (*ChangeStats, error) {
	// #nosec G204 - git diff with validated file path
	cmd := exec.Command("git", "diff", "--numstat", "HEAD", "--", filePath)
	cmd.Dir = cm.workingDir

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	line := strings.TrimSpace(string(output))
	if line == "" {
		return &ChangeStats{}, nil
	}

	parts := strings.Fields(line)
	if len(parts) < 2 {
		return &ChangeStats{}, nil
	}

	var additions, deletions int
	// Parse additions count, default to 0 if parsing fails
	if _, err := fmt.Sscanf(parts[0], "%d", &additions); err != nil {
		// Leave additions as 0 if parsing fails
		additions = 0
	}
	// Parse deletions count, default to 0 if parsing fails
	if _, err := fmt.Sscanf(parts[1], "%d", &deletions); err != nil {
		// Leave deletions as 0 if parsing fails
		deletions = 0
	}

	return &ChangeStats{
		Additions: additions,
		Deletions: deletions,
	}, nil
}

// Helper methods

func (cm *CommitManager) extractFilePaths(changes []FileChange) []string {
	var paths []string
	for _, change := range changes {
		paths = append(paths, change.Path)
	}
	return paths
}

func (cm *CommitManager) determineCommitType(changes []FileChange) CommitType {
	// Analyze changes to determine commit type
	hasTests := false
	hasDocs := false
	hasFeatures := false
	hasFixes := false
	hasConfig := false

	for _, change := range changes {
		path := strings.ToLower(change.Path)

		switch {
		case strings.Contains(path, "test"):
			hasTests = true
		case strings.Contains(path, "doc") || strings.HasSuffix(path, ".md"):
			hasDocs = true
		case strings.Contains(path, "config") || strings.Contains(path, ".config"):
			hasConfig = true
		case change.Type == ChangeTypeAdd:
			hasFeatures = true
		case strings.Contains(path, "fix") || strings.Contains(path, "bug"):
			hasFixes = true
		}
	}

	// Priority order for commit type determination
	if hasFixes {
		return CommitTypeFix
	}
	if hasFeatures {
		return CommitTypeFeat
	}
	if hasTests {
		return CommitTypeTest
	}
	if hasDocs {
		return CommitTypeDocs
	}
	if hasConfig {
		return CommitTypeChore
	}

	return CommitTypeChore
}

func (cm *CommitManager) determineScope(changes []FileChange, analysisResult *analysis.AnalysisResult) string {
	if analysisResult == nil {
		return ""
	}

	// Determine scope based on file paths and project structure
	scopeMap := make(map[string]int)

	for _, change := range changes {
		dir := filepath.Dir(change.Path)
		parts := strings.Split(dir, "/")

		if len(parts) > 0 {
			scope := parts[0]
			if scope != "." {
				scopeMap[scope]++
			}
		}
	}

	// Return the most common scope
	var maxScope string
	var maxCount int
	for scope, count := range scopeMap {
		if count > maxCount {
			maxCount = count
			maxScope = scope
		}
	}

	return maxScope
}

func (cm *CommitManager) isBreakingChange(changes []FileChange) bool {
	// Simple heuristic for breaking changes
	for _, change := range changes {
		if change.Type == ChangeTypeDelete {
			return true
		}
		if strings.Contains(change.Path, "api") && change.Type == ChangeTypeModify {
			return true
		}
	}
	return false
}

func (cm *CommitManager) calculateCommitSize(changes []FileChange) CommitSize {
	totalChanges := 0
	for _, change := range changes {
		totalChanges += change.Additions + change.Deletions
	}

	switch {
	case totalChanges <= 10:
		return CommitSizeSmall
	case totalChanges <= 50:
		return CommitSizeMedium
	case totalChanges <= 200:
		return CommitSizeLarge
	default:
		return CommitSizeHuge
	}
}

func (cm *CommitManager) calculatePriority(commitType CommitType, size CommitSize, breaking bool) int {
	priority := 10

	// Adjust for commit type
	switch commitType {
	case CommitTypeFix:
		priority += 20
	case CommitTypeFeat:
		priority += 15
	case CommitTypeTest:
		priority += 5
	case CommitTypeDocs:
		priority += 3
	}

	// Adjust for size (smaller commits get higher priority)
	switch size {
	case CommitSizeSmall:
		priority += 10
	case CommitSizeMedium:
		priority += 5
	case CommitSizeLarge:
		priority += 0
	case CommitSizeHuge:
		priority -= 5
	}

	// Breaking changes get highest priority
	if breaking {
		priority += 50
	}

	return priority
}

func (cm *CommitManager) analyzeDependencies(commit PlannedCommit, existingCommits []PlannedCommit) []string {
	var dependencies []string

	// Check for file dependencies
	for _, existing := range existingCommits {
		for _, file := range commit.Files {
			for _, existingFile := range existing.Files {
				if file == existingFile {
					dependencies = append(dependencies, existing.ID)
					break
				}
			}
		}
	}

	return dependencies
}

func (cm *CommitManager) sortCommitsByDependencies(commits []PlannedCommit, dependencies map[string][]string) []PlannedCommit {
	// Create a map for quick lookup
	commitMap := make(map[string]PlannedCommit)
	for _, commit := range commits {
		commitMap[commit.ID] = commit
	}

	// Sort by priority first, then by dependencies
	sort.Slice(commits, func(i, j int) bool {
		// Check dependencies first
		for _, dep := range dependencies[commits[j].ID] {
			if dep == commits[i].ID {
				return true
			}
		}
		for _, dep := range dependencies[commits[i].ID] {
			if dep == commits[j].ID {
				return false
			}
		}

		// If no dependency relationship, sort by priority
		return commits[i].Priority > commits[j].Priority
	})

	return commits
}

func (cm *CommitManager) determineStrategy(commits []PlannedCommit) CommitStrategy {
	if len(commits) <= 1 {
		return CommitStrategyAtomic
	}

	// Analyze commit patterns to determine strategy
	hasLogicalGroups := cm.hasLogicalGroups(commits)
	hasFeatureGroups := cm.hasFeatureGroups(commits)

	if hasFeatureGroups {
		return CommitStrategyFeature
	}
	if hasLogicalGroups {
		return CommitStrategyLogical
	}

	return CommitStrategyAtomic
}

func (cm *CommitManager) hasLogicalGroups(commits []PlannedCommit) bool {
	typeCount := make(map[CommitType]int)
	for _, commit := range commits {
		typeCount[commit.Type]++
	}
	return len(typeCount) > 1
}

func (cm *CommitManager) hasFeatureGroups(commits []PlannedCommit) bool {
	for _, commit := range commits {
		if commit.Type == CommitTypeFeat {
			return true
		}
	}
	return false
}

func (cm *CommitManager) estimateCommitTime(commits []PlannedCommit) time.Duration {
	// Estimate time based on number of commits and their complexity
	baseTime := time.Duration(len(commits)) * time.Minute * 2

	for _, commit := range commits {
		switch commit.Size {
		case CommitSizeHuge:
			baseTime += time.Minute * 5
		case CommitSizeLarge:
			baseTime += time.Minute * 3
		case CommitSizeMedium:
			baseTime += time.Minute * 1
		}
	}

	return baseTime
}

func (cm *CommitManager) createCommitMetadata(ctx context.Context, analysisResult *analysis.AnalysisResult) (*CommitMetadata, error) {
	metadata := &CommitMetadata{
		Timestamp: time.Now(),
	}

	// Get current branch
	// #nosec G204 - git branch command with fixed arguments
	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	cmd.Dir = cm.workingDir
	if output, err := cmd.Output(); err == nil {
		metadata.Branch = strings.TrimSpace(string(output))
	}

	// Get author info
	// #nosec G204 - git config command with fixed arguments
	cmd = exec.CommandContext(ctx, "git", "config", "user.name")
	cmd.Dir = cm.workingDir
	if output, err := cmd.Output(); err == nil {
		metadata.Author = strings.TrimSpace(string(output))
	}

	// Add project info if available
	if analysisResult != nil {
		metadata.ProjectInfo = analysisResult.ProjectInfo
	}

	return metadata, nil
}

func (cm *CommitManager) previewCommitPlan(plan *CommitPlan) error {
	fmt.Println("=== Commit Plan Preview ===")
	fmt.Printf("Strategy: %s\n", plan.Strategy)
	fmt.Printf("Total Commits: %d\n", len(plan.Commits))
	fmt.Printf("Total Changes: %d\n", plan.TotalChanges)
	fmt.Printf("Estimated Time: %s\n", plan.EstimatedTime)
	fmt.Println()

	for i, commit := range plan.Commits {
		fmt.Printf("Commit %d: %s\n", i+1, commit.ID)
		fmt.Printf("  Type: %s\n", commit.Type)
		fmt.Printf("  Scope: %s\n", commit.Scope)
		fmt.Printf("  Message: %s\n", commit.Message.Full)
		fmt.Printf("  Files: %v\n", commit.Files)
		if len(commit.Dependencies) > 0 {
			fmt.Printf("  Dependencies: %v\n", commit.Dependencies)
		}
		fmt.Println()
	}

	return nil
}

// ChangeStats represents change statistics for a file
type ChangeStats struct {
	Additions int
	Deletions int
}

// String methods for enums

func (ms MessageStyle) String() string {
	switch ms {
	case MessageStyleConventional:
		return "conventional"
	case MessageStyleTraditional:
		return "traditional"
	case MessageStyleCustom:
		return "custom"
	default:
		return statusUnknown
	}
}

func (cs CommitStrategy) String() string {
	switch cs {
	case CommitStrategyAtomic:
		return "atomic"
	case CommitStrategyLogical:
		return "logical"
	case CommitStrategyFeature:
		return "feature"
	case CommitStrategyFile:
		return "file"
	case CommitStrategyTemporal:
		return "temporal"
	default:
		return statusUnknown
	}
}

func (ct CommitType) String() string {
	switch ct {
	case CommitTypeFeat:
		return "feat"
	case CommitTypeFix:
		return "fix"
	case CommitTypeDocs:
		return "docs"
	case CommitTypeStyle:
		return "style"
	case CommitTypeRefactor:
		return "refactor"
	case CommitTypePerf:
		return "perf"
	case CommitTypeTest:
		return "test"
	case CommitTypeBuild:
		return "build"
	case CommitTypeCI:
		return "ci"
	case CommitTypeChore:
		return "chore"
	case CommitTypeRevert:
		return "revert"
	default:
		return statusUnknown
	}
}
