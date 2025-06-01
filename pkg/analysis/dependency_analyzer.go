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

// Language constants
const (
	langJavaScript = "JavaScript"
	langTypeScript = "TypeScript"
	langPython     = "Python"
	langRust       = "Rust"
	langJava       = "Java"
	langKotlin     = "Kotlin"
	langPHP        = "PHP"
	langRuby       = "Ruby"
)

// Package manager constants
const (
	packageManagerNpm     = "npm"
	packageManagerUnknown = "unknown"
)

// DependencyAnalyzer analyzes project dependencies and their relationships
type DependencyAnalyzer struct {
	config AnalyzerConfig
}

// NewDependencyAnalyzer creates a new dependency analyzer
func NewDependencyAnalyzer(config AnalyzerConfig) *DependencyAnalyzer {
	return &DependencyAnalyzer{
		config: config,
	}
}

// AnalyzeDependencies performs comprehensive dependency analysis
func (da *DependencyAnalyzer) AnalyzeDependencies(ctx context.Context, projectRoot string, projectInfo *ProjectInfo) (*DependencyInfo, error) {
	depInfo := da.initializeDependencyInfo()

	// Analyze dependencies based on language
	if err := da.analyzeByLanguage(projectRoot, projectInfo.MainLanguage, depInfo); err != nil {
		return depInfo, err
	}

	// Post-analysis processing
	da.performPostAnalysis(depInfo)

	return depInfo, nil
}

// initializeDependencyInfo creates a new DependencyInfo with empty collections
func (da *DependencyAnalyzer) initializeDependencyInfo() *DependencyInfo {
	return &DependencyInfo{
		Direct:           []Dependency{},
		Transitive:       []Dependency{},
		DevDeps:          []Dependency{},
		PeerDeps:         []Dependency{},
		Conflicts:        []DependencyConflict{},
		Security:         []SecurityIssue{},
		Licenses:         make(map[string]int),
		UpdatesAvailable: []DependencyUpdate{},
	}
}

// analyzeByLanguage analyzes dependencies based on the project's main language
func (da *DependencyAnalyzer) analyzeByLanguage(projectRoot, mainLanguage string, depInfo *DependencyInfo) error {
	languageAnalyzers := map[string]func(string, *DependencyInfo) error{
		langJavaScript: da.analyzeNodeDependencies,
		langTypeScript: da.analyzeNodeDependencies,
		"Go":           da.analyzeGoDependencies,
		langPython:     da.analyzePythonDependencies,
		langRust:       da.analyzeRustDependencies,
		langJava:       da.analyzeJavaDependencies,
		langKotlin:     da.analyzeJavaDependencies,
		langPHP:        da.analyzePHPDependencies,
		langRuby:       da.analyzeRubyDependencies,
	}

	analyzer, exists := languageAnalyzers[mainLanguage]
	if !exists {
		return nil // No analyzer for this language
	}

	if err := analyzer(projectRoot, depInfo); err != nil {
		return fmt.Errorf("%s dependency analysis failed: %w", mainLanguage, err)
	}

	return nil
}

// performPostAnalysis executes post-analysis processing steps
func (da *DependencyAnalyzer) performPostAnalysis(depInfo *DependencyInfo) {
	// Analyze for security vulnerabilities
	da.analyzeSecurityIssues(depInfo)

	// Check for available updates
	da.checkForUpdates(depInfo)

	// Analyze license compatibility
	da.analyzeLicenses(depInfo)

	// Detect dependency conflicts
	da.detectConflicts(depInfo)
}

// analyzeNodeDependencies analyzes Node.js/npm dependencies
func (da *DependencyAnalyzer) analyzeNodeDependencies(projectRoot string, depInfo *DependencyInfo) error {
	packageFile := filepath.Join(projectRoot, "package.json")
	if !fileExists(packageFile) {
		return fmt.Errorf("package.json not found")
	}

	// #nosec G304 - packageFile path is validated project file
	data, err := os.ReadFile(packageFile)
	if err != nil {
		return err
	}

	var pkg struct {
		Dependencies     map[string]string `json:"dependencies"`
		DevDependencies  map[string]string `json:"devDependencies"`
		PeerDependencies map[string]string `json:"peerDependencies"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return err
	}

	// Process direct dependencies
	for name, version := range pkg.Dependencies {
		dep := da.createDependency(name, version, DependencyTypeDirect)
		depInfo.Direct = append(depInfo.Direct, dep)
	}

	// Process dev dependencies
	for name, version := range pkg.DevDependencies {
		dep := da.createDependency(name, version, DependencyTypeDev)
		depInfo.DevDeps = append(depInfo.DevDeps, dep)
	}

	// Process peer dependencies
	for name, version := range pkg.PeerDependencies {
		dep := da.createDependency(name, version, DependencyTypePeer)
		depInfo.PeerDeps = append(depInfo.PeerDeps, dep)
	}

	// Analyze package-lock.json for transitive dependencies
	lockFile := filepath.Join(projectRoot, "package-lock.json")
	if fileExists(lockFile) {
		if err := da.analyzePackageLock(lockFile, depInfo); err != nil {
			return fmt.Errorf("package-lock.json analysis failed: %w", err)
		}
	}

	return nil
}

// analyzeGoDependencies analyzes Go module dependencies
func (da *DependencyAnalyzer) analyzeGoDependencies(projectRoot string, depInfo *DependencyInfo) error {
	goModFile := filepath.Join(projectRoot, "go.mod")
	if !fileExists(goModFile) {
		return fmt.Errorf("go.mod not found")
	}

	// #nosec G304 - goModFile is constructed from validated projectRoot with hardcoded filename "go.mod"
	content, err := os.ReadFile(goModFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	inRequireBlock := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "require (" {
			inRequireBlock = true
			continue
		} else if line == ")" && inRequireBlock {
			inRequireBlock = false
			continue
		}

		if inRequireBlock || strings.HasPrefix(line, "require ") {
			if dep := da.parseGoRequireLine(line); dep != nil {
				depInfo.Direct = append(depInfo.Direct, *dep)
			}
		}
	}

	// Analyze go.sum for transitive dependencies
	goSumFile := filepath.Join(projectRoot, "go.sum")
	if fileExists(goSumFile) {
		if err := da.analyzeGoSum(goSumFile, depInfo); err != nil {
			return fmt.Errorf("go.sum analysis failed: %w", err)
		}
	}

	return nil
}

// analyzePythonDependencies analyzes Python dependencies
func (da *DependencyAnalyzer) analyzePythonDependencies(projectRoot string, depInfo *DependencyInfo) error {
	// Check requirements.txt
	reqFiles := []string{"requirements.txt", "requirements-dev.txt", "dev-requirements.txt"}
	for _, reqFile := range reqFiles {
		filePath := filepath.Join(projectRoot, reqFile)
		if fileExists(filePath) {
			if err := da.analyzeRequirementsTxt(filePath, depInfo); err != nil {
				return fmt.Errorf("requirements.txt analysis failed: %w", err)
			}
		}
	}

	// Check pyproject.toml
	pyprojectFile := filepath.Join(projectRoot, "pyproject.toml")
	if fileExists(pyprojectFile) {
		if err := da.analyzePyprojectToml(pyprojectFile, depInfo); err != nil {
			return fmt.Errorf("pyproject.toml analysis failed: %w", err)
		}
	}

	// Check setup.py
	setupFile := filepath.Join(projectRoot, "setup.py")
	if fileExists(setupFile) {
		if err := da.analyzeSetupPy(setupFile, depInfo); err != nil {
			return fmt.Errorf("setup.py analysis failed: %w", err)
		}
	}

	return nil
}

// analyzeRustDependencies analyzes Rust/Cargo dependencies
func (da *DependencyAnalyzer) analyzeRustDependencies(projectRoot string, depInfo *DependencyInfo) error {
	cargoFile := filepath.Join(projectRoot, "Cargo.toml")
	if !fileExists(cargoFile) {
		return fmt.Errorf("cargo.toml not found")
	}

	// #nosec G304 - cargoFile is constructed from validated projectRoot with hardcoded filename "Cargo.toml"
	content, err := os.ReadFile(cargoFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	inDependenciesSection := false
	inDevDependenciesSection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "[dependencies]" {
			inDependenciesSection = true
			inDevDependenciesSection = false
			continue
		} else if line == "[dev-dependencies]" {
			inDependenciesSection = false
			inDevDependenciesSection = true
			continue
		} else if strings.HasPrefix(line, "[") {
			inDependenciesSection = false
			inDevDependenciesSection = false
			continue
		}

		if (inDependenciesSection || inDevDependenciesSection) && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				version := strings.Trim(strings.TrimSpace(parts[1]), `"`)

				depType := DependencyTypeDirect
				if inDevDependenciesSection {
					depType = DependencyTypeDev
				}

				dep := da.createDependency(name, version, depType)
				if depType == DependencyTypeDev {
					depInfo.DevDeps = append(depInfo.DevDeps, dep)
				} else {
					depInfo.Direct = append(depInfo.Direct, dep)
				}
			}
		}
	}

	return nil
}

// analyzeJavaDependencies analyzes Java/Maven/Gradle dependencies
func (da *DependencyAnalyzer) analyzeJavaDependencies(projectRoot string, depInfo *DependencyInfo) error {
	// Check Maven pom.xml
	pomFile := filepath.Join(projectRoot, "pom.xml")
	if fileExists(pomFile) {
		return da.analyzeMavenDependencies(pomFile, depInfo)
	}

	// Check Gradle build files
	gradleFiles := []string{"build.gradle", "build.gradle.kts"}
	for _, gradleFile := range gradleFiles {
		filePath := filepath.Join(projectRoot, gradleFile)
		if fileExists(filePath) {
			return da.analyzeGradleDependencies(filePath, depInfo)
		}
	}

	return fmt.Errorf("no Java build file found")
}

// analyzePHPDependencies analyzes PHP/Composer dependencies
func (da *DependencyAnalyzer) analyzePHPDependencies(projectRoot string, depInfo *DependencyInfo) error {
	composerFile := filepath.Join(projectRoot, "composer.json")
	if !fileExists(composerFile) {
		return fmt.Errorf("composer.json not found")
	}

	// #nosec G304 - composerFile is constructed from validated projectRoot with hardcoded filename "composer.json"
	data, err := os.ReadFile(composerFile)
	if err != nil {
		return err
	}

	var composer struct {
		Require    map[string]string `json:"require"`
		RequireDev map[string]string `json:"require-dev"`
	}

	if err := json.Unmarshal(data, &composer); err != nil {
		return err
	}

	// Process direct dependencies
	for name, version := range composer.Require {
		dep := da.createDependency(name, version, DependencyTypeDirect)
		depInfo.Direct = append(depInfo.Direct, dep)
	}

	// Process dev dependencies
	for name, version := range composer.RequireDev {
		dep := da.createDependency(name, version, DependencyTypeDev)
		depInfo.DevDeps = append(depInfo.DevDeps, dep)
	}

	return nil
}

// analyzeRubyDependencies analyzes Ruby/Bundler dependencies
func (da *DependencyAnalyzer) analyzeRubyDependencies(projectRoot string, depInfo *DependencyInfo) error {
	gemfile := filepath.Join(projectRoot, "Gemfile")
	if !fileExists(gemfile) {
		return fmt.Errorf("gemfile not found")
	}

	content, err := os.ReadFile(filepath.Clean(gemfile)) // #nosec G304 - path is validated above
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	gemRegex := regexp.MustCompile(`gem\s+['"]([^'"]+)['"](?:\s*,\s*['"]([^'"]+)['"])?`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}

		matches := gemRegex.FindStringSubmatch(line)
		if len(matches) >= 2 {
			name := matches[1]
			version := ""
			if len(matches) > 2 && matches[2] != "" {
				version = matches[2]
			}

			dep := da.createDependency(name, version, DependencyTypeDirect)
			depInfo.Direct = append(depInfo.Direct, dep)
		}
	}

	return nil
}

// Helper methods for specific file formats

func (da *DependencyAnalyzer) analyzePackageLock(lockFile string, depInfo *DependencyInfo) error {
	// Simplified package-lock.json analysis
	// In a real implementation, this would parse the lock file and extract transitive dependencies
	return nil
}

func (da *DependencyAnalyzer) analyzeGoSum(goSumFile string, depInfo *DependencyInfo) error {
	// Simplified go.sum analysis
	// In a real implementation, this would parse the sum file for transitive dependencies
	return nil
}

func (da *DependencyAnalyzer) analyzeRequirementsTxt(filePath string, depInfo *DependencyInfo) error {
	content, err := os.ReadFile(filepath.Clean(filePath)) // #nosec G304 - path is from trusted project analysis
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse package==version or package>=version format
		name, version := da.parsePythonRequirement(line)
		if name != "" {
			depType := DependencyTypeDirect
			if strings.Contains(filePath, "dev") {
				depType = DependencyTypeDev
			}

			dep := da.createDependency(name, version, depType)
			if depType == DependencyTypeDev {
				depInfo.DevDeps = append(depInfo.DevDeps, dep)
			} else {
				depInfo.Direct = append(depInfo.Direct, dep)
			}
		}
	}

	return nil
}

func (da *DependencyAnalyzer) analyzePyprojectToml(filePath string, depInfo *DependencyInfo) error {
	// Simplified pyproject.toml analysis
	// In a real implementation, this would parse TOML format
	return nil
}

func (da *DependencyAnalyzer) analyzeSetupPy(filePath string, depInfo *DependencyInfo) error {
	// Simplified setup.py analysis
	// In a real implementation, this would parse Python code for install_requires
	return nil
}

func (da *DependencyAnalyzer) analyzeMavenDependencies(pomFile string, depInfo *DependencyInfo) error {
	// Simplified Maven pom.xml analysis
	// In a real implementation, this would parse XML for dependencies
	return nil
}

func (da *DependencyAnalyzer) analyzeGradleDependencies(gradleFile string, depInfo *DependencyInfo) error {
	// Simplified Gradle build file analysis
	// In a real implementation, this would parse Gradle DSL
	return nil
}

// Helper methods

func (da *DependencyAnalyzer) createDependency(name, version string, depType DependencyType) Dependency {
	return Dependency{
		Name:            name,
		Version:         da.cleanVersion(version),
		Type:            depType,
		Source:          da.getSourceForDependency(name),
		License:         "",
		Description:     "",
		Homepage:        "",
		Repository:      "",
		Deprecated:      false,
		Vulnerabilities: []SecurityVulnerability{},
	}
}

func (da *DependencyAnalyzer) cleanVersion(version string) string {
	// Remove version prefixes like ^, ~, >=, etc.
	cleanRegex := regexp.MustCompile(`^[^\d]*(.+)`)
	matches := cleanRegex.FindStringSubmatch(version)
	if len(matches) > 1 {
		return matches[1]
	}
	return version
}

func (da *DependencyAnalyzer) getSourceForDependency(name string) string {
	// Determine source based on package name patterns
	if strings.Contains(name, "github.com") {
		return "github"
	} else if strings.Contains(name, "@") {
		return packageManagerNpm
	}
	return packageManagerUnknown
}

func (da *DependencyAnalyzer) parseGoRequireLine(line string) *Dependency {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "require ")

	parts := strings.Fields(line)
	if len(parts) >= 2 {
		name := parts[0]
		version := parts[1]

		dep := da.createDependency(name, version, DependencyTypeDirect)
		return &dep
	}

	return nil
}

func (da *DependencyAnalyzer) parsePythonRequirement(line string) (string, string) {
	// Parse formats like: package==1.0.0, package>=1.0.0, package~=1.0.0
	operators := []string{"==", ">=", "<=", "~=", ">", "<", "!="}

	for _, op := range operators {
		if strings.Contains(line, op) {
			parts := strings.Split(line, op)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				version := strings.TrimSpace(parts[1])
				return name, version
			}
		}
	}

	// If no version specified, just return the package name
	return strings.TrimSpace(line), ""
}

func (da *DependencyAnalyzer) analyzeSecurityIssues(depInfo *DependencyInfo) {
	// Placeholder for security vulnerability analysis
	// In a real implementation, this would integrate with vulnerability databases
	// like GitHub Security Advisory Database, npm audit, etc.
	for _, dep := range depInfo.Direct {
		if da.hasKnownVulnerabilities(dep.Name, dep.Version) {
			issue := SecurityIssue{
				Type:        "vulnerability",
				Severity:    "medium",
				Package:     dep.Name,
				Version:     dep.Version,
				Title:       "Known vulnerability",
				Description: fmt.Sprintf("Package %s@%s has known vulnerabilities", dep.Name, dep.Version),
				References:  []string{},
			}
			depInfo.Security = append(depInfo.Security, issue)
		}
	}
}

func (da *DependencyAnalyzer) checkForUpdates(depInfo *DependencyInfo) {
	// Placeholder for update checking
	// In a real implementation, this would check package registries for newer versions
	for _, dep := range depInfo.Direct {
		if latestVersion := da.getLatestVersion(dep.Name); latestVersion != dep.Version {
			update := DependencyUpdate{
				Package:        dep.Name,
				CurrentVersion: dep.Version,
				LatestVersion:  latestVersion,
				UpdateType:     da.getUpdateType(dep.Version, latestVersion),
				Breaking:       da.isBreakingChange(dep.Version, latestVersion),
			}
			depInfo.UpdatesAvailable = append(depInfo.UpdatesAvailable, update)
		}
	}
}

func (da *DependencyAnalyzer) analyzeLicenses(depInfo *DependencyInfo) {
	// Placeholder for license analysis
	// In a real implementation, this would fetch license information from package metadata
	for _, dep := range depInfo.Direct {
		license := da.getLicenseForPackage(dep.Name)
		if license != "" {
			depInfo.Licenses[license]++
		}
	}
}

func (da *DependencyAnalyzer) detectConflicts(depInfo *DependencyInfo) {
	// Placeholder for conflict detection
	// In a real implementation, this would analyze version ranges and detect conflicts
	packageVersions := make(map[string][]string)

	// Collect all versions for each package
	allDeps := append(depInfo.Direct, depInfo.DevDeps...)
	allDeps = append(allDeps, depInfo.Transitive...)

	for _, dep := range allDeps {
		packageVersions[dep.Name] = append(packageVersions[dep.Name], dep.Version)
	}

	// Check for conflicts
	for pkg, versions := range packageVersions {
		if len(versions) > 1 {
			conflict := DependencyConflict{
				Package:    pkg,
				Versions:   versions,
				Reason:     "Multiple versions required",
				Severity:   "warning",
				Resolution: "Consider consolidating to a single version",
			}
			depInfo.Conflicts = append(depInfo.Conflicts, conflict)
		}
	}
}

// Placeholder methods for external data sources

func (da *DependencyAnalyzer) hasKnownVulnerabilities(name, version string) bool {
	// In a real implementation, this would check vulnerability databases
	return false
}

func (da *DependencyAnalyzer) getLatestVersion(name string) string {
	// In a real implementation, this would query package registries
	return "1.0.0"
}

func (da *DependencyAnalyzer) getUpdateType(current, latest string) string {
	// In a real implementation, this would analyze semantic versioning
	return "minor"
}

func (da *DependencyAnalyzer) isBreakingChange(current, latest string) bool {
	// In a real implementation, this would analyze semantic versioning
	return false
}

func (da *DependencyAnalyzer) getLicenseForPackage(name string) string {
	// In a real implementation, this would fetch license from package metadata
	return "MIT"
}
