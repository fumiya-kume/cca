package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ProjectDetector detects project information and structure
type ProjectDetector struct {
	config AnalyzerConfig
}

// NewProjectDetector creates a new project detector
func NewProjectDetector(config AnalyzerConfig) *ProjectDetector {
	return &ProjectDetector{
		config: config,
	}
}

// DetectProjectInfo detects comprehensive project information
func (pd *ProjectDetector) DetectProjectInfo(ctx context.Context, projectRoot string) (*ProjectInfo, error) {
	info := &ProjectInfo{
		Name:              filepath.Base(projectRoot),
		ProjectType:       ProjectTypeUnknown,
		ConfigFiles:       []string{},
		EntryPoints:       []string{},
		SourceDirs:        []string{},
		TestDirs:          []string{},
		DocumentationDirs: []string{},
		Tags:              []string{},
		Metadata:          make(map[string]interface{}),
	}

	// Detect from various manifest files
	if err := pd.detectFromPackageJSON(projectRoot, info); err == nil {
		// Successfully detected Node.js project
	} else if err := pd.detectFromCargoToml(projectRoot, info); err == nil {
		// Successfully detected Rust project
	} else if err := pd.detectFromGoMod(projectRoot, info); err == nil {
		// Successfully detected Go project
	} else if err := pd.detectFromPyprojectToml(projectRoot, info); err == nil {
		// Successfully detected Python project
	} else if err := pd.detectFromPomXML(projectRoot, info); err == nil {
		// Successfully detected Java Maven project
	} else if err := pd.detectFromBuildGradle(projectRoot, info); err == nil {
		// Successfully detected Java Gradle project
	} else if err := pd.detectFromComposerJSON(projectRoot, info); err == nil {
		// Successfully detected PHP project
	} else if err := pd.detectFromGemspec(projectRoot, info); err == nil {
		// Successfully detected Ruby project
	} else {
		// Fallback to generic detection
		pd.detectGenericProject(projectRoot, info)
	}

	// Detect project structure
	pd.detectProjectStructure(projectRoot, info)

	// Detect build system
	pd.detectBuildSystem(projectRoot, info)

	// Detect entry points
	pd.detectEntryPoints(projectRoot, info)

	// Detect project type based on structure and files
	pd.inferProjectType(projectRoot, info)

	return info, nil
}

// detectFromPackageJSON detects Node.js project information
func (pd *ProjectDetector) detectFromPackageJSON(projectRoot string, info *ProjectInfo) error {
	packageFile := filepath.Join(projectRoot, "package.json")
	if !fileExists(packageFile) {
		return fmt.Errorf("package.json not found")
	}

	// #nosec G304 - packageFile is constructed from validated projectRoot, not user input
	data, err := os.ReadFile(packageFile)
	if err != nil {
		return err
	}

	var pkg struct {
		Name            string            `json:"name"`
		Version         string            `json:"version"`
		Description     string            `json:"description"`
		License         string            `json:"license"`
		Homepage        string            `json:"homepage"`
		Repository      interface{}       `json:"repository"`
		Main            string            `json:"main"`
		Scripts         map[string]string `json:"scripts"`
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
		Keywords        []string          `json:"keywords"`
		Author          interface{}       `json:"author"`
		Private         bool              `json:"private"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return err
	}

	info.Name = pkg.Name
	info.Version = pkg.Version
	info.Description = pkg.Description
	info.License = pkg.License
	info.Homepage = pkg.Homepage
	info.MainLanguage = langJavaScript
	info.PackageManager = pd.detectNodePackageManager(projectRoot)
	info.ConfigFiles = append(info.ConfigFiles, "package.json")

	// Handle repository field
	if repo, ok := pkg.Repository.(string); ok {
		info.Repository = repo
	} else if repoObj, ok := pkg.Repository.(map[string]interface{}); ok {
		if url, exists := repoObj["url"].(string); exists {
			info.Repository = url
		}
	}

	// Detect if it's TypeScript
	if pd.hasTypeScript(projectRoot) {
		info.MainLanguage = langTypeScript
		info.Tags = append(info.Tags, "typescript")
	}

	// Add keywords as tags
	info.Tags = append(info.Tags, pkg.Keywords...)

	// Detect project type from scripts and dependencies
	if pd.isReactProject(pkg.Dependencies, pkg.DevDependencies) {
		info.Tags = append(info.Tags, "react")
		info.ProjectType = ProjectTypeWebApp
	} else if pd.isVueProject(pkg.Dependencies, pkg.DevDependencies) {
		info.Tags = append(info.Tags, "vue")
		info.ProjectType = ProjectTypeWebApp
	} else if pd.isAngularProject(pkg.Dependencies, pkg.DevDependencies) {
		info.Tags = append(info.Tags, "angular")
		info.ProjectType = ProjectTypeWebApp
	} else if pd.isExpressProject(pkg.Dependencies) {
		info.Tags = append(info.Tags, "express", "backend")
		info.ProjectType = ProjectTypeAPI
	} else if pd.isElectronProject(pkg.Dependencies, pkg.DevDependencies) {
		info.Tags = append(info.Tags, "electron")
		info.ProjectType = ProjectTypeDesktop
	} else if strings.Contains(pkg.Main, "bin/") || pkg.Scripts["start"] != "" {
		info.ProjectType = ProjectTypeApplication
	} else {
		info.ProjectType = ProjectTypeLibrary
	}

	return nil
}

// detectFromGoMod detects Go project information
func (pd *ProjectDetector) detectFromGoMod(projectRoot string, info *ProjectInfo) error {
	goModFile := filepath.Join(projectRoot, "go.mod")
	if !fileExists(goModFile) {
		return fmt.Errorf("go.mod not found")
	}

	// #nosec G304 - goModFile is constructed from validated projectRoot, not user input
	data, err := os.ReadFile(goModFile)
	if err != nil {
		return err
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			info.Name = strings.TrimSpace(strings.TrimPrefix(line, "module"))
			break
		}
	}

	// Extract Go version
	goVersionRegex := regexp.MustCompile(`go\s+(\d+\.\d+(?:\.\d+)?)`)
	if match := goVersionRegex.FindStringSubmatch(content); len(match) > 1 {
		info.Version = match[1]
	}

	info.MainLanguage = "Go"
	info.BuildSystem = "go"
	info.ConfigFiles = append(info.ConfigFiles, "go.mod")

	// Check for go.sum
	if fileExists(filepath.Join(projectRoot, "go.sum")) {
		info.ConfigFiles = append(info.ConfigFiles, "go.sum")
	}

	// Detect project type
	if pd.hasMainPackage(projectRoot) {
		if pd.hasWebFramework(projectRoot) {
			info.ProjectType = ProjectTypeWebApp
			info.Tags = append(info.Tags, "web")
		} else if pd.hasCLIFramework(projectRoot) {
			info.ProjectType = ProjectTypeCLI
			info.Tags = append(info.Tags, "cli")
		} else {
			info.ProjectType = ProjectTypeApplication
		}
	} else {
		info.ProjectType = ProjectTypeLibrary
	}

	return nil
}

// detectFromCargoToml detects Rust project information
func (pd *ProjectDetector) detectFromCargoToml(projectRoot string, info *ProjectInfo) error {
	cargoFile := filepath.Join(projectRoot, "Cargo.toml")
	if !fileExists(cargoFile) {
		return fmt.Errorf("cargo.toml not found")
	}

	// #nosec G304 - cargoFile is constructed from validated projectRoot, not user input
	data, err := os.ReadFile(cargoFile)
	if err != nil {
		return err
	}

	content := string(data)

	// Parse package section
	if err := pd.parseCargoPackageSection(content, info); err != nil {
		return err
	}

	// Set basic Rust project info
	pd.setCargoBasicInfo(info, projectRoot)

	// Detect project type
	pd.detectCargoProjectType(content, info)

	return nil
}

// parseCargoPackageSection parses the [package] section of Cargo.toml
func (pd *ProjectDetector) parseCargoPackageSection(content string, info *ProjectInfo) error {
	lines := strings.Split(content, "\n")
	inPackageSection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if pd.isPackageSectionStart(line) {
			inPackageSection = true
			continue
		}

		if pd.isNewSection(line) {
			inPackageSection = false
			continue
		}

		if inPackageSection {
			pd.parseCargoKeyValue(line, info)
		}
	}

	return nil
}

// isPackageSectionStart checks if line starts the [package] section
func (pd *ProjectDetector) isPackageSectionStart(line string) bool {
	return line == "[package]"
}

// isNewSection checks if line starts a new TOML section
func (pd *ProjectDetector) isNewSection(line string) bool {
	return strings.HasPrefix(line, "[")
}

// parseCargoKeyValue parses a key=value line in the package section
func (pd *ProjectDetector) parseCargoKeyValue(line string, info *ProjectInfo) {
	if !strings.Contains(line, "=") {
		return
	}

	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return
	}

	key := strings.TrimSpace(parts[0])
	value := strings.Trim(strings.TrimSpace(parts[1]), `"`)

	switch key {
	case "name":
		info.Name = value
	case "version":
		info.Version = value
	case "description":
		info.Description = value
	case "license":
		info.License = value
	case "homepage":
		info.Homepage = value
	case "repository":
		info.Repository = value
	}
}

// setCargoBasicInfo sets basic information for Rust projects
func (pd *ProjectDetector) setCargoBasicInfo(info *ProjectInfo, projectRoot string) {
	info.MainLanguage = langRust
	info.BuildSystem = "cargo"
	info.ConfigFiles = append(info.ConfigFiles, "Cargo.toml")

	if fileExists(filepath.Join(projectRoot, "Cargo.lock")) {
		info.ConfigFiles = append(info.ConfigFiles, "Cargo.lock")
	}
}

// detectCargoProjectType determines the project type based on Cargo.toml content
func (pd *ProjectDetector) detectCargoProjectType(content string, info *ProjectInfo) {
	if pd.isCargoApplication(content) {
		info.ProjectType = ProjectTypeApplication
		return
	}

	if pd.isCargoWebApp(content) {
		info.ProjectType = ProjectTypeWebApp
		info.Tags = append(info.Tags, "web")
		return
	}

	info.ProjectType = ProjectTypeLibrary
}

// isCargoApplication checks if the project is a Rust application
func (pd *ProjectDetector) isCargoApplication(content string) bool {
	return strings.Contains(content, "[[bin]]") || strings.Contains(content, `name = "main"`)
}

// isCargoWebApp checks if the project is a Rust web application
func (pd *ProjectDetector) isCargoWebApp(content string) bool {
	webFrameworks := []string{"actix", "warp", "rocket"}
	for _, framework := range webFrameworks {
		if strings.Contains(content, framework) {
			return true
		}
	}
	return false
}

// detectFromPyprojectToml detects Python project information
func (pd *ProjectDetector) detectFromPyprojectToml(projectRoot string, info *ProjectInfo) error {
	pyprojectFile := filepath.Join(projectRoot, "pyproject.toml")
	setupPyFile := filepath.Join(projectRoot, "setup.py")
	requirementsFile := filepath.Join(projectRoot, "requirements.txt")

	if fileExists(pyprojectFile) {
		info.ConfigFiles = append(info.ConfigFiles, "pyproject.toml")
		// TODO: Parse pyproject.toml
	} else if fileExists(setupPyFile) {
		info.ConfigFiles = append(info.ConfigFiles, "setup.py")
		// TODO: Parse setup.py
	} else if fileExists(requirementsFile) {
		info.ConfigFiles = append(info.ConfigFiles, "requirements.txt")
	} else {
		return fmt.Errorf("no Python project files found")
	}

	info.MainLanguage = langPython
	info.PackageManager = "pip"

	// Detect if using poetry, pipenv, etc.
	if fileExists(filepath.Join(projectRoot, "poetry.lock")) {
		info.PackageManager = "poetry"
		info.ConfigFiles = append(info.ConfigFiles, "poetry.lock")
	} else if fileExists(filepath.Join(projectRoot, "Pipfile")) {
		info.PackageManager = "pipenv"
		info.ConfigFiles = append(info.ConfigFiles, "Pipfile")
	}

	// Detect project type
	if pd.hasDjangoFiles(projectRoot) {
		info.ProjectType = ProjectTypeWebApp
		info.Tags = append(info.Tags, "django", "web")
	} else if pd.hasFlaskFiles(projectRoot) {
		info.ProjectType = ProjectTypeWebApp
		info.Tags = append(info.Tags, "flask", "web")
	} else if pd.hasFastAPIFiles(projectRoot) {
		info.ProjectType = ProjectTypeAPI
		info.Tags = append(info.Tags, "fastapi", "api")
	} else {
		info.ProjectType = ProjectTypeApplication
	}

	return nil
}

// detectFromPomXML detects Java Maven project information
func (pd *ProjectDetector) detectFromPomXML(projectRoot string, info *ProjectInfo) error {
	pomFile := filepath.Join(projectRoot, "pom.xml")
	if !fileExists(pomFile) {
		return fmt.Errorf("pom.xml not found")
	}

	// Simple XML parsing for basic fields
	// #nosec G304 - pomFile is constructed from validated projectRoot, not user input
	data, err := os.ReadFile(pomFile)
	if err != nil {
		return err
	}

	content := string(data)

	// Extract basic information using regex
	if match := regexp.MustCompile(`<artifactId>([^<]+)</artifactId>`).FindStringSubmatch(content); len(match) > 1 {
		info.Name = match[1]
	}
	if match := regexp.MustCompile(`<version>([^<]+)</version>`).FindStringSubmatch(content); len(match) > 1 {
		info.Version = match[1]
	}
	if match := regexp.MustCompile(`<description>([^<]+)</description>`).FindStringSubmatch(content); len(match) > 1 {
		info.Description = match[1]
	}

	info.MainLanguage = langJava
	info.BuildSystem = "maven"
	info.ConfigFiles = append(info.ConfigFiles, "pom.xml")

	// Detect Spring Boot
	if strings.Contains(content, "spring-boot") {
		info.ProjectType = ProjectTypeWebApp
		info.Tags = append(info.Tags, "spring-boot", "web")
	} else {
		info.ProjectType = ProjectTypeApplication
	}

	return nil
}

// detectFromBuildGradle detects Java Gradle project information
func (pd *ProjectDetector) detectFromBuildGradle(projectRoot string, info *ProjectInfo) error {
	buildFile := filepath.Join(projectRoot, "build.gradle")
	buildKtsFile := filepath.Join(projectRoot, "build.gradle.kts")

	var buildFilePath string
	if fileExists(buildFile) {
		buildFilePath = buildFile
	} else if fileExists(buildKtsFile) {
		buildFilePath = buildKtsFile
		info.Tags = append(info.Tags, "kotlin")
	} else {
		return fmt.Errorf("build.gradle not found")
	}

	// #nosec G304 - buildFilePath is constructed from validated projectRoot, not user input
	data, err := os.ReadFile(buildFilePath)
	if err != nil {
		return err
	}

	content := string(data)

	info.MainLanguage = langJava
	info.BuildSystem = "gradle"
	info.ConfigFiles = append(info.ConfigFiles, filepath.Base(buildFilePath))

	// Detect Kotlin
	if strings.Contains(content, "kotlin") {
		info.MainLanguage = "Kotlin"
	}

	// Detect Android
	if strings.Contains(content, "com.android.application") {
		info.ProjectType = ProjectTypeMobile
		info.Tags = append(info.Tags, "android", "mobile")
	} else if strings.Contains(content, "spring-boot") {
		info.ProjectType = ProjectTypeWebApp
		info.Tags = append(info.Tags, "spring-boot", "web")
	} else {
		info.ProjectType = ProjectTypeApplication
	}

	return nil
}

// detectFromComposerJSON detects PHP project information
func (pd *ProjectDetector) detectFromComposerJSON(projectRoot string, info *ProjectInfo) error {
	composerFile := filepath.Join(projectRoot, "composer.json")
	if !fileExists(composerFile) {
		return fmt.Errorf("composer.json not found")
	}

	// #nosec G304 - composerFile is constructed from validated projectRoot, not user input
	data, err := os.ReadFile(composerFile)
	if err != nil {
		return err
	}

	var composer struct {
		Name        string            `json:"name"`
		Description string            `json:"description"`
		Version     string            `json:"version"`
		License     string            `json:"license"`
		Homepage    string            `json:"homepage"`
		Require     map[string]string `json:"require"`
		RequireDev  map[string]string `json:"require-dev"`
		Keywords    []string          `json:"keywords"`
	}

	if err := json.Unmarshal(data, &composer); err != nil {
		return err
	}

	info.Name = composer.Name
	info.Version = composer.Version
	info.Description = composer.Description
	info.License = composer.License
	info.Homepage = composer.Homepage
	info.MainLanguage = langPHP
	info.PackageManager = "composer"
	info.ConfigFiles = append(info.ConfigFiles, "composer.json")
	info.Tags = append(info.Tags, composer.Keywords...)

	// Detect Laravel
	if pd.isLaravelProject(composer.Require) {
		info.ProjectType = ProjectTypeWebApp
		info.Tags = append(info.Tags, "laravel", "web")
	} else if pd.isSymfonyProject(composer.Require) {
		info.ProjectType = ProjectTypeWebApp
		info.Tags = append(info.Tags, "symfony", "web")
	} else {
		info.ProjectType = ProjectTypeApplication
	}

	return nil
}

// detectFromGemspec detects Ruby project information
func (pd *ProjectDetector) detectFromGemspec(projectRoot string, info *ProjectInfo) error {
	gemspecs, err := filepath.Glob(filepath.Join(projectRoot, "*.gemspec"))
	if err != nil || len(gemspecs) == 0 {
		// Check for Gemfile
		gemfile := filepath.Join(projectRoot, "Gemfile")
		if !fileExists(gemfile) {
			return fmt.Errorf("no Ruby project files found")
		}
		info.ConfigFiles = append(info.ConfigFiles, "Gemfile")
	} else {
		info.ConfigFiles = append(info.ConfigFiles, filepath.Base(gemspecs[0]))
	}

	info.MainLanguage = langRuby
	info.PackageManager = "bundler"

	// Detect Rails
	if pd.isRailsProject(projectRoot) {
		info.ProjectType = ProjectTypeWebApp
		info.Tags = append(info.Tags, "rails", "web")
	} else {
		info.ProjectType = ProjectTypeApplication
	}

	return nil
}

// detectGenericProject performs generic project detection
func (pd *ProjectDetector) detectGenericProject(projectRoot string, info *ProjectInfo) {
	languageFiles := pd.collectLanguageFiles(projectRoot)
	pd.determineMainLanguage(languageFiles, info)
}

// collectLanguageFiles walks the directory and counts files by extension
func (pd *ProjectDetector) collectLanguageFiles(projectRoot string) map[string]int {
	languageFiles := make(map[string]int)

	err := filepath.Walk(projectRoot, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil || fileInfo.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		languageFiles[ext]++
		return nil
	})

	if err != nil {
		// Return empty map on error for best-effort analysis
		return make(map[string]int)
	}

	return languageFiles
}

// determineMainLanguage determines the main language based on file counts
func (pd *ProjectDetector) determineMainLanguage(languageFiles map[string]int, info *ProjectInfo) {
	maxCount := 0
	mainExt := ""

	// Find extension with highest count
	for ext, count := range languageFiles {
		if count > maxCount {
			maxCount = count
			mainExt = ext
		}
	}

	// Set language based on extension
	if mainExt != "" {
		info.MainLanguage = pd.mapExtensionToLanguage(mainExt)
	}
}

// mapExtensionToLanguage maps file extensions to language names
func (pd *ProjectDetector) mapExtensionToLanguage(ext string) string {
	languageMap := map[string]string{
		".go":          "Go",
		".rs":          langRust,
		".py":          langPython,
		".js":          langJavaScript,
		".jsx":         langJavaScript,
		".ts":          langTypeScript,
		".tsx":         langTypeScript,
		".java":        langJava,
		".kt":          "Kotlin",
		".php":         langPHP,
		".rb":          langRuby,
		".cpp":         "C++",
		".cc":          "C++",
		".cxx":         "C++",
		".c":           "C",
		".cs":          "C#",
		".swift":       "Swift",
	}

	if language, exists := languageMap[ext]; exists {
		return language
	}

	return "" // Unknown language
}

// Helper functions for project detection

func (pd *ProjectDetector) detectNodePackageManager(projectRoot string) string {
	if fileExists(filepath.Join(projectRoot, "yarn.lock")) {
		return "yarn"
	} else if fileExists(filepath.Join(projectRoot, "pnpm-lock.yaml")) {
		return "pnpm"
	} else if fileExists(filepath.Join(projectRoot, "package-lock.json")) {
		return "npm"
	}
	return "npm"
}

func (pd *ProjectDetector) hasTypeScript(projectRoot string) bool {
	return fileExists(filepath.Join(projectRoot, "tsconfig.json")) ||
		fileExists(filepath.Join(projectRoot, "tsconfig.base.json"))
}

func (pd *ProjectDetector) isReactProject(deps, devDeps map[string]string) bool {
	return hasAnyDependency(deps, devDeps, "react", "@types/react")
}

func (pd *ProjectDetector) isVueProject(deps, devDeps map[string]string) bool {
	return hasAnyDependency(deps, devDeps, "vue", "@vue/cli")
}

func (pd *ProjectDetector) isAngularProject(deps, devDeps map[string]string) bool {
	return hasAnyDependency(deps, devDeps, "@angular/core", "@angular/cli")
}

func (pd *ProjectDetector) isExpressProject(deps map[string]string) bool {
	_, exists := deps["express"]
	return exists
}

func (pd *ProjectDetector) isElectronProject(deps, devDeps map[string]string) bool {
	return hasAnyDependency(deps, devDeps, "electron")
}

func (pd *ProjectDetector) hasMainPackage(projectRoot string) bool {
	mainFile := filepath.Join(projectRoot, "main.go")
	return fileExists(mainFile) || pd.hasGoMainInAnyFile(projectRoot)
}

func (pd *ProjectDetector) hasGoMainInAnyFile(projectRoot string) bool {
	found := false
	// Walk through directory to find Go main functions
	// Ignore walk errors as this is best-effort detection
	if err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// #nosec G304 - path comes from validated hasGoMainInAnyFile call, not user input
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		if strings.Contains(string(content), "package main") && strings.Contains(string(content), "func main()") {
			found = true
			return filepath.SkipDir
		}
		return nil
	}); err != nil {
		// Log error but continue as this is best-effort detection
		// In a real implementation, we might want to log this
		return false
	}
	return found
}

func (pd *ProjectDetector) hasWebFramework(projectRoot string) bool {
	// Check go.mod for web frameworks
	goModFile := filepath.Join(projectRoot, "go.mod")
	if fileExists(goModFile) {
		// #nosec G304 - goModFile is constructed from validated projectRoot, not user input
		content, err := os.ReadFile(goModFile)
		if err != nil {
			// If we can't read go.mod, assume no web framework
			return false
		}
		contentStr := string(content)
		return strings.Contains(contentStr, "gin-gonic/gin") ||
			strings.Contains(contentStr, "gorilla/mux") ||
			strings.Contains(contentStr, "echo") ||
			strings.Contains(contentStr, "fiber")
	}
	return false
}

func (pd *ProjectDetector) hasCLIFramework(projectRoot string) bool {
	// Check go.mod for CLI frameworks
	goModFile := filepath.Join(projectRoot, "go.mod")
	if fileExists(goModFile) {
		// #nosec G304 - goModFile is constructed from validated projectRoot, not user input
		content, err := os.ReadFile(goModFile)
		if err != nil {
			// If we can't read go.mod, assume no CLI framework
			return false
		}
		contentStr := string(content)
		return strings.Contains(contentStr, "spf13/cobra") ||
			strings.Contains(contentStr, "urfave/cli")
	}
	return false
}

func (pd *ProjectDetector) hasDjangoFiles(projectRoot string) bool {
	return fileExists(filepath.Join(projectRoot, "manage.py")) ||
		fileExists(filepath.Join(projectRoot, "django_project"))
}

func (pd *ProjectDetector) hasFlaskFiles(projectRoot string) bool {
	// Look for app.py or files importing Flask
	return pd.hasFileWithContent(projectRoot, "*.py", "from flask import") ||
		pd.hasFileWithContent(projectRoot, "*.py", "import flask")
}

func (pd *ProjectDetector) hasFastAPIFiles(projectRoot string) bool {
	return pd.hasFileWithContent(projectRoot, "*.py", "from fastapi import") ||
		pd.hasFileWithContent(projectRoot, "*.py", "import fastapi")
}

func (pd *ProjectDetector) isLaravelProject(deps map[string]string) bool {
	_, exists := deps["laravel/framework"]
	return exists
}

func (pd *ProjectDetector) isSymfonyProject(deps map[string]string) bool {
	_, exists := deps["symfony/framework-bundle"]
	return exists
}

func (pd *ProjectDetector) isRailsProject(projectRoot string) bool {
	return fileExists(filepath.Join(projectRoot, "config", "application.rb")) ||
		fileExists(filepath.Join(projectRoot, "Rakefile"))
}

func (pd *ProjectDetector) hasFileWithContent(projectRoot, pattern, content string) bool {
	found := false
	// Walk through directory to find files with specific content
	// Ignore walk errors as this is best-effort detection
	if err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			// If pattern matching fails, skip this file
			return nil
		}
		if !matched {
			return nil
		}

		// #nosec G304 - path comes from validated filepath.Walk, not user input
		fileContent, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		if strings.Contains(string(fileContent), content) {
			found = true
			return filepath.SkipDir
		}
		return nil
	}); err != nil {
		// Log error but continue as this is best-effort detection
		// In a real implementation, we might want to log this
		return false
	}
	return found
}

func (pd *ProjectDetector) detectProjectStructure(projectRoot string, info *ProjectInfo) {
	// Common source directories
	sourceDirs := []string{"src", "lib", "app", "source", "code"}
	for _, dir := range sourceDirs {
		if dirExists(filepath.Join(projectRoot, dir)) {
			info.SourceDirs = append(info.SourceDirs, dir)
		}
	}

	// Common test directories
	testDirs := []string{"test", "tests", "__tests__", "spec", "specs"}
	for _, dir := range testDirs {
		if dirExists(filepath.Join(projectRoot, dir)) {
			info.TestDirs = append(info.TestDirs, dir)
		}
	}

	// Common documentation directories
	docDirs := []string{"docs", "doc", "documentation", "guide", "guides"}
	for _, dir := range docDirs {
		if dirExists(filepath.Join(projectRoot, dir)) {
			info.DocumentationDirs = append(info.DocumentationDirs, dir)
		}
	}
}

func (pd *ProjectDetector) detectBuildSystem(projectRoot string, info *ProjectInfo) {
	buildFiles := map[string]string{
		"Makefile":          "make",
		"CMakeLists.txt":    "cmake",
		"meson.build":       "meson",
		"BUILD.bazel":       "bazel",
		"BUILD":             "bazel",
		"SConstruct":        "scons",
		"build.xml":         "ant",
		"gulpfile.js":       "gulp",
		"webpack.config.js": "webpack",
		"rollup.config.js":  "rollup",
		"vite.config.js":    "vite",
	}

	for file, system := range buildFiles {
		if fileExists(filepath.Join(projectRoot, file)) {
			info.BuildSystem = system
			info.ConfigFiles = append(info.ConfigFiles, file)
			break
		}
	}
}

func (pd *ProjectDetector) detectEntryPoints(projectRoot string, info *ProjectInfo) {
	entryPoints := []string{
		"main.go", "main.rs", "main.py", "main.js", "main.ts",
		"index.js", "index.ts", "index.html", "app.js", "app.py",
		"server.js", "server.py", "run.py", "cli.py",
	}

	for _, entry := range entryPoints {
		if fileExists(filepath.Join(projectRoot, entry)) {
			info.EntryPoints = append(info.EntryPoints, entry)
		}
	}
}

func (pd *ProjectDetector) inferProjectType(projectRoot string, info *ProjectInfo) {
	if info.ProjectType != ProjectTypeUnknown {
		return // Already detected
	}

	// Infer from directory structure and files
	if dirExists(filepath.Join(projectRoot, "cmd")) {
		info.ProjectType = ProjectTypeCLI
	} else if fileExists(filepath.Join(projectRoot, "Dockerfile")) {
		info.ProjectType = ProjectTypeMicroservice
	} else if len(info.EntryPoints) > 0 {
		info.ProjectType = ProjectTypeApplication
	} else {
		info.ProjectType = ProjectTypeLibrary
	}
}

// Utility functions

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func hasAnyDependency(deps, devDeps map[string]string, packages ...string) bool {
	for _, pkg := range packages {
		if _, exists := deps[pkg]; exists {
			return true
		}
		if _, exists := devDeps[pkg]; exists {
			return true
		}
	}
	return false
}
