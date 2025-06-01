package commit

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fumiya-kume/cca/pkg/analysis"
)

// ChangeAnalyzer analyzes code changes and groups them into logical commits
type ChangeAnalyzer struct {
	config ChangeAnalyzerConfig
}

// ChangeAnalyzerConfig configures the change analyzer
type ChangeAnalyzerConfig struct {
	AtomicChanges  bool
	MaxCommitSize  int
	GroupByFeature bool
	GroupByModule  bool
	GroupByType    bool
	SeparateTests  bool
	SeparateDocs   bool
	MinGroupSize   int
	MaxGroupSize   int
}

// ChangeGroup represents a group of related changes
type ChangeGroup struct {
	ID          string       `json:"id"`
	Type        GroupType    `json:"type"`
	Description string       `json:"description"`
	Changes     []FileChange `json:"changes"`
	Score       float64      `json:"score"`
	Rationale   string       `json:"rationale"`
}

// GroupType represents the type of change group
type GroupType int

const (
	GroupTypeFeature GroupType = iota
	GroupTypeFix
	GroupTypeRefactor
	GroupTypeTest
	GroupTypeDocs
	GroupTypeConfig
	GroupTypeStyle
	GroupTypeMixed
)

// NewChangeAnalyzer creates a new change analyzer
func NewChangeAnalyzer(config ChangeAnalyzerConfig) *ChangeAnalyzer {
	// Set defaults
	if config.MaxCommitSize == 0 {
		config.MaxCommitSize = 100
	}
	if config.MinGroupSize == 0 {
		config.MinGroupSize = 1
	}
	if config.MaxGroupSize == 0 {
		config.MaxGroupSize = 20
	}

	return &ChangeAnalyzer{
		config: config,
	}
}

// GroupChanges groups file changes into logical commit groups
func (ca *ChangeAnalyzer) GroupChanges(ctx context.Context, changes []FileChange, analysisResult *analysis.AnalysisResult) ([][]FileChange, error) {
	if len(changes) == 0 {
		return [][]FileChange{}, nil
	}

	// Create initial groups
	groups := ca.createInitialGroups(changes, analysisResult)

	// Refine groups based on configuration
	groups = ca.refineGroups(groups, analysisResult)

	// Validate and adjust group sizes
	groups = ca.adjustGroupSizes(groups)

	// Convert to file change groups
	var result [][]FileChange
	for _, group := range groups {
		result = append(result, group.Changes)
	}

	return result, nil
}

// createInitialGroups creates initial groupings based on file patterns and types
func (ca *ChangeAnalyzer) createInitialGroups(changes []FileChange, analysisResult *analysis.AnalysisResult) []*ChangeGroup {
	var groups []*ChangeGroup

	if ca.config.AtomicChanges {
		// Create atomic groups (one file per group)
		for i, change := range changes {
			group := &ChangeGroup{
				ID:          fmt.Sprintf("atomic_%d", i+1),
				Type:        ca.determineGroupType([]FileChange{change}),
				Description: ca.generateDescription([]FileChange{change}),
				Changes:     []FileChange{change},
				Score:       1.0,
				Rationale:   "Atomic commit strategy",
			}
			groups = append(groups, group)
		}
		return groups
	}

	// Group by different strategies
	if ca.config.GroupByType {
		groups = append(groups, ca.groupByChangeType(changes)...)
	}

	if ca.config.GroupByModule {
		groups = append(groups, ca.groupByModule(changes, analysisResult)...)
	}

	if ca.config.GroupByFeature {
		groups = append(groups, ca.groupByFeature(changes, analysisResult)...)
	}

	// If no specific grouping strategy, create logical groups
	if len(groups) == 0 {
		groups = ca.createLogicalGroups(changes, analysisResult)
	}

	return groups
}

// groupByChangeType groups changes by their type (add, modify, delete, etc.)
func (ca *ChangeAnalyzer) groupByChangeType(changes []FileChange) []*ChangeGroup {
	typeGroups := make(map[ChangeType][]FileChange)

	for _, change := range changes {
		typeGroups[change.Type] = append(typeGroups[change.Type], change)
	}

	var groups []*ChangeGroup
	for changeType, groupChanges := range typeGroups {
		if len(groupChanges) > 0 {
			group := &ChangeGroup{
				ID:          fmt.Sprintf("type_%s", changeType.String()),
				Type:        ca.determineGroupType(groupChanges),
				Description: ca.generateDescription(groupChanges),
				Changes:     groupChanges,
				Score:       ca.calculateGroupScore(groupChanges),
				Rationale:   fmt.Sprintf("Grouped by change type: %s", changeType.String()),
			}
			groups = append(groups, group)
		}
	}

	return groups
}

// groupByModule groups changes by module/directory structure
func (ca *ChangeAnalyzer) groupByModule(changes []FileChange, analysisResult *analysis.AnalysisResult) []*ChangeGroup {
	moduleGroups := make(map[string][]FileChange)

	for _, change := range changes {
		module := ca.extractModule(change.Path, analysisResult)
		moduleGroups[module] = append(moduleGroups[module], change)
	}

	var groups []*ChangeGroup
	for module, groupChanges := range moduleGroups {
		if len(groupChanges) > 0 {
			group := &ChangeGroup{
				ID:          fmt.Sprintf("module_%s", module),
				Type:        ca.determineGroupType(groupChanges),
				Description: ca.generateDescription(groupChanges),
				Changes:     groupChanges,
				Score:       ca.calculateGroupScore(groupChanges),
				Rationale:   fmt.Sprintf("Grouped by module: %s", module),
			}
			groups = append(groups, group)
		}
	}

	return groups
}

// groupByFeature groups changes by feature based on file patterns and analysis
func (ca *ChangeAnalyzer) groupByFeature(changes []FileChange, analysisResult *analysis.AnalysisResult) []*ChangeGroup {
	featureGroups := make(map[string][]FileChange)

	for _, change := range changes {
		feature := ca.detectFeature(change, analysisResult)
		featureGroups[feature] = append(featureGroups[feature], change)
	}

	var groups []*ChangeGroup
	for feature, groupChanges := range featureGroups {
		if len(groupChanges) > 0 {
			group := &ChangeGroup{
				ID:          fmt.Sprintf("feature_%s", feature),
				Type:        ca.determineGroupType(groupChanges),
				Description: ca.generateDescription(groupChanges),
				Changes:     groupChanges,
				Score:       ca.calculateGroupScore(groupChanges),
				Rationale:   fmt.Sprintf("Grouped by feature: %s", feature),
			}
			groups = append(groups, group)
		}
	}

	return groups
}

// createLogicalGroups creates logical groups based on file relationships and patterns
func (ca *ChangeAnalyzer) createLogicalGroups(changes []FileChange, analysisResult *analysis.AnalysisResult) []*ChangeGroup {
	var groups []*ChangeGroup

	// Separate different types of changes
	testChanges := []FileChange{}
	docChanges := []FileChange{}
	configChanges := []FileChange{}
	codeChanges := []FileChange{}

	for _, change := range changes {
		switch {
		case ca.isTestFile(change.Path):
			testChanges = append(testChanges, change)
		case ca.isDocFile(change.Path):
			docChanges = append(docChanges, change)
		case ca.isConfigFile(change.Path):
			configChanges = append(configChanges, change)
		default:
			codeChanges = append(codeChanges, change)
		}
	}

	// If we didn't separate tests/docs, include them in code groups
	if !ca.config.SeparateTests {
		codeChanges = append(codeChanges, testChanges...)
	}
	if !ca.config.SeparateDocs {
		codeChanges = append(codeChanges, docChanges...)
	}

	// Create groups for each type
	if len(codeChanges) > 0 {
		codeGroups := ca.groupCodeChanges(codeChanges, analysisResult)
		groups = append(groups, codeGroups...)
	}

	if len(testChanges) > 0 && ca.config.SeparateTests {
		testGroup := &ChangeGroup{
			ID:          "tests",
			Type:        GroupTypeTest,
			Description: "Test updates",
			Changes:     testChanges,
			Score:       ca.calculateGroupScore(testChanges),
			Rationale:   "Separated test files",
		}
		groups = append(groups, testGroup)
	}

	if len(docChanges) > 0 && ca.config.SeparateDocs {
		docGroup := &ChangeGroup{
			ID:          "docs",
			Type:        GroupTypeDocs,
			Description: "Documentation updates",
			Changes:     docChanges,
			Score:       ca.calculateGroupScore(docChanges),
			Rationale:   "Separated documentation files",
		}
		groups = append(groups, docGroup)
	}

	if len(configChanges) > 0 {
		configGroup := &ChangeGroup{
			ID:          "config",
			Type:        GroupTypeConfig,
			Description: "Configuration changes",
			Changes:     configChanges,
			Score:       ca.calculateGroupScore(configChanges),
			Rationale:   "Separated configuration files",
		}
		groups = append(groups, configGroup)
	}

	// If no groups were created, create a single group with all changes
	if len(groups) == 0 {
		group := &ChangeGroup{
			ID:          "all_changes",
			Type:        GroupTypeMixed,
			Description: "All changes",
			Changes:     changes,
			Score:       ca.calculateGroupScore(changes),
			Rationale:   "Single group for all changes",
		}
		groups = append(groups, group)
	}

	return groups
}

// groupCodeChanges groups code changes based on file relationships
func (ca *ChangeAnalyzer) groupCodeChanges(changes []FileChange, analysisResult *analysis.AnalysisResult) []*ChangeGroup {
	if len(changes) <= ca.config.MaxGroupSize {
		// Small enough to be a single group
		group := &ChangeGroup{
			ID:          "code_changes",
			Type:        ca.determineGroupType(changes),
			Description: ca.generateDescription(changes),
			Changes:     changes,
			Score:       ca.calculateGroupScore(changes),
			Rationale:   "Code changes grouped together",
		}
		return []*ChangeGroup{group}
	}

	// Split large groups based on directory structure
	dirGroups := make(map[string][]FileChange)
	for _, change := range changes {
		dir := filepath.Dir(change.Path)
		if dir == "." {
			dir = "root"
		}
		dirGroups[dir] = append(dirGroups[dir], change)
	}

	var groups []*ChangeGroup
	for dir, groupChanges := range dirGroups {
		group := &ChangeGroup{
			ID:          fmt.Sprintf("dir_%s", strings.ReplaceAll(dir, "/", "_")),
			Type:        ca.determineGroupType(groupChanges),
			Description: ca.generateDescription(groupChanges),
			Changes:     groupChanges,
			Score:       ca.calculateGroupScore(groupChanges),
			Rationale:   fmt.Sprintf("Grouped by directory: %s", dir),
		}
		groups = append(groups, group)
	}

	return groups
}

// refineGroups refines the groups based on additional analysis
func (ca *ChangeAnalyzer) refineGroups(groups []*ChangeGroup, analysisResult *analysis.AnalysisResult) []*ChangeGroup {
	// Merge small groups with similar types
	groups = ca.mergeSmallGroups(groups)

	// Split large groups
	groups = ca.splitLargeGroups(groups)

	// Optimize group composition
	groups = ca.optimizeGroups(groups, analysisResult)

	return groups
}

// mergeSmallGroups merges groups that are too small
func (ca *ChangeAnalyzer) mergeSmallGroups(groups []*ChangeGroup) []*ChangeGroup {
	var refined []*ChangeGroup
	var smallGroups []*ChangeGroup

	for _, group := range groups {
		if len(group.Changes) < ca.config.MinGroupSize {
			smallGroups = append(smallGroups, group)
		} else {
			refined = append(refined, group)
		}
	}

	if len(smallGroups) > 0 {
		// Merge small groups by type
		typeGroups := make(map[GroupType][]*ChangeGroup)
		for _, group := range smallGroups {
			typeGroups[group.Type] = append(typeGroups[group.Type], group)
		}

		for groupType, groups := range typeGroups {
			var allChanges []FileChange
			var descriptions []string
			for _, group := range groups {
				allChanges = append(allChanges, group.Changes...)
				descriptions = append(descriptions, group.Description)
			}

			mergedGroup := &ChangeGroup{
				ID:          fmt.Sprintf("merged_%s", groupType.String()),
				Type:        groupType,
				Description: strings.Join(descriptions, "; "),
				Changes:     allChanges,
				Score:       ca.calculateGroupScore(allChanges),
				Rationale:   "Merged small groups of same type",
			}
			refined = append(refined, mergedGroup)
		}
	}

	return refined
}

// splitLargeGroups splits groups that are too large
func (ca *ChangeAnalyzer) splitLargeGroups(groups []*ChangeGroup) []*ChangeGroup {
	var refined []*ChangeGroup

	for _, group := range groups {
		if len(group.Changes) > ca.config.MaxGroupSize {
			subGroups := ca.splitGroupWithSize(group, ca.config.MaxGroupSize)
			refined = append(refined, subGroups...)
		} else {
			refined = append(refined, group)
		}
	}

	return refined
}

// splitGroup splits a large group into smaller groups using MaxGroupSize
func (ca *ChangeAnalyzer) splitGroup(group *ChangeGroup) []*ChangeGroup {
	return ca.splitGroupWithSize(group, ca.config.MaxGroupSize)
}

// splitGroupWithSize splits a large group into smaller groups with specified max size
func (ca *ChangeAnalyzer) splitGroupWithSize(group *ChangeGroup, maxSize int) []*ChangeGroup {
	var subGroups []*ChangeGroup
	changes := group.Changes

	// Split by file type/extension
	extGroups := make(map[string][]FileChange)
	for _, change := range changes {
		ext := filepath.Ext(change.Path)
		if ext == "" {
			ext = "no_ext"
		}
		extGroups[ext] = append(extGroups[ext], change)
	}

	groupIndex := 1
	for ext, extChanges := range extGroups {
		// Further split if still too large
		for len(extChanges) > 0 {
			size := maxSize
			if len(extChanges) < size {
				size = len(extChanges)
			}

			subGroupChanges := extChanges[:size]
			extChanges = extChanges[size:]

			subGroup := &ChangeGroup{
				ID:          fmt.Sprintf("%s_split_%d", group.ID, groupIndex),
				Type:        ca.determineGroupType(subGroupChanges),
				Description: ca.generateDescription(subGroupChanges),
				Changes:     subGroupChanges,
				Score:       ca.calculateGroupScore(subGroupChanges),
				Rationale:   fmt.Sprintf("Split from large group by extension: %s", ext),
			}
			subGroups = append(subGroups, subGroup)
			groupIndex++
		}
	}

	return subGroups
}

// optimizeGroups optimizes group composition for better commit organization
func (ca *ChangeAnalyzer) optimizeGroups(groups []*ChangeGroup, analysisResult *analysis.AnalysisResult) []*ChangeGroup {
	// Sort groups by score (higher score = better grouping)
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Score > groups[j].Score
	})

	return groups
}

// adjustGroupSizes ensures groups are within acceptable size limits
func (ca *ChangeAnalyzer) adjustGroupSizes(groups []*ChangeGroup) []*ChangeGroup {
	var adjusted []*ChangeGroup

	for _, group := range groups {
		if len(group.Changes) > ca.config.MaxCommitSize {
			// Split oversized groups
			split := ca.splitGroupWithSize(group, ca.config.MaxCommitSize)
			adjusted = append(adjusted, split...)
		} else {
			adjusted = append(adjusted, group)
		}
	}

	return adjusted
}

// Helper methods

func (ca *ChangeAnalyzer) determineGroupType(changes []FileChange) GroupType {
	// Count different indicators
	testCount := 0
	docCount := 0
	configCount := 0
	featureCount := 0
	fixCount := 0

	for _, change := range changes {
		path := strings.ToLower(change.Path)

		switch {
		case ca.isTestFile(path):
			testCount++
		case ca.isDocFile(path):
			docCount++
		case ca.isConfigFile(path):
			configCount++
		case change.Type == ChangeTypeAdd:
			featureCount++
		case strings.Contains(path, "fix") || strings.Contains(path, "bug"):
			fixCount++
		}
	}

	// Determine dominant type
	total := len(changes)
	if testCount > total/2 {
		return GroupTypeTest
	}
	if docCount > total/2 {
		return GroupTypeDocs
	}
	if configCount > total/2 {
		return GroupTypeConfig
	}
	if fixCount > 0 {
		return GroupTypeFix
	}
	if featureCount > 0 {
		return GroupTypeFeature
	}

	return GroupTypeMixed
}

func (ca *ChangeAnalyzer) generateDescription(changes []FileChange) string {
	if len(changes) == 1 {
		return fmt.Sprintf("Update %s", changes[0].Path)
	}

	groupType := ca.determineGroupType(changes)
	switch groupType {
	case GroupTypeTest:
		return fmt.Sprintf("Update tests (%d files)", len(changes))
	case GroupTypeDocs:
		return fmt.Sprintf("Update documentation (%d files)", len(changes))
	case GroupTypeConfig:
		return fmt.Sprintf("Update configuration (%d files)", len(changes))
	case GroupTypeFeature:
		return fmt.Sprintf("Add new features (%d files)", len(changes))
	case GroupTypeFix:
		return fmt.Sprintf("Fix issues (%d files)", len(changes))
	default:
		return fmt.Sprintf("Update %d files", len(changes))
	}
}

func (ca *ChangeAnalyzer) calculateGroupScore(changes []FileChange) float64 {
	if len(changes) == 0 {
		return 0.0
	}

	score := 1.0

	// Bonus for similar file types
	extMap := make(map[string]int)
	for _, change := range changes {
		ext := filepath.Ext(change.Path)
		extMap[ext]++
	}

	if len(extMap) == 1 {
		score += 0.3 // Bonus for same file type
	} else if len(extMap) <= 3 {
		score += 0.1 // Small bonus for few file types
	}

	// Bonus for similar directories
	dirMap := make(map[string]int)
	for _, change := range changes {
		dir := filepath.Dir(change.Path)
		dirMap[dir]++
	}

	if len(dirMap) == 1 {
		score += 0.2 // Bonus for same directory
	}

	// Penalty for mixed change types
	typeMap := make(map[ChangeType]int)
	for _, change := range changes {
		typeMap[change.Type]++
	}

	if len(typeMap) > 2 {
		score -= 0.2 // Penalty for many different change types
	}

	// Optimal size bonus
	optimalSize := ca.config.MaxCommitSize / 4
	if len(changes) <= optimalSize {
		score += 0.1
	}

	return score
}

func (ca *ChangeAnalyzer) extractModule(path string, analysisResult *analysis.AnalysisResult) string {
	// Extract module based on directory structure
	parts := strings.Split(path, "/")
	if len(parts) > 1 {
		return parts[0]
	}
	return "root"
}

func (ca *ChangeAnalyzer) detectFeature(change FileChange, analysisResult *analysis.AnalysisResult) string {
	// Simple feature detection based on file patterns
	path := strings.ToLower(change.Path)

	// Check for common feature indicators
	if strings.Contains(path, "auth") {
		return "authentication"
	}
	if strings.Contains(path, "user") {
		return "user_management"
	}
	if strings.Contains(path, "api") {
		return "api"
	}
	if strings.Contains(path, "ui") || strings.Contains(path, "view") {
		return "ui"
	}
	if strings.Contains(path, "db") || strings.Contains(path, "database") {
		return "database"
	}
	if strings.Contains(path, "config") {
		return "configuration"
	}

	// Use directory as feature
	dir := filepath.Dir(change.Path)
	if dir != "." {
		return strings.ReplaceAll(dir, "/", "_")
	}

	return "core"
}

func (ca *ChangeAnalyzer) isTestFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, "test") ||
		strings.Contains(lower, "spec") ||
		strings.HasSuffix(lower, "_test.go") ||
		strings.HasSuffix(lower, ".test.js") ||
		strings.HasSuffix(lower, ".spec.js")
}

func (ca *ChangeAnalyzer) isDocFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".md") ||
		strings.HasSuffix(lower, ".rst") ||
		strings.HasSuffix(lower, ".txt") ||
		strings.Contains(lower, "readme") ||
		strings.Contains(lower, "doc") ||
		strings.Contains(lower, "manual")
}

func (ca *ChangeAnalyzer) isConfigFile(path string) bool {
	lower := strings.ToLower(path)
	configExts := []string{".json", ".yaml", ".yml", ".toml", ".ini", ".conf", ".config"}

	for _, ext := range configExts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}

	return strings.Contains(lower, "config") ||
		strings.Contains(lower, ".env") ||
		strings.Contains(lower, "dockerfile") ||
		strings.Contains(lower, "makefile")
}

// String methods for enums

func (gt GroupType) String() string {
	switch gt {
	case GroupTypeFeature:
		return "feature"
	case GroupTypeFix:
		return "fix"
	case GroupTypeRefactor:
		return "refactor"
	case GroupTypeTest:
		return "test"
	case GroupTypeDocs:
		return "docs"
	case GroupTypeConfig:
		return "config"
	case GroupTypeStyle:
		return "style"
	case GroupTypeMixed:
		return "mixed"
	default:
		return statusUnknown
	}
}

func (ct ChangeType) String() string {
	switch ct {
	case ChangeTypeAdd:
		return "add"
	case ChangeTypeModify:
		return "modify"
	case ChangeTypeDelete:
		return "delete"
	case ChangeTypeRename:
		return "rename"
	case ChangeTypeCopy:
		return "copy"
	case ChangeTypeUntracked:
		return "untracked"
	default:
		return statusUnknown
	}
}
