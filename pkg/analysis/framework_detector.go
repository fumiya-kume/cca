package analysis

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FrameworkDetector detects frameworks and libraries used in a project
type FrameworkDetector struct {
	config AnalyzerConfig
}

// NewFrameworkDetector creates a new framework detector
func NewFrameworkDetector(config AnalyzerConfig) *FrameworkDetector {
	return &FrameworkDetector{
		config: config,
	}
}

// DetectFrameworks detects all frameworks used in the project
func (fd *FrameworkDetector) DetectFrameworks(ctx context.Context, projectRoot string) ([]FrameworkInfo, error) {
	var frameworks []FrameworkInfo

	// Detect from package managers
	if jsFrameworks := fd.detectJavaScriptFrameworks(projectRoot); len(jsFrameworks) > 0 {
		frameworks = append(frameworks, jsFrameworks...)
	}

	if pyFrameworks := fd.detectPythonFrameworks(projectRoot); len(pyFrameworks) > 0 {
		frameworks = append(frameworks, pyFrameworks...)
	}

	if goFrameworks := fd.detectGoFrameworks(projectRoot); len(goFrameworks) > 0 {
		frameworks = append(frameworks, goFrameworks...)
	}

	if javaFrameworks := fd.detectJavaFrameworks(projectRoot); len(javaFrameworks) > 0 {
		frameworks = append(frameworks, javaFrameworks...)
	}

	if rustFrameworks := fd.detectRustFrameworks(projectRoot); len(rustFrameworks) > 0 {
		frameworks = append(frameworks, rustFrameworks...)
	}

	if phpFrameworks := fd.detectPHPFrameworks(projectRoot); len(phpFrameworks) > 0 {
		frameworks = append(frameworks, phpFrameworks...)
	}

	if rubyFrameworks := fd.detectRubyFrameworks(projectRoot); len(rubyFrameworks) > 0 {
		frameworks = append(frameworks, rubyFrameworks...)
	}

	// Detect from file patterns and content
	if fileFrameworks := fd.detectFromFilePatterns(projectRoot); len(fileFrameworks) > 0 {
		frameworks = append(frameworks, fileFrameworks...)
	}

	// Remove duplicates and enrich with additional information
	frameworks = fd.deduplicateAndEnrich(frameworks, projectRoot)

	return frameworks, nil
}

// detectJavaScriptFrameworks detects JavaScript/TypeScript frameworks
func (fd *FrameworkDetector) detectJavaScriptFrameworks(projectRoot string) []FrameworkInfo {
	allDeps, err := fd.parsePackageJSON(projectRoot)
	if err != nil {
		return []FrameworkInfo{}
	}

	var frameworks []FrameworkInfo
	frameworks = append(frameworks, fd.detectWebFrameworks(projectRoot, allDeps)...)
	frameworks = append(frameworks, fd.detectBackendFrameworks(projectRoot, allDeps)...)
	frameworks = append(frameworks, fd.detectDesktopFrameworks(projectRoot, allDeps)...)
	frameworks = append(frameworks, fd.detectTestingFrameworks(projectRoot, allDeps)...)
	frameworks = append(frameworks, fd.detectBuildTools(projectRoot, allDeps)...)

	return frameworks
}

// parsePackageJSON parses package.json and returns all dependencies
func (fd *FrameworkDetector) parsePackageJSON(projectRoot string) (map[string]string, error) {
	packageFile := filepath.Join(projectRoot, "package.json")
	if !fileExists(packageFile) {
		return nil, nil
	}

	data, err := os.ReadFile(filepath.Clean(packageFile)) // #nosec G304 - path is constructed from trusted project root
	if err != nil {
		return nil, err
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
		Scripts         map[string]string `json:"scripts"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	allDeps := make(map[string]string)
	for k, v := range pkg.Dependencies {
		allDeps[k] = v
	}
	for k, v := range pkg.DevDependencies {
		allDeps[k] = v
	}

	return allDeps, nil
}

// detectWebFrameworks detects frontend web frameworks
func (fd *FrameworkDetector) detectWebFrameworks(projectRoot string, allDeps map[string]string) []FrameworkInfo {
	var frameworks []FrameworkInfo

	if version, exists := allDeps["react"]; exists {
		frameworks = append(frameworks, FrameworkInfo{
			Name:         "React",
			Version:      cleanVersion(version),
			Type:         FrameworkTypeWeb,
			ConfigFiles:  fd.findReactConfigFiles(projectRoot),
			Dependencies: []string{"react-dom"},
			Patterns:     []string{"jsx", "components", "hooks"},
			Conventions:  fd.getReactConventions(projectRoot),
		})
	}

	if version, exists := allDeps["vue"]; exists {
		frameworks = append(frameworks, FrameworkInfo{
			Name:         "Vue.js",
			Version:      cleanVersion(version),
			Type:         FrameworkTypeWeb,
			ConfigFiles:  fd.findVueConfigFiles(projectRoot),
			Dependencies: []string{"@vue/cli"},
			Patterns:     []string{"vue", "components", "composition-api"},
			Conventions:  fd.getVueConventions(projectRoot),
		})
	}

	if version, exists := allDeps["@angular/core"]; exists {
		frameworks = append(frameworks, FrameworkInfo{
			Name:         "Angular",
			Version:      cleanVersion(version),
			Type:         FrameworkTypeWeb,
			ConfigFiles:  fd.findAngularConfigFiles(projectRoot),
			Dependencies: []string{"@angular/cli", "@angular/common"},
			Patterns:     []string{"components", "services", "modules"},
			Conventions:  fd.getAngularConventions(projectRoot),
		})
	}

	if version, exists := allDeps["svelte"]; exists {
		frameworks = append(frameworks, FrameworkInfo{
			Name:         "Svelte",
			Version:      cleanVersion(version),
			Type:         FrameworkTypeWeb,
			ConfigFiles:  fd.findSvelteConfigFiles(projectRoot),
			Dependencies: []string{"@sveltejs/kit"},
			Patterns:     []string{"svelte", "components", "stores"},
			Conventions:  fd.getSvelteConventions(projectRoot),
		})
	}

	return frameworks
}

// detectBackendFrameworks detects backend Node.js frameworks
func (fd *FrameworkDetector) detectBackendFrameworks(projectRoot string, allDeps map[string]string) []FrameworkInfo {
	var frameworks []FrameworkInfo

	if version, exists := allDeps["express"]; exists {
		frameworks = append(frameworks, FrameworkInfo{
			Name:         "Express.js",
			Version:      cleanVersion(version),
			Type:         FrameworkTypeAPI,
			ConfigFiles:  []string{},
			Dependencies: []string{},
			Patterns:     []string{"middleware", "routes", "controllers"},
			Conventions:  fd.getExpressConventions(projectRoot),
		})
	}

	if version, exists := allDeps["fastify"]; exists {
		frameworks = append(frameworks, FrameworkInfo{
			Name:         "Fastify",
			Version:      cleanVersion(version),
			Type:         FrameworkTypeAPI,
			ConfigFiles:  []string{},
			Dependencies: []string{},
			Patterns:     []string{"plugins", "routes", "schemas"},
			Conventions:  fd.getFastifyConventions(projectRoot),
		})
	}

	if version, exists := allDeps["koa"]; exists {
		frameworks = append(frameworks, FrameworkInfo{
			Name:         "Koa.js",
			Version:      cleanVersion(version),
			Type:         FrameworkTypeAPI,
			ConfigFiles:  []string{},
			Dependencies: []string{},
			Patterns:     []string{"middleware", "context", "async"},
			Conventions:  fd.getKoaConventions(projectRoot),
		})
	}

	return frameworks
}

// detectDesktopFrameworks detects desktop application frameworks
func (fd *FrameworkDetector) detectDesktopFrameworks(projectRoot string, allDeps map[string]string) []FrameworkInfo {
	var frameworks []FrameworkInfo

	if version, exists := allDeps["electron"]; exists {
		frameworks = append(frameworks, FrameworkInfo{
			Name:         "Electron",
			Version:      cleanVersion(version),
			Type:         FrameworkTypeDesktop,
			ConfigFiles:  fd.findElectronConfigFiles(projectRoot),
			Dependencies: []string{},
			Patterns:     []string{"main-process", "renderer-process", "ipc"},
			Conventions:  fd.getElectronConventions(projectRoot),
		})
	}

	return frameworks
}

// detectTestingFrameworks detects testing frameworks
func (fd *FrameworkDetector) detectTestingFrameworks(projectRoot string, allDeps map[string]string) []FrameworkInfo {
	var frameworks []FrameworkInfo

	testingFrameworks := map[string]string{
		"jest":               "Jest",
		"mocha":              "Mocha",
		"vitest":             "Vitest",
		"cypress":            "Cypress",
		"playwright":         "Playwright",
		"selenium-webdriver": "Selenium",
	}

	for dep, name := range testingFrameworks {
		if version, exists := allDeps[dep]; exists {
			frameworks = append(frameworks, FrameworkInfo{
				Name:         name,
				Version:      cleanVersion(version),
				Type:         FrameworkTypeTesting,
				ConfigFiles:  fd.findTestingConfigFiles(projectRoot, dep),
				Dependencies: []string{},
				Patterns:     []string{"test", "spec", "describe", "it"},
				Conventions:  fd.getTestingConventions(projectRoot, dep),
			})
		}
	}

	return frameworks
}

// detectBuildTools detects build tools and bundlers
func (fd *FrameworkDetector) detectBuildTools(projectRoot string, allDeps map[string]string) []FrameworkInfo {
	var frameworks []FrameworkInfo

	buildTools := map[string]string{
		"webpack":  "Webpack",
		"rollup":   "Rollup",
		"vite":     "Vite",
		"parcel":   "Parcel",
		"esbuild":  "ESBuild",
		"snowpack": "Snowpack",
	}

	for dep, name := range buildTools {
		if version, exists := allDeps[dep]; exists {
			frameworks = append(frameworks, FrameworkInfo{
				Name:         name,
				Version:      cleanVersion(version),
				Type:         FrameworkTypeBuild,
				ConfigFiles:  fd.findBuildConfigFiles(projectRoot, dep),
				Dependencies: []string{},
				Patterns:     []string{"bundling", "modules", "assets"},
				Conventions:  fd.getBuildToolConventions(projectRoot, dep),
			})
		}
	}

	return frameworks
}

// detectPythonFrameworks detects Python frameworks
func (fd *FrameworkDetector) detectPythonFrameworks(projectRoot string) []FrameworkInfo {
	var frameworks []FrameworkInfo

	requirements := fd.readPythonRequirements(projectRoot)
	if len(requirements) == 0 {
		return frameworks
	}

	// Web frameworks
	if version := requirements["django"]; version != "" {
		frameworks = append(frameworks, FrameworkInfo{
			Name:         "Django",
			Version:      version,
			Type:         FrameworkTypeWeb,
			ConfigFiles:  fd.findDjangoConfigFiles(projectRoot),
			Dependencies: []string{"django-admin"},
			Patterns:     []string{"models", "views", "templates", "urls"},
			Conventions:  fd.getDjangoConventions(projectRoot),
		})
	}

	if version := requirements["flask"]; version != "" {
		frameworks = append(frameworks, FrameworkInfo{
			Name:         "Flask",
			Version:      version,
			Type:         FrameworkTypeWeb,
			ConfigFiles:  fd.findFlaskConfigFiles(projectRoot),
			Dependencies: []string{},
			Patterns:     []string{"routes", "blueprints", "templates"},
			Conventions:  fd.getFlaskConventions(projectRoot),
		})
	}

	if version := requirements["fastapi"]; version != "" {
		frameworks = append(frameworks, FrameworkInfo{
			Name:         "FastAPI",
			Version:      version,
			Type:         FrameworkTypeAPI,
			ConfigFiles:  fd.findFastAPIConfigFiles(projectRoot),
			Dependencies: []string{"uvicorn"},
			Patterns:     []string{"routers", "dependencies", "schemas"},
			Conventions:  fd.getFastAPIConventions(projectRoot),
		})
	}

	// Testing frameworks
	if version := requirements["pytest"]; version != "" {
		frameworks = append(frameworks, FrameworkInfo{
			Name:         "pytest",
			Version:      version,
			Type:         FrameworkTypeTesting,
			ConfigFiles:  fd.findPytestConfigFiles(projectRoot),
			Dependencies: []string{},
			Patterns:     []string{"test_", "fixtures", "parametrize"},
			Conventions:  fd.getPytestConventions(projectRoot),
		})
	}

	return frameworks
}

// detectGoFrameworks detects Go frameworks
func (fd *FrameworkDetector) detectGoFrameworks(projectRoot string) []FrameworkInfo {
	var frameworks []FrameworkInfo

	goModFile := filepath.Join(projectRoot, "go.mod")
	if !fileExists(goModFile) {
		return frameworks
	}

	// #nosec G304 - goModFile is constructed from validated projectRoot, not user input
	content, err := os.ReadFile(goModFile)
	if err != nil {
		return frameworks
	}

	goModContent := string(content)

	// Web frameworks
	webFrameworks := map[string]FrameworkInfo{
		"github.com/gin-gonic/gin": {
			Name:        "Gin",
			Type:        FrameworkTypeWeb,
			Patterns:    []string{"gin.Engine", "middleware", "routes"},
			Conventions: fd.getGinConventions(projectRoot),
		},
		"github.com/gorilla/mux": {
			Name:        "Gorilla Mux",
			Type:        FrameworkTypeWeb,
			Patterns:    []string{"mux.Router", "handlers", "subrouters"},
			Conventions: fd.getGorillaMuxConventions(projectRoot),
		},
		"github.com/labstack/echo": {
			Name:        "Echo",
			Type:        FrameworkTypeWeb,
			Patterns:    []string{"echo.Echo", "middleware", "context"},
			Conventions: fd.getEchoConventions(projectRoot),
		},
		"github.com/gofiber/fiber": {
			Name:        "Fiber",
			Type:        FrameworkTypeWeb,
			Patterns:    []string{"fiber.App", "middleware", "handlers"},
			Conventions: fd.getFiberConventions(projectRoot),
		},
	}

	for dependency, framework := range webFrameworks {
		if strings.Contains(goModContent, dependency) {
			version := fd.extractGoModVersion(goModContent, dependency)
			framework.Version = version
			frameworks = append(frameworks, framework)
		}
	}

	// CLI frameworks
	cliFrameworks := map[string]FrameworkInfo{
		"github.com/spf13/cobra": {
			Name:        "Cobra",
			Type:        FrameworkTypeBuild,
			Patterns:    []string{"cobra.Command", "cmd", "flags"},
			Conventions: fd.getCobraConventions(projectRoot),
		},
		"github.com/urfave/cli": {
			Name:        "CLI",
			Type:        FrameworkTypeBuild,
			Patterns:    []string{"cli.App", "commands", "flags"},
			Conventions: fd.getUfaveCLIConventions(projectRoot),
		},
	}

	for dependency, framework := range cliFrameworks {
		if strings.Contains(goModContent, dependency) {
			version := fd.extractGoModVersion(goModContent, dependency)
			framework.Version = version
			frameworks = append(frameworks, framework)
		}
	}

	return frameworks
}

// detectJavaFrameworks detects Java frameworks
func (fd *FrameworkDetector) detectJavaFrameworks(projectRoot string) []FrameworkInfo {
	var frameworks []FrameworkInfo

	// Check Maven pom.xml
	if pomFrameworks := fd.detectFromMavenPom(projectRoot); len(pomFrameworks) > 0 {
		frameworks = append(frameworks, pomFrameworks...)
	}

	// Check Gradle build files
	if gradleFrameworks := fd.detectFromGradleBuild(projectRoot); len(gradleFrameworks) > 0 {
		frameworks = append(frameworks, gradleFrameworks...)
	}

	return frameworks
}

// detectRustFrameworks detects Rust frameworks
func (fd *FrameworkDetector) detectRustFrameworks(projectRoot string) []FrameworkInfo {
	var frameworks []FrameworkInfo

	cargoFile := filepath.Join(projectRoot, "Cargo.toml")
	if !fileExists(cargoFile) {
		return frameworks
	}

	// #nosec G304 - cargoFile is constructed from validated projectRoot, not user input
	content, err := os.ReadFile(cargoFile)
	if err != nil {
		return frameworks
	}

	cargoContent := string(content)

	rustFrameworks := map[string]FrameworkInfo{
		"actix-web": {
			Name:        "Actix Web",
			Type:        FrameworkTypeWeb,
			Patterns:    []string{"actix_web", "App", "HttpServer"},
			Conventions: fd.getActixConventions(projectRoot),
		},
		"warp": {
			Name:        "Warp",
			Type:        FrameworkTypeWeb,
			Patterns:    []string{"warp::Filter", "routes", "handlers"},
			Conventions: fd.getWarpConventions(projectRoot),
		},
		"rocket": {
			Name:        "Rocket",
			Type:        FrameworkTypeWeb,
			Patterns:    []string{"rocket", "routes", "launch"},
			Conventions: fd.getRocketConventions(projectRoot),
		},
	}

	for dependency, framework := range rustFrameworks {
		if strings.Contains(cargoContent, dependency) {
			version := fd.extractCargoVersion(cargoContent, dependency)
			framework.Version = version
			frameworks = append(frameworks, framework)
		}
	}

	return frameworks
}

// detectPHPFrameworks detects PHP frameworks
func (fd *FrameworkDetector) detectPHPFrameworks(projectRoot string) []FrameworkInfo {
	var frameworks []FrameworkInfo

	composerFile := filepath.Join(projectRoot, "composer.json")
	if !fileExists(composerFile) {
		return frameworks
	}

	// #nosec G304 - composerFile is constructed from validated projectRoot, not user input
	data, err := os.ReadFile(composerFile)
	if err != nil {
		return frameworks
	}

	var composer struct {
		Require    map[string]string `json:"require"`
		RequireDev map[string]string `json:"require-dev"`
	}

	if err := json.Unmarshal(data, &composer); err != nil {
		return frameworks
	}

	allDeps := make(map[string]string)
	for k, v := range composer.Require {
		allDeps[k] = v
	}
	for k, v := range composer.RequireDev {
		allDeps[k] = v
	}

	phpFrameworks := map[string]FrameworkInfo{
		"laravel/framework": {
			Name:        "Laravel",
			Type:        FrameworkTypeWeb,
			ConfigFiles: fd.findLaravelConfigFiles(projectRoot),
			Patterns:    []string{"artisan", "eloquent", "blade"},
			Conventions: fd.getLaravelConventions(projectRoot),
		},
		"symfony/framework-bundle": {
			Name:        "Symfony",
			Type:        FrameworkTypeWeb,
			ConfigFiles: fd.findSymfonyConfigFiles(projectRoot),
			Patterns:    []string{"console", "controller", "entity"},
			Conventions: fd.getSymfonyConventions(projectRoot),
		},
	}

	for dependency, framework := range phpFrameworks {
		if version, exists := allDeps[dependency]; exists {
			framework.Version = cleanVersion(version)
			frameworks = append(frameworks, framework)
		}
	}

	return frameworks
}

// detectRubyFrameworks detects Ruby frameworks
func (fd *FrameworkDetector) detectRubyFrameworks(projectRoot string) []FrameworkInfo {
	var frameworks []FrameworkInfo

	gemfile := filepath.Join(projectRoot, "Gemfile")
	if !fileExists(gemfile) {
		return frameworks
	}

	// #nosec G304 - gemfile is constructed from validated projectRoot, not user input
	content, err := os.ReadFile(gemfile)
	if err != nil {
		return frameworks
	}

	gemfileContent := string(content)

	if strings.Contains(gemfileContent, "gem 'rails'") ||
		strings.Contains(gemfileContent, `gem "rails"`) {
		frameworks = append(frameworks, FrameworkInfo{
			Name:        "Ruby on Rails",
			Type:        FrameworkTypeWeb,
			ConfigFiles: fd.findRailsConfigFiles(projectRoot),
			Patterns:    []string{"mvc", "activerecord", "actionpack"},
			Conventions: fd.getRailsConventions(projectRoot),
		})
	}

	return frameworks
}

// detectFromFilePatterns detects frameworks from file patterns
func (fd *FrameworkDetector) detectFromFilePatterns(projectRoot string) []FrameworkInfo {
	var frameworks []FrameworkInfo

	// Docker
	if fileExists(filepath.Join(projectRoot, "Dockerfile")) ||
		fileExists(filepath.Join(projectRoot, "docker-compose.yml")) ||
		fileExists(filepath.Join(projectRoot, "docker-compose.yaml")) {
		frameworks = append(frameworks, FrameworkInfo{
			Name:        "Docker",
			Type:        FrameworkTypeBuild,
			ConfigFiles: fd.findDockerConfigFiles(projectRoot),
			Patterns:    []string{"containerization", "images", "services"},
			Conventions: fd.getDockerConventions(projectRoot),
		})
	}

	// Kubernetes
	if fd.hasKubernetesFiles(projectRoot) {
		frameworks = append(frameworks, FrameworkInfo{
			Name:        "Kubernetes",
			Type:        FrameworkTypeBuild,
			ConfigFiles: fd.findKubernetesConfigFiles(projectRoot),
			Patterns:    []string{"pods", "services", "deployments"},
			Conventions: fd.getKubernetesConventions(projectRoot),
		})
	}

	return frameworks
}

// Helper methods for specific framework detection

func (fd *FrameworkDetector) readPythonRequirements(projectRoot string) map[string]string {
	requirements := make(map[string]string)

	reqFiles := []string{"requirements.txt", "requirements-dev.txt"}
	for _, file := range reqFiles {
		path := filepath.Join(projectRoot, file)
		if fileExists(path) {
			// #nosec G304 - path is constructed from validated projectRoot and known requirements file names
			content, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}

				// Parse package==version format
				if strings.Contains(line, "==") {
					parts := strings.Split(line, "==")
					if len(parts) == 2 {
						requirements[parts[0]] = parts[1]
					}
				} else {
					// Package without version
					requirements[line] = ""
				}
			}
		}
	}

	return requirements
}

func (fd *FrameworkDetector) extractGoModVersion(content, dependency string) string {
	regex := regexp.MustCompile(dependency + `\s+v([^\s]+)`)
	matches := regex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (fd *FrameworkDetector) extractCargoVersion(content, dependency string) string {
	regex := regexp.MustCompile(dependency + `\s*=\s*"([^"]+)"`)
	matches := regex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (fd *FrameworkDetector) detectFromMavenPom(projectRoot string) []FrameworkInfo {
	var frameworks []FrameworkInfo

	pomFile := filepath.Join(projectRoot, "pom.xml")
	if !fileExists(pomFile) {
		return frameworks
	}

	// #nosec G304 - pomFile is constructed from validated projectRoot, not user input
	content, err := os.ReadFile(pomFile)
	if err != nil {
		return frameworks
	}

	pomContent := string(content)

	if strings.Contains(pomContent, "spring-boot-starter") {
		frameworks = append(frameworks, FrameworkInfo{
			Name:        "Spring Boot",
			Type:        FrameworkTypeWeb,
			ConfigFiles: fd.findSpringBootConfigFiles(projectRoot),
			Patterns:    []string{"annotations", "auto-configuration", "starters"},
			Conventions: fd.getSpringBootConventions(projectRoot),
		})
	}

	return frameworks
}

func (fd *FrameworkDetector) detectFromGradleBuild(projectRoot string) []FrameworkInfo {
	var frameworks []FrameworkInfo

	buildFiles := []string{"build.gradle", "build.gradle.kts"}
	for _, file := range buildFiles {
		path := filepath.Join(projectRoot, file)
		if fileExists(path) {
			// #nosec G304 - path is constructed from validated projectRoot and known gradle file names
			content, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			buildContent := string(content)

			if strings.Contains(buildContent, "org.springframework.boot") {
				frameworks = append(frameworks, FrameworkInfo{
					Name:        "Spring Boot",
					Type:        FrameworkTypeWeb,
					ConfigFiles: fd.findSpringBootConfigFiles(projectRoot),
					Patterns:    []string{"annotations", "auto-configuration", "gradle"},
					Conventions: fd.getSpringBootConventions(projectRoot),
				})
			}

			if strings.Contains(buildContent, "com.android.application") {
				frameworks = append(frameworks, FrameworkInfo{
					Name:        "Android",
					Type:        FrameworkTypeMobile,
					ConfigFiles: fd.findAndroidConfigFiles(projectRoot),
					Patterns:    []string{"activities", "fragments", "layouts"},
					Conventions: fd.getAndroidConventions(projectRoot),
				})
			}
		}
	}

	return frameworks
}

func (fd *FrameworkDetector) hasKubernetesFiles(projectRoot string) bool {
	patterns := []string{
		"*.yaml", "*.yml",
		"k8s/*.yaml", "k8s/*.yml",
		"kubernetes/*.yaml", "kubernetes/*.yml",
		"manifests/*.yaml", "manifests/*.yml",
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(projectRoot, pattern))
		if err != nil {
			continue
		}
		for _, match := range matches {
			// #nosec G304 - match comes from secure filepath.Glob, not user input
			content, err := os.ReadFile(match)
			if err != nil {
				continue
			}

			contentStr := string(content)
			if strings.Contains(contentStr, "apiVersion:") &&
				(strings.Contains(contentStr, "kind: Pod") ||
					strings.Contains(contentStr, "kind: Service") ||
					strings.Contains(contentStr, "kind: Deployment")) {
				return true
			}
		}
	}

	return false
}

func (fd *FrameworkDetector) deduplicateAndEnrich(frameworks []FrameworkInfo, projectRoot string) []FrameworkInfo {
	seen := make(map[string]bool)
	var result []FrameworkInfo

	for _, framework := range frameworks {
		if !seen[framework.Name] {
			seen[framework.Name] = true
			result = append(result, framework)
		}
	}

	return result
}

func cleanVersion(version string) string {
	// Remove common prefixes like ^, ~, >=, etc.
	cleanRegex := regexp.MustCompile(`^[\^~>=<]*(.+)`)
	matches := cleanRegex.FindStringSubmatch(version)
	if len(matches) > 1 {
		return matches[1]
	}
	return version
}

// Placeholder methods for framework-specific configurations and conventions
// These would be implemented with actual logic to detect specific patterns

func (fd *FrameworkDetector) findReactConfigFiles(projectRoot string) []string {
	return fd.findConfigFiles(projectRoot, []string{".babelrc", "babel.config.js", "webpack.config.js"})
}

func (fd *FrameworkDetector) getReactConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{
		DirectoryStructure: []string{"src/components", "src/hooks", "src/contexts"},
		NamingPatterns:     map[string]string{"components": "PascalCase", "hooks": "use*"},
		ConfigurationStyle: "jsx",
		ComponentStructure: "functional-components",
		StateManagement:    "hooks",
		TestingStrategy:    "react-testing-library",
	}
}

func (fd *FrameworkDetector) findVueConfigFiles(projectRoot string) []string {
	return fd.findConfigFiles(projectRoot, []string{"vue.config.js", "vite.config.js"})
}

func (fd *FrameworkDetector) getVueConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{
		DirectoryStructure: []string{"src/components", "src/views", "src/store"},
		NamingPatterns:     map[string]string{"components": "PascalCase", "views": "PascalCase"},
		ConfigurationStyle: "vue",
		ComponentStructure: "single-file-components",
		StateManagement:    "vuex",
		TestingStrategy:    "vue-test-utils",
	}
}

// Add more placeholder methods for other frameworks...
func (fd *FrameworkDetector) findAngularConfigFiles(projectRoot string) []string {
	return fd.findConfigFiles(projectRoot, []string{"angular.json", "tsconfig.json"})
}

func (fd *FrameworkDetector) getAngularConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{
		DirectoryStructure: []string{"src/app", "src/assets", "src/environments"},
		NamingPatterns:     map[string]string{"components": "kebab-case", "services": "kebab-case"},
		ConfigurationStyle: "typescript",
		ComponentStructure: "component-decorator",
		StateManagement:    "rxjs",
		TestingStrategy:    "jasmine-karma",
	}
}

// Implement all other getXXXConventions and findXXXConfigFiles methods
// ... (shortened for brevity, but would include implementations for all frameworks)

func (fd *FrameworkDetector) findConfigFiles(projectRoot string, files []string) []string {
	var found []string
	for _, file := range files {
		if fileExists(filepath.Join(projectRoot, file)) {
			found = append(found, file)
		}
	}
	return found
}

// Placeholder implementations for remaining methods
func (fd *FrameworkDetector) findSvelteConfigFiles(projectRoot string) []string { return []string{} }
func (fd *FrameworkDetector) getSvelteConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) getExpressConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) getFastifyConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) getKoaConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) findElectronConfigFiles(projectRoot string) []string { return []string{} }
func (fd *FrameworkDetector) getElectronConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) findTestingConfigFiles(projectRoot string, framework string) []string {
	return []string{}
}
func (fd *FrameworkDetector) getTestingConventions(projectRoot string, framework string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) findBuildConfigFiles(projectRoot string, tool string) []string {
	return []string{}
}
func (fd *FrameworkDetector) getBuildToolConventions(projectRoot string, tool string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) findDjangoConfigFiles(projectRoot string) []string { return []string{} }
func (fd *FrameworkDetector) getDjangoConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) findFlaskConfigFiles(projectRoot string) []string { return []string{} }
func (fd *FrameworkDetector) getFlaskConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) findFastAPIConfigFiles(projectRoot string) []string { return []string{} }
func (fd *FrameworkDetector) getFastAPIConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) findPytestConfigFiles(projectRoot string) []string { return []string{} }
func (fd *FrameworkDetector) getPytestConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) getGinConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) getGorillaMuxConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) getEchoConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) getFiberConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) getCobraConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) getUfaveCLIConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) getActixConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) getWarpConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) getRocketConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) findLaravelConfigFiles(projectRoot string) []string { return []string{} }
func (fd *FrameworkDetector) getLaravelConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) findSymfonyConfigFiles(projectRoot string) []string { return []string{} }
func (fd *FrameworkDetector) getSymfonyConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) findRailsConfigFiles(projectRoot string) []string { return []string{} }
func (fd *FrameworkDetector) getRailsConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) findDockerConfigFiles(projectRoot string) []string { return []string{} }
func (fd *FrameworkDetector) getDockerConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) findKubernetesConfigFiles(projectRoot string) []string {
	return []string{}
}
func (fd *FrameworkDetector) getKubernetesConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) findSpringBootConfigFiles(projectRoot string) []string {
	return []string{}
}
func (fd *FrameworkDetector) getSpringBootConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
func (fd *FrameworkDetector) findAndroidConfigFiles(projectRoot string) []string { return []string{} }
func (fd *FrameworkDetector) getAndroidConventions(projectRoot string) *FrameworkConventions {
	return &FrameworkConventions{}
}
