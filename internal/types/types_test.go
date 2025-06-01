package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueReference(t *testing.T) {
	tests := []struct {
		name     string
		ref      IssueReference
		expected IssueReference
	}{
		{
			name: "valid issue reference",
			ref: IssueReference{
				Owner:  "owner",
				Repo:   "repo",
				Number: 123,
				Source: "url",
			},
			expected: IssueReference{
				Owner:  "owner",
				Repo:   "repo",
				Number: 123,
				Source: "url",
			},
		},
		{
			name: "shorthand reference",
			ref: IssueReference{
				Owner:  "org",
				Repo:   "project",
				Number: 456,
				Source: "shorthand",
			},
			expected: IssueReference{
				Owner:  "org",
				Repo:   "project",
				Number: 456,
				Source: "shorthand",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected.Owner, tt.ref.Owner)
			assert.Equal(t, tt.expected.Repo, tt.ref.Repo)
			assert.Equal(t, tt.expected.Number, tt.ref.Number)
			assert.Equal(t, tt.expected.Source, tt.ref.Source)
		})
	}
}

func TestIssueData(t *testing.T) {
	now := time.Now()

	issueData := IssueData{
		Number:           42,
		Title:            "Test Issue",
		Body:             "Issue body content",
		Description:      "Issue description",
		State:            "open",
		Repository:       "test/repo",
		Labels:           []string{"bug", "enhancement"},
		Assignees:        []string{"user1", "user2"},
		Milestone:        "v1.0.0",
		Comments:         []IssueComment{{ID: 1, Body: "Comment", Author: "user1", CreatedAt: now}},
		WorkingDirectory: "/path/to/work",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Test JSON serialization
	data, err := json.Marshal(issueData)
	require.NoError(t, err)

	var unmarshaled IssueData
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, issueData.Number, unmarshaled.Number)
	assert.Equal(t, issueData.Title, unmarshaled.Title)
	assert.Equal(t, issueData.Body, unmarshaled.Body)
	assert.Equal(t, issueData.State, unmarshaled.State)
	assert.Equal(t, issueData.Repository, unmarshaled.Repository)
	assert.Equal(t, issueData.Labels, unmarshaled.Labels)
	assert.Equal(t, issueData.Assignees, unmarshaled.Assignees)
	assert.Equal(t, issueData.Milestone, unmarshaled.Milestone)
	assert.Len(t, unmarshaled.Comments, 1)
	assert.Equal(t, issueData.WorkingDirectory, unmarshaled.WorkingDirectory)
}

func TestComment(t *testing.T) {
	now := time.Now()

	comment := Comment{
		ID:        123,
		Body:      "This is a comment",
		Author:    "testuser",
		CreatedAt: now,
	}

	// Test JSON serialization
	data, err := json.Marshal(comment)
	require.NoError(t, err)

	var unmarshaled Comment
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, comment.ID, unmarshaled.ID)
	assert.Equal(t, comment.Body, unmarshaled.Body)
	assert.Equal(t, comment.Author, unmarshaled.Author)
	assert.WithinDuration(t, comment.CreatedAt, unmarshaled.CreatedAt, time.Second)
}

func TestIssueCommentAlias(t *testing.T) {
	// Test that IssueComment is properly aliased to Comment
	comment := Comment{ID: 1, Body: "test", Author: "user"}
	issueComment := IssueComment(comment)

	assert.Equal(t, comment.ID, issueComment.ID)
	assert.Equal(t, comment.Body, issueComment.Body)
	assert.Equal(t, comment.Author, issueComment.Author)
}

func TestCodeContext(t *testing.T) {
	ctx := CodeContext{
		ProjectSummary: &ProjectSummaryInfo{
			Name:        "Test Project",
			Description: "A test project",
			Version:     "1.0.0",
		},
		TechnicalStack: &TechnicalStackInfo{
			Languages: []LanguageInfo{
				{Name: "Go", Version: "1.21", Percentage: 90.0},
			},
		},
		Architecture: &ArchitectureInfo{
			DesignPatterns: []string{"MVC", "Repository"},
		},
		RelevantFiles: []RelevantFileInfo{
			{Path: "main.go", Purpose: "entry point", Language: "go"},
		},
	}

	// Test JSON serialization
	data, err := json.Marshal(ctx)
	require.NoError(t, err)

	var unmarshaled CodeContext
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, ctx.ProjectSummary.Name, unmarshaled.ProjectSummary.Name)
	assert.Equal(t, ctx.TechnicalStack.Languages[0].Name, unmarshaled.TechnicalStack.Languages[0].Name)
	assert.Equal(t, ctx.Architecture.DesignPatterns, unmarshaled.Architecture.DesignPatterns)
	assert.Len(t, unmarshaled.RelevantFiles, 1)
}

func TestProjectSummaryInfo(t *testing.T) {
	summary := ProjectSummaryInfo{
		Name:                 "ccAgents",
		Description:          "AI-powered GitHub automation tool",
		Version:              "2.0.0",
		License:              "MIT",
		MainLanguage:         "Go",
		ProjectType:          "CLI Tool",
		BuildSystem:          "Go Modules",
		PackageManager:       "go mod",
		Tags:                 []string{"ai", "automation", "github"},
		LanguageDistribution: map[string]float64{"Go": 95.5, "YAML": 4.5},
		PrimaryFrameworks:    []string{"Cobra", "Bubble Tea"},
		TotalFiles:           92,
		TotalLines:           15000,
		ProjectSize:          1024000,
	}

	// Test JSON serialization
	data, err := json.Marshal(summary)
	require.NoError(t, err)

	var unmarshaled ProjectSummaryInfo
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, summary.Name, unmarshaled.Name)
	assert.Equal(t, summary.Description, unmarshaled.Description)
	assert.Equal(t, summary.LanguageDistribution["Go"], unmarshaled.LanguageDistribution["Go"])
	assert.Equal(t, summary.PrimaryFrameworks, unmarshaled.PrimaryFrameworks)
	assert.Equal(t, summary.TotalFiles, unmarshaled.TotalFiles)
}

func TestLanguageInfo(t *testing.T) {
	lang := LanguageInfo{
		Name:        "Go",
		Version:     "1.21.5",
		Percentage:  85.7,
		FileCount:   45,
		LineCount:   8500,
		Extensions:  []string{".go"},
		ConfigFiles: []string{"go.mod", "go.sum"},
	}

	data, err := json.Marshal(lang)
	require.NoError(t, err)

	var unmarshaled LanguageInfo
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, lang.Name, unmarshaled.Name)
	assert.Equal(t, lang.Version, unmarshaled.Version)
	assert.Equal(t, lang.Percentage, unmarshaled.Percentage)
	assert.Equal(t, lang.FileCount, unmarshaled.FileCount)
	assert.Equal(t, lang.LineCount, unmarshaled.LineCount)
	assert.Equal(t, lang.Extensions, unmarshaled.Extensions)
	assert.Equal(t, lang.ConfigFiles, unmarshaled.ConfigFiles)
}

func TestFrameworkInfo(t *testing.T) {
	framework := FrameworkInfo{
		Name:    "Cobra",
		Version: "v1.8.0",
		Type:    "CLI Framework",
		Purpose: "Command line interface construction",
	}

	data, err := json.Marshal(framework)
	require.NoError(t, err)

	var unmarshaled FrameworkInfo
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, framework.Name, unmarshaled.Name)
	assert.Equal(t, framework.Version, unmarshaled.Version)
	assert.Equal(t, framework.Type, unmarshaled.Type)
	assert.Equal(t, framework.Purpose, unmarshaled.Purpose)
}

func TestRelevantFileInfo(t *testing.T) {
	now := time.Now()

	fileInfo := RelevantFileInfo{
		Path:         "pkg/agents/base_agent.go",
		Purpose:      "base agent implementation",
		Language:     "go",
		Size:         2048,
		LastModified: now,
		Importance:   0.95,
		Summary:      "Contains base agent interface and common functionality",
	}

	data, err := json.Marshal(fileInfo)
	require.NoError(t, err)

	var unmarshaled RelevantFileInfo
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, fileInfo.Path, unmarshaled.Path)
	assert.Equal(t, fileInfo.Purpose, unmarshaled.Purpose)
	assert.Equal(t, fileInfo.Language, unmarshaled.Language)
	assert.Equal(t, fileInfo.Size, unmarshaled.Size)
	assert.Equal(t, fileInfo.Importance, unmarshaled.Importance)
	assert.Equal(t, fileInfo.Summary, unmarshaled.Summary)
}

func TestFileContext(t *testing.T) {
	now := time.Now()

	fileCtx := FileContext{
		Path:         "main.go",
		Content:      "package main\n\nfunc main() {}\n",
		Language:     "go",
		Size:         256,
		LastModified: now,
	}

	data, err := json.Marshal(fileCtx)
	require.NoError(t, err)

	var unmarshaled FileContext
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, fileCtx.Path, unmarshaled.Path)
	assert.Equal(t, fileCtx.Content, unmarshaled.Content)
	assert.Equal(t, fileCtx.Language, unmarshaled.Language)
	assert.Equal(t, fileCtx.Size, unmarshaled.Size)
}

func TestPRData(t *testing.T) {
	now := time.Now()

	pr := PRData{
		Number:     42,
		Title:      "Add new feature",
		Body:       "This PR adds a new feature",
		State:      "open",
		Branch:     "feature/new-feature",
		BaseBranch: "main",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	data, err := json.Marshal(pr)
	require.NoError(t, err)

	var unmarshaled PRData
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, pr.Number, unmarshaled.Number)
	assert.Equal(t, pr.Title, unmarshaled.Title)
	assert.Equal(t, pr.Body, unmarshaled.Body)
	assert.Equal(t, pr.State, unmarshaled.State)
	assert.Equal(t, pr.Branch, unmarshaled.Branch)
	assert.Equal(t, pr.BaseBranch, unmarshaled.BaseBranch)
}

func TestWorkflowState(t *testing.T) {
	tests := []struct {
		name  string
		state WorkflowState
		value int
	}{
		{"idle", WorkflowIdle, 0},
		{"running", WorkflowRunning, 1},
		{"paused", WorkflowPaused, 2},
		{"completed", WorkflowCompleted, 3},
		{"failed", WorkflowFailed, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, int(tt.state))
		})
	}
}

func TestStageStatus(t *testing.T) {
	tests := []struct {
		name   string
		status StageStatus
		value  int
	}{
		{"pending", StagePending, 0},
		{"running", StageRunning, 1},
		{"completed", StageCompleted, 2},
		{"failed", StageFailed, 3},
		{"skipped", StageSkipped, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, int(tt.status))
		})
	}
}

func TestStage(t *testing.T) {
	now := time.Now()

	stage := Stage{
		Name:         "Build",
		Status:       StageRunning,
		Output:       []string{"Building...", "Success"},
		Dependencies: []string{"Test"},
		RetryCount:   1,
		MaxRetries:   3,
		StartTime:    now,
		EndTime:      now.Add(time.Minute),
	}

	assert.Equal(t, "Build", stage.Name)
	assert.Equal(t, StageRunning, stage.Status)
	assert.Equal(t, []string{"Building...", "Success"}, stage.Output)
	assert.Equal(t, []string{"Test"}, stage.Dependencies)
	assert.Equal(t, 1, stage.RetryCount)
	assert.Equal(t, 3, stage.MaxRetries)
	assert.Equal(t, now, stage.StartTime)
	assert.Equal(t, now.Add(time.Minute), stage.EndTime)
}

func TestProcessState(t *testing.T) {
	now := time.Now()

	process := ProcessState{
		ID:        "proc-123",
		Command:   "go",
		Args:      []string{"test", "./..."},
		Status:    "running",
		Output:    []string{"Running tests..."},
		ExitCode:  0,
		StartTime: now,
		EndTime:   now.Add(time.Second * 30),
	}

	assert.Equal(t, "proc-123", process.ID)
	assert.Equal(t, "go", process.Command)
	assert.Equal(t, []string{"test", "./..."}, process.Args)
	assert.Equal(t, "running", process.Status)
	assert.Equal(t, []string{"Running tests..."}, process.Output)
	assert.Equal(t, 0, process.ExitCode)
	assert.Equal(t, now, process.StartTime)
	assert.Equal(t, now.Add(time.Second*30), process.EndTime)
}

func TestIssueContextInfo(t *testing.T) {
	issueContext := IssueContextInfo{
		IssueNumber:         123,
		Title:               "Add user authentication",
		Description:         "Implement user authentication system",
		Labels:              []string{"feature", "security"},
		Requirements:        []string{"JWT tokens", "Password hashing"},
		AcceptanceCriteria:  []string{"User can login", "Secure password storage"},
		MentionedFiles:      []string{"auth.go", "user.go"},
		RelatedIssues:       []int{124, 125},
		Complexity:          "Medium",
		EstimatedEffort:     time.Hour * 8,
		RiskFactors:         []string{"Security implications"},
		TestStrategy:        "Unit and integration tests",
		ImplementationHints: []string{"Use bcrypt for hashing"},
	}

	data, err := json.Marshal(issueContext)
	require.NoError(t, err)

	var unmarshaled IssueContextInfo
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, issueContext.IssueNumber, unmarshaled.IssueNumber)
	assert.Equal(t, issueContext.Title, unmarshaled.Title)
	assert.Equal(t, issueContext.Labels, unmarshaled.Labels)
	assert.Equal(t, issueContext.Requirements, unmarshaled.Requirements)
	assert.Equal(t, issueContext.Complexity, unmarshaled.Complexity)
	assert.Equal(t, issueContext.EstimatedEffort, unmarshaled.EstimatedEffort)
}

func TestContextMetadata(t *testing.T) {
	now := time.Now()

	metadata := ContextMetadata{
		GeneratedAt:      now,
		AnalysisVersion:  "2.0.0",
		ContextVersion:   "1.5.0",
		AnalysisDuration: time.Minute * 2,
		FilesAnalyzed:    92,
		DataSources:      []string{"git", "github", "filesystem"},
		Confidence:       0.95,
		Completeness:     0.87,
	}

	data, err := json.Marshal(metadata)
	require.NoError(t, err)

	var unmarshaled ContextMetadata
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, metadata.AnalysisVersion, unmarshaled.AnalysisVersion)
	assert.Equal(t, metadata.ContextVersion, unmarshaled.ContextVersion)
	assert.Equal(t, metadata.AnalysisDuration, unmarshaled.AnalysisDuration)
	assert.Equal(t, metadata.FilesAnalyzed, unmarshaled.FilesAnalyzed)
	assert.Equal(t, metadata.DataSources, unmarshaled.DataSources)
	assert.Equal(t, metadata.Confidence, unmarshaled.Confidence)
	assert.Equal(t, metadata.Completeness, unmarshaled.Completeness)
}

func TestComplexStruct_DependencyContextInfo(t *testing.T) {
	depContext := DependencyContextInfo{
		DirectDependencies:  15,
		DevDependencies:     8,
		TotalDependencies:   23,
		SecurityIssues:      0,
		LicenseDistribution: map[string]int{"MIT": 18, "Apache-2.0": 5},
		UpdatesAvailable:    3,
		KeyDependencies:     []string{"github.com/spf13/cobra", "github.com/stretchr/testify"},
		DependencyConflicts: 0,
	}

	data, err := json.Marshal(depContext)
	require.NoError(t, err)

	var unmarshaled DependencyContextInfo
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, depContext.DirectDependencies, unmarshaled.DirectDependencies)
	assert.Equal(t, depContext.DevDependencies, unmarshaled.DevDependencies)
	assert.Equal(t, depContext.TotalDependencies, unmarshaled.TotalDependencies)
	assert.Equal(t, depContext.SecurityIssues, unmarshaled.SecurityIssues)
	assert.Equal(t, depContext.LicenseDistribution, unmarshaled.LicenseDistribution)
	assert.Equal(t, depContext.UpdatesAvailable, unmarshaled.UpdatesAvailable)
	assert.Equal(t, depContext.KeyDependencies, unmarshaled.KeyDependencies)
	assert.Equal(t, depContext.DependencyConflicts, unmarshaled.DependencyConflicts)
}

func TestComplexStruct_TestingStrategyInfo(t *testing.T) {
	testStrategy := TestingStrategyInfo{
		TestFrameworks:  []string{"testify", "ginkgo"},
		TestDirectories: []string{"test/", "pkg/*/test/"},
		CoverageTools:   []string{"go test -cover"},
		TestPatterns:    []string{"*_test.go"},
		TestingApproach: "unit + integration",
		MockingStrategy: "interfaces with testify/mock",
		E2ETestingTools: []string{"docker-compose"},
	}

	data, err := json.Marshal(testStrategy)
	require.NoError(t, err)

	var unmarshaled TestingStrategyInfo
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, testStrategy.TestFrameworks, unmarshaled.TestFrameworks)
	assert.Equal(t, testStrategy.TestDirectories, unmarshaled.TestDirectories)
	assert.Equal(t, testStrategy.CoverageTools, unmarshaled.CoverageTools)
	assert.Equal(t, testStrategy.TestPatterns, unmarshaled.TestPatterns)
	assert.Equal(t, testStrategy.TestingApproach, unmarshaled.TestingApproach)
	assert.Equal(t, testStrategy.MockingStrategy, unmarshaled.MockingStrategy)
	assert.Equal(t, testStrategy.E2ETestingTools, unmarshaled.E2ETestingTools)
}
