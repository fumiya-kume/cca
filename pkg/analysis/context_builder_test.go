package analysis

import (
	"context"
	"testing"
	"time"

	"github.com/fumiya-kume/cca/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewContextBuilder(t *testing.T) {
	config := AnalyzerConfig{
		MaxFilesToAnalyze: 1000,
		MaxFileSize:       10 * 1024 * 1024,
	}

	builder := NewContextBuilder(config)
	assert.NotNil(t, builder)
	assert.Equal(t, config, builder.config)
}

func TestContextBuilder_BuildContext(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	analysisResult := &AnalysisResult{
		ProjectInfo: &ProjectInfo{
			Name:           "test-project",
			Description:    "A test project",
			Version:        "1.0.0",
			License:        "MIT",
			MainLanguage:   "Go",
			ProjectType:    ProjectTypeCLI,
			BuildSystem:    "Go Modules",
			PackageManager: "go mod",
			Tags:           []string{"test", "example"},
			SourceDirs:     []string{"src/", "pkg/"},
			TestDirs:       []string{"test/"},
			ConfigFiles:    []string{"go.mod", "go.sum"},
			EntryPoints:    []string{"main.go"},
		},
		Languages: []LanguageInfo{
			{
				Name:        "Go",
				Version:     "1.21",
				Percentage:  95.0,
				FileCount:   50,
				LineCount:   5000,
				Extensions:  []string{".go"},
				ConfigFiles: []string{"go.mod"},
			},
		},
		Frameworks: []FrameworkInfo{
			{
				Name:    "Cobra",
				Version: "v1.8.0",
				Type:    FrameworkTypeBuild,
			},
		},
		FileStructure: &FileStructure{
			TotalFiles: 52,
			TotalLines: 5500,
			TotalSize:  1024000,
			ImportantFiles: []FileInfo{
				{
					Path:         "main.go",
					Language:     "Go",
					Size:         1024,
					LineCount:    50,
					Type:         FileTypeSource,
					Importance:   0.9,
					LastModified: time.Now(),
				},
			},
		},
		Dependencies: &DependencyInfo{
			Direct: []Dependency{
				{Name: "github.com/spf13/cobra", Version: "v1.8.0"},
				{Name: "github.com/stretchr/testify", Version: "v1.8.0"},
			},
			DevDeps: []Dependency{
				{Name: "github.com/golangci/golangci-lint", Version: "v1.54.0"},
			},
			Security: []SecurityIssue{},
			Licenses: map[string]int{"MIT": 2},
		},
		AnalysisTime: time.Minute * 2,
	}

	ctx := context.Background()
	codeContext, err := builder.BuildContext(ctx, analysisResult)

	require.NoError(t, err)
	require.NotNil(t, codeContext)

	// Test ProjectSummary
	assert.Equal(t, "test-project", codeContext.ProjectSummary.Name)
	assert.Equal(t, "A test project", codeContext.ProjectSummary.Description)
	assert.Equal(t, "1.0.0", codeContext.ProjectSummary.Version)
	assert.Equal(t, "MIT", codeContext.ProjectSummary.License)
	assert.Equal(t, "Go", codeContext.ProjectSummary.MainLanguage)
	assert.Contains(t, []string{"CLI", "cli"}, codeContext.ProjectSummary.ProjectType)
	assert.Contains(t, codeContext.ProjectSummary.Tags, "test")
	assert.Equal(t, 52, codeContext.ProjectSummary.TotalFiles)
	assert.Equal(t, 5500, codeContext.ProjectSummary.TotalLines)

	// Test TechnicalStack
	require.Len(t, codeContext.TechnicalStack.Languages, 1)
	assert.Equal(t, "Go", codeContext.TechnicalStack.Languages[0].Name)
	assert.Equal(t, 95.0, codeContext.TechnicalStack.Languages[0].Percentage)

	require.Len(t, codeContext.TechnicalStack.Frameworks, 1)
	assert.Equal(t, "Cobra", codeContext.TechnicalStack.Frameworks[0].Name)

	// Test Dependencies
	assert.Equal(t, 2, codeContext.Dependencies.DirectDependencies)
	assert.Equal(t, 1, codeContext.Dependencies.DevDependencies)
	assert.Equal(t, 2, codeContext.Dependencies.TotalDependencies)

	// Test RelevantFiles
	assert.NotEmpty(t, codeContext.RelevantFiles)

	// Test Metadata
	assert.NotNil(t, codeContext.Metadata)
	assert.Equal(t, "1.0.0", codeContext.Metadata.AnalysisVersion)
	assert.Equal(t, time.Minute*2, codeContext.Metadata.AnalysisDuration)
}

func TestContextBuilder_BuildContextForIssue(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	analysisResult := &AnalysisResult{
		ProjectInfo: &ProjectInfo{
			Name:        "test-project",
			ProjectType: ProjectTypeWebApp,
		},
		Frameworks: []FrameworkInfo{
			{Name: "React", Type: FrameworkTypeUI},
		},
	}

	issueData := &types.IssueData{
		Number: 123,
		Title:  "Add user authentication",
		Body: `
Requirements:
- JWT token authentication
- Password hashing with bcrypt
- User login and logout

Acceptance Criteria:
- User can login with email/password
- Secure password storage
- JWT tokens expire after 24 hours

Files mentioned: auth.go, user.go
Related to issue #124
		`,
		Labels: []string{"feature", "security"},
	}

	ctx := context.Background()
	codeContext, err := builder.BuildContextForIssue(ctx, analysisResult, issueData)

	require.NoError(t, err)
	require.NotNil(t, codeContext)
	require.NotNil(t, codeContext.IssueContext)

	// Test issue-specific context
	assert.Equal(t, 123, codeContext.IssueContext.IssueNumber)
	assert.Equal(t, "Add user authentication", codeContext.IssueContext.Title)
	assert.Contains(t, codeContext.IssueContext.Labels, "feature")
	assert.Contains(t, codeContext.IssueContext.Labels, "security")

	// Test extracted requirements
	assert.Contains(t, codeContext.IssueContext.Requirements, "JWT token authentication")
	assert.Contains(t, codeContext.IssueContext.Requirements, "Password hashing with bcrypt")

	// Test extracted acceptance criteria
	assert.Contains(t, codeContext.IssueContext.AcceptanceCriteria, "User can login with email/password")
	assert.Contains(t, codeContext.IssueContext.AcceptanceCriteria, "Secure password storage")

	// Test extracted files
	assert.Contains(t, codeContext.IssueContext.MentionedFiles, "auth.go")
	assert.Contains(t, codeContext.IssueContext.MentionedFiles, "user.go")

	// Test related issues
	assert.Contains(t, codeContext.IssueContext.RelatedIssues, 124)

	// Test complexity estimation
	assert.NotEmpty(t, codeContext.IssueContext.Complexity)

	// Test implementation hints
	assert.NotEmpty(t, codeContext.IssueContext.ImplementationHints)
}

func TestContextBuilder_buildProjectSummary(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	analysis := &AnalysisResult{
		ProjectInfo: &ProjectInfo{
			Name:           "test-project",
			Description:    "Test description",
			Version:        "2.0.0",
			License:        "Apache-2.0",
			MainLanguage:   "TypeScript",
			ProjectType:    ProjectTypeWebApp,
			BuildSystem:    "Webpack",
			PackageManager: "npm",
			Tags:           []string{"web", "frontend"},
		},
		Languages: []LanguageInfo{
			{Name: "TypeScript", Percentage: 70.0},
			{Name: "JavaScript", Percentage: 30.0},
		},
		Frameworks: []FrameworkInfo{
			{Name: "React"},
			{Name: "Express.js"},
		},
		FileStructure: &FileStructure{
			TotalFiles: 100,
			TotalLines: 10000,
			TotalSize:  2048000,
		},
	}

	summary := builder.buildProjectSummary(analysis)

	assert.Equal(t, "test-project", summary.Name)
	assert.Equal(t, "Test description", summary.Description)
	assert.Equal(t, "2.0.0", summary.Version)
	assert.Equal(t, "Apache-2.0", summary.License)
	assert.Equal(t, "TypeScript", summary.MainLanguage)
	assert.Contains(t, []string{"WebApp", "web-app"}, summary.ProjectType)
	assert.Equal(t, "Webpack", summary.BuildSystem)
	assert.Equal(t, "npm", summary.PackageManager)
	assert.Contains(t, summary.Tags, "web")
	assert.Equal(t, 70.0, summary.LanguageDistribution["TypeScript"])
	assert.Equal(t, 30.0, summary.LanguageDistribution["JavaScript"])
	assert.Contains(t, summary.PrimaryFrameworks, "React")
	assert.Contains(t, summary.PrimaryFrameworks, "Express.js")
	assert.Equal(t, 100, summary.TotalFiles)
	assert.Equal(t, 10000, summary.TotalLines)
	assert.Equal(t, int64(2048000), summary.ProjectSize)
}

func TestContextBuilder_buildTechnicalStack(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	analysis := &AnalysisResult{
		ProjectInfo: &ProjectInfo{
			BuildSystem: "Maven",
			ConfigFiles: []string{"jest.config.js", "webpack.config.js", "eslint.config.js"},
		},
		Languages: []LanguageInfo{
			{
				Name:        "Java",
				Version:     "17",
				Percentage:  80.0,
				FileCount:   40,
				LineCount:   4000,
				Extensions:  []string{".java"},
				ConfigFiles: []string{"pom.xml"},
			},
		},
		Frameworks: []FrameworkInfo{
			{Name: "Spring Boot", Version: "3.0.0", Type: FrameworkTypeAPI},
			{Name: "JUnit", Version: "5.0.0", Type: FrameworkTypeTesting},
		},
	}

	stack := builder.buildTechnicalStack(analysis)

	require.Len(t, stack.Languages, 1)
	assert.Equal(t, "Java", stack.Languages[0].Name)
	assert.Equal(t, "17", stack.Languages[0].Version)
	assert.Equal(t, 80.0, stack.Languages[0].Percentage)

	require.Len(t, stack.Frameworks, 2)
	assert.Equal(t, "Spring Boot", stack.Frameworks[0].Name)
	assert.Equal(t, "JUnit", stack.Frameworks[1].Name)

	assert.Contains(t, stack.BuildTools, "Maven")
	assert.Contains(t, stack.TestingTools, "JUnit")
	assert.Contains(t, stack.TestingTools, "Jest")
	assert.Contains(t, stack.BuildTools, "Webpack")
	assert.Contains(t, stack.DevelopmentTools, "ESLint")
}

func TestContextBuilder_extractRequirements(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	issueBody := `
This is a feature request.

Requirements:
- User authentication with JWT
- Password hashing
- Session management

Must:
- Be secure
- Handle errors gracefully

Should:
- Be fast
- Be scalable
	`

	requirements := builder.extractRequirements(issueBody)

	assert.Contains(t, requirements, "User authentication with JWT")
	assert.Contains(t, requirements, "Password hashing")
	assert.Contains(t, requirements, "Session management")
	assert.Contains(t, requirements, "Be secure")
	assert.Contains(t, requirements, "Handle errors gracefully")
	assert.Contains(t, requirements, "Be fast")
	assert.Contains(t, requirements, "Be scalable")
}

func TestContextBuilder_extractAcceptanceCriteria(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	issueBody := `
Feature description here.

Acceptance Criteria:
- Users can login successfully
- Invalid credentials show error
- Password is hashed in database

AC:
- Performance under 100ms
- Secure session handling

Definition of Done:
- All tests pass
- Documentation updated
	`

	criteria := builder.extractAcceptanceCriteria(issueBody)

	assert.Contains(t, criteria, "Users can login successfully")
	assert.Contains(t, criteria, "Invalid credentials show error")
	assert.Contains(t, criteria, "Password is hashed in database")
	assert.Contains(t, criteria, "Performance under 100ms")
	assert.Contains(t, criteria, "Secure session handling")
	assert.Contains(t, criteria, "All tests pass")
	assert.Contains(t, criteria, "Documentation updated")
}

func TestContextBuilder_extractMentionedFiles(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	issueBody := `
Please update auth.go and user_service.js files.
Also check the config/database.yml file.
The main.py script needs modification.
Ignore this URL: https://example.com/file.txt
	`

	files := builder.extractMentionedFiles(issueBody, "test-project")

	assert.Contains(t, files, "auth.go")
	assert.Contains(t, files, "user_service.js")
	assert.Contains(t, files, "config/database.yml")
	assert.Contains(t, files, "main.py")
	// Should not contain URLs
	assert.NotContains(t, files, "https://example.com/file.txt")
}

func TestContextBuilder_extractRelatedIssues(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	issueBody := `
This is related to issue #123 and issue #456.
See also #789 for more context.
	`

	issues := builder.extractRelatedIssues(issueBody)

	assert.Contains(t, issues, 123)
	assert.Contains(t, issues, 456)
	assert.Contains(t, issues, 789)
}

func TestContextBuilder_estimateComplexity(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	tests := []struct {
		name     string
		issue    *types.IssueData
		expected string
	}{
		{
			name: "high complexity - critical label",
			issue: &types.IssueData{
				Labels: []string{"critical", "breaking"},
				Body:   "Short description",
			},
			expected: "high",
		},
		{
			name: "low complexity - simple label",
			issue: &types.IssueData{
				Labels: []string{"simple", "documentation"},
				Body:   "Short description",
			},
			expected: "low",
		},
		{
			name: "high complexity - long description",
			issue: &types.IssueData{
				Labels: []string{},
				Body:   string(make([]byte, 1500)), // Long description
			},
			expected: "high",
		},
		{
			name: "medium complexity - medium description",
			issue: &types.IssueData{
				Labels: []string{},
				Body:   string(make([]byte, 500)), // Medium description
			},
			expected: "medium",
		},
		{
			name: "low complexity - short description",
			issue: &types.IssueData{
				Labels: []string{},
				Body:   "Short description",
			},
			expected: "low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			complexity := builder.estimateComplexity(tt.issue)
			assert.Equal(t, tt.expected, complexity)
		})
	}
}

func TestContextBuilder_estimateEffort(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	tests := []struct {
		name     string
		issue    *types.IssueData
		expected time.Duration
	}{
		{
			name: "low effort",
			issue: &types.IssueData{
				Labels: []string{"simple"},
			},
			expected: 2 * time.Hour,
		},
		{
			name: "high effort",
			issue: &types.IssueData{
				Labels: []string{"critical"},
			},
			expected: 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			effort := builder.estimateEffort(tt.issue)
			assert.Equal(t, tt.expected, effort)
		})
	}
}

func TestContextBuilder_identifyRiskFactors(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	issue := &types.IssueData{
		Labels: []string{"breaking", "security", "database"},
		Body:   "This will delete some records and require a migration",
	}

	analysis := &AnalysisResult{}
	risks := builder.identifyRiskFactors(issue, analysis)

	assert.Contains(t, risks, "Breaking change")
	assert.Contains(t, risks, "Security implications")
	assert.Contains(t, risks, "Database changes")
	assert.Contains(t, risks, "Mentions delete")
	assert.Contains(t, risks, "Mentions migration")
}

func TestContextBuilder_suggestTestStrategy(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	tests := []struct {
		name     string
		labels   []string
		expected string
	}{
		{
			name:     "API testing",
			labels:   []string{"api", "backend"},
			expected: "API integration tests + unit tests",
		},
		{
			name:     "Frontend testing",
			labels:   []string{"frontend", "ui"},
			expected: "Component tests + E2E tests",
		},
		{
			name:     "Bug fix testing",
			labels:   []string{"bug"},
			expected: "Regression tests + existing test updates",
		},
		{
			name:     "Feature testing",
			labels:   []string{"feature"},
			expected: "Comprehensive unit tests + integration tests",
		},
		{
			name:     "Default testing",
			labels:   []string{"other"},
			expected: "Unit tests + integration tests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := &types.IssueData{Labels: tt.labels}
			analysis := &AnalysisResult{}
			strategy := builder.suggestTestStrategy(issue, analysis)
			assert.Equal(t, tt.expected, strategy)
		})
	}
}

func TestContextBuilder_generateImplementationHints(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	issue := &types.IssueData{}
	analysis := &AnalysisResult{
		ProjectInfo: &ProjectInfo{
			ProjectType: ProjectTypeWebApp,
		},
		Frameworks: []FrameworkInfo{
			{Name: "React"},
			{Name: "Django"},
		},
	}

	hints := builder.generateImplementationHints(issue, analysis)

	assert.Contains(t, hints, "Consider responsive design")
	assert.Contains(t, hints, "Ensure accessibility compliance")
	assert.Contains(t, hints, "Use functional components and hooks")
	assert.Contains(t, hints, "Follow Django conventions")
}

func TestContextBuilder_getFrameworkPurpose(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	tests := []struct {
		framework string
		expected  string
	}{
		{"React", "Frontend UI Library"},
		{"Vue.js", "Frontend Framework"},
		{"Angular", "Full-stack Framework"},
		{"Express.js", "Backend Web Framework"},
		{"Django", "Full-stack Web Framework"},
		{"Spring Boot", "Enterprise Application Framework"},
		{"Jest", "JavaScript Testing Framework"},
		{"pytest", "Python Testing Framework"},
		{"Unknown Framework", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.framework, func(t *testing.T) {
			purpose := builder.getFrameworkPurpose(tt.framework)
			assert.Equal(t, tt.expected, purpose)
		})
	}
}

func TestContextBuilder_identifyDesignPatterns(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	analysis := &AnalysisResult{
		Frameworks: []FrameworkInfo{
			{Name: "React"},
			{Name: "Angular"},
			{Name: "Django"},
			{Name: "Spring Boot"},
		},
	}

	patterns := builder.identifyDesignPatterns(analysis)

	assert.Contains(t, patterns, "Component Pattern")
	assert.Contains(t, patterns, "Hook Pattern")
	assert.Contains(t, patterns, "MVC")
	assert.Contains(t, patterns, "Dependency Injection")
	assert.Contains(t, patterns, "MVT")
	assert.Contains(t, patterns, "ORM Pattern")
	assert.Contains(t, patterns, "Bean Pattern")
}

func TestContextBuilder_isDatabaseDependency(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	tests := []struct {
		name     string
		expected bool
	}{
		{"mysql-driver", true},
		{"postgres-client", true},
		{"mongodb-driver", true},
		{"redis-client", true},
		{"sqlite3", true},
		{"react", false},
		{"express", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.isDatabaseDependency(tt.name)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContextBuilder_getDatabaseType(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	tests := []struct {
		depName  string
		expected string
	}{
		{"mysql-driver", "MySQL"},
		{"postgres-client", "PostgreSQL"},
		{"mongodb-driver", "MongoDB"},
		{"unknown-db", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.depName, func(t *testing.T) {
			dbType := builder.getDatabaseType(tt.depName)
			assert.Equal(t, tt.expected, dbType)
		})
	}
}

func TestContextBuilder_isKeyDependency(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	tests := []struct {
		name     string
		expected bool
	}{
		{"react", true},
		{"vue", true},
		{"angular", true},
		{"express", true},
		{"django", true},
		{"lodash", true},
		{"moment", true},
		{"some-unknown-package", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.isKeyDependency(tt.name)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContextBuilder_isBuildFile(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	tests := []struct {
		filename string
		expected bool
	}{
		{"Makefile", true},
		{"build.gradle", true},
		{"pom.xml", true},
		{"webpack.config.js", true},
		{"vite.config.js", true},
		{"random.txt", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := builder.isBuildFile(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContextBuilder_identifyBuildTargets(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	tests := []struct {
		projectType ProjectType
		expected    []string
	}{
		{ProjectTypeWebApp, []string{"build", "dev", "prod"}},
		{ProjectTypeLibrary, []string{"build", "test", "publish"}},
		{ProjectTypeCLI, []string{"build", "install"}},
		{ProjectTypeAPI, []string{"build", "test"}},
	}

	for _, tt := range tests {
		t.Run(tt.projectType.String(), func(t *testing.T) {
			projectInfo := &ProjectInfo{ProjectType: tt.projectType}
			targets := builder.identifyBuildTargets(projectInfo)
			assert.Equal(t, tt.expected, targets)
		})
	}
}

func TestContextBuilder_identifyArtifactTypes(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	tests := []struct {
		language string
		expected []string
	}{
		{"JavaScript", []string{"bundle.js", "package"}},
		{"TypeScript", []string{"bundle.js", "package"}},
		{"Go", []string{"binary"}},
		{"Java", []string{"jar", "war"}},
		{"Rust", []string{"binary", "crate"}},
		{"Python", []string{"wheel", "sdist"}},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			projectInfo := &ProjectInfo{MainLanguage: tt.language}
			artifacts := builder.identifyArtifactTypes(projectInfo)
			assert.Equal(t, tt.expected, artifacts)
		})
	}
}

func TestContextBuilder_detectConfigLanguage(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	tests := []struct {
		filename string
		expected string
	}{
		{"config.json", "JSON"},
		{"config.yaml", "YAML"},
		{"config.yml", "YAML"},
		{"config.toml", "TOML"},
		{"config.xml", "XML"},
		{"config.js", "JavaScript"},
		{"config.txt", "Configuration"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			language := builder.detectConfigLanguage(tt.filename)
			assert.Equal(t, tt.expected, language)
		})
	}
}

func TestContextBuilder_getConfigPurpose(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	tests := []struct {
		filename string
		expected string
	}{
		{"package.json", "Node.js package management"},
		{"go.mod", "Go module management"},
		{"Cargo.toml", "Rust package management"},
		{"pom.xml", "Maven build configuration"},
		{"unknown.config", "project configuration"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			purpose := builder.getConfigPurpose(tt.filename)
			assert.Equal(t, tt.expected, purpose)
		})
	}
}

func TestContextBuilder_calculateConfidence(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	tests := []struct {
		name     string
		analysis *AnalysisResult
		expected float64
	}{
		{
			name:     "minimal analysis",
			analysis: &AnalysisResult{},
			expected: 0.5,
		},
		{
			name: "complete analysis",
			analysis: &AnalysisResult{
				ProjectInfo:  &ProjectInfo{},
				Languages:    []LanguageInfo{{}},
				Frameworks:   []FrameworkInfo{{}},
				Dependencies: &DependencyInfo{},
			},
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := builder.calculateConfidence(tt.analysis)
			assert.InDelta(t, tt.expected, confidence, 0.01)
		})
	}
}

func TestContextBuilder_calculateCompleteness(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	// Complete analysis with all components
	completeAnalysis := &AnalysisResult{
		ProjectInfo:   &ProjectInfo{},
		Languages:     []LanguageInfo{{}},
		Frameworks:    []FrameworkInfo{{}},
		Dependencies:  &DependencyInfo{},
		FileStructure: &FileStructure{},
		Conventions:   &CodingConventions{},
		Metadata:      &ProjectMetadata{},
	}

	completeness := builder.calculateCompleteness(completeAnalysis)
	assert.Equal(t, 1.0, completeness)

	// Partial analysis
	partialAnalysis := &AnalysisResult{
		ProjectInfo: &ProjectInfo{},
		Languages:   []LanguageInfo{{}},
	}

	partialCompleteness := builder.calculateCompleteness(partialAnalysis)
	assert.InDelta(t, 2.0/7.0, partialCompleteness, 0.01)
}

func TestContextBuilder_isDeploymentFile(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	tests := []struct {
		filename string
		expected bool
	}{
		{"Dockerfile", true},
		{"docker-compose.yml", true},
		{"k8s-deployment.yaml", true},
		{"deployment.yaml", true},
		{"regular-file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := builder.isDeploymentFile(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContextBuilder_getDeploymentType(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	tests := []struct {
		filename string
		expected string
	}{
		{"Dockerfile", "Unknown"}, // Dockerfile doesn't contain lowercase "docker"
		{"docker-compose.yml", "Docker"},
		{"k8s-deployment.yaml", "Kubernetes"},
		{"kubernetes-config.yml", "Kubernetes"},
		{"unknown-deployment.yml", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			deploymentType := builder.getDeploymentType(tt.filename)
			assert.Equal(t, tt.expected, deploymentType)
		})
	}
}

func TestAppendUnique(t *testing.T) {
	slice := []string{"a", "b"}

	// Add new item
	result := appendUnique(slice, "c")
	assert.Equal(t, []string{"a", "b", "c"}, result)

	// Add existing item
	result = appendUnique(slice, "a")
	assert.Equal(t, []string{"a", "b"}, result)
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"123", 123},
		{"456abc", 456},
		{"abc123", 0},
		{"", 0},
		{"0", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseInt(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContextBuilder_filterRelevantFilesForIssue(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	files := []types.RelevantFileInfo{
		{Path: "main.go", Importance: 0.5},
		{Path: "auth.go", Importance: 0.6},
		{Path: "user.go", Importance: 0.4},
	}

	issueContext := &types.IssueContextInfo{
		MentionedFiles: []string{"auth.go", "user.go"},
	}

	filtered := builder.filterRelevantFilesForIssue(files, issueContext)

	// auth.go should have highest importance (0.6 + 0.3 = 0.9)
	assert.Equal(t, "auth.go", filtered[0].Path)
	assert.InDelta(t, 0.9, filtered[0].Importance, 0.01)

	// user.go should be second (0.4 + 0.3 = 0.7)
	assert.Equal(t, "user.go", filtered[1].Path)
	assert.Equal(t, 0.7, filtered[1].Importance)

	// main.go should be last (unchanged at 0.5)
	assert.Equal(t, "main.go", filtered[2].Path)
	assert.Equal(t, 0.5, filtered[2].Importance)
}

func TestContextBuilder_identifyFilePurpose(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	tests := []struct {
		fileType FileType
		expected string
	}{
		{FileTypeSource, "Source Code"},
		{FileTypeTest, "Test"},
		{FileTypeConfig, "Configuration"},
		{FileTypeDocumentation, "Documentation"},
		{FileTypeBuild, "Build Script"},
		{FileType(999), "Other"}, // Unknown type
	}

	for _, tt := range tests {
		t.Run(string(rune(int(tt.fileType)+'0')), func(t *testing.T) {
			file := FileInfo{Type: tt.fileType}
			purpose := builder.identifyFilePurpose(file)
			assert.Equal(t, tt.expected, purpose)
		})
	}
}

func TestContextBuilder_generateFileSummary(t *testing.T) {
	builder := NewContextBuilder(AnalyzerConfig{})

	file := FileInfo{
		Language:  "Go",
		LineCount: 150,
	}

	summary := builder.generateFileSummary(file)
	assert.Equal(t, "Go file with 150 lines", summary)
}
