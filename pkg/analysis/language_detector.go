package analysis

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Language name constants
const (
	langObjectiveC = "Objective-C"
	langPerl       = "Perl"
)

// LanguageDetector detects programming languages in a project
type LanguageDetector struct {
	config          AnalyzerConfig
	languageMap     map[string]*LanguageInfo
	extensionMap    map[string]string
	shebangPatterns map[string]*regexp.Regexp
}

// NewLanguageDetector creates a new language detector
func NewLanguageDetector(config AnalyzerConfig) *LanguageDetector {
	ld := &LanguageDetector{
		config:          config,
		languageMap:     make(map[string]*LanguageInfo),
		extensionMap:    getExtensionToLanguageMap(),
		shebangPatterns: getShebangPatterns(),
	}
	return ld
}

// DetectLanguages detects all programming languages used in the project
func (ld *LanguageDetector) DetectLanguages(ctx context.Context, projectRoot string) ([]LanguageInfo, error) {
	ld.languageMap = make(map[string]*LanguageInfo)

	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on errors
		}

		// Skip directories and large files
		if info.IsDir() || info.Size() > ld.config.MaxFileSize {
			return nil
		}

		// Skip ignored patterns
		relPath, err := filepath.Rel(projectRoot, path)
		if err != nil {
			relPath = path // Fallback to absolute path
		}
		if ld.shouldIgnoreFile(relPath) {
			return nil
		}

		// Detect language for this file
		if language := ld.detectFileLanguage(path, info); language != "" {
			ld.addFileToLanguage(language, path, info)
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert map to slice and calculate percentages
	languages := ld.finalizeLanguages()

	// Detect language-specific configurations and conventions
	for i := range languages {
		ld.detectLanguageSpecifics(projectRoot, &languages[i])
	}

	return languages, nil
}

// detectFileLanguage determines the language of a single file
func (ld *LanguageDetector) detectFileLanguage(path string, info os.FileInfo) string {
	// First try extension-based detection
	ext := strings.ToLower(filepath.Ext(path))
	if language, exists := ld.extensionMap[ext]; exists {
		return language
	}

	// Try filename-based detection
	filename := strings.ToLower(info.Name())
	if language := ld.detectByFilename(filename); language != "" {
		return language
	}

	// Try shebang detection for files without extensions
	if ext == "" {
		if language := ld.detectByShebang(path); language != "" {
			return language
		}
	}

	// Try content-based detection for ambiguous files
	return ld.detectByContent(path, ext)
}

// detectByFilename detects language by specific filenames
func (ld *LanguageDetector) detectByFilename(filename string) string {
	filenameMap := map[string]string{
		"makefile":        "Make",
		"dockerfile":      "Dockerfile",
		"vagrantfile":     "Ruby",
		"rakefile":        "Ruby",
		"gemfile":         "Ruby",
		"podfile":         "Ruby",
		"fastfile":        "Ruby",
		"brewfile":        "Ruby",
		"guardfile":       "Ruby",
		"capfile":         "Ruby",
		"dangerfile":      "Ruby",
		"puppetfile":      "Ruby",
		"berksfile":       "Ruby",
		"cheffile":        "Ruby",
		"thorfile":        "Ruby",
		"procfile":        "Text",
		"buildfile":       "Scala",
		"build.sbt":       "Scala",
		"pom.xml":         "Maven POM",
		"build.gradle":    "Gradle",
		"settings.gradle": "Gradle",
		"package.json":    "JSON",
		"composer.json":   "JSON",
		"bower.json":      "JSON",
		"tsconfig.json":   "JSON",
		"tslint.json":     "JSON",
		"eslintrc.json":   "JSON",
		"babelrc":         "JSON",
		"prettierrc":      "JSON",
		"editorconfig":    "EditorConfig",
		"gitignore":       "Ignore List",
		"gitattributes":   "Git Attributes",
		"npmignore":       "Ignore List",
		"dockerignore":    "Ignore List",
		"eslintignore":    "Ignore List",
		"cargo.toml":      "TOML",
		"cargo.lock":      "TOML",
		"pyproject.toml":  "TOML",
		"poetry.lock":     "TOML",
		"pipfile":         "TOML",
		"go.mod":          "Go Module",
		"go.sum":          "Go Module",
	}

	return filenameMap[filename]
}

// detectByShebang detects language from shebang lines
func (ld *LanguageDetector) detectByShebang(path string) string {
	// #nosec G304 - path comes from validated file system walk, not user input
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = file.Close() }() //nolint:errcheck // Close in defer is best effort

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		firstLine := scanner.Text()
		if strings.HasPrefix(firstLine, "#!") {
			for language, pattern := range ld.shebangPatterns {
				if pattern.MatchString(firstLine) {
					return language
				}
			}
		}
	}

	return ""
}

// detectByContent detects language by analyzing file content
func (ld *LanguageDetector) detectByContent(path, ext string) string {
	// For ambiguous extensions, analyze content
	switch ext {
	case ".h":
		return ld.detectCOrCpp(path)
	case ".m":
		return ld.detectObjectiveCOrMatlab(path)
	case ".r":
		return ld.detectROrRebol(path)
	case ".pl":
		return ld.detectPerlOrProlog(path)
	case ".t":
		return ld.detectPerlTest(path)
	}

	return ""
}

// detectCOrCpp distinguishes between C and C++ header files
func (ld *LanguageDetector) detectCOrCpp(path string) string {
	content, err := ld.readFileContent(path, 8192) // Read first 8KB
	if err != nil {
		return "C" // Default to C
	}

	cppIndicators := []string{
		"#include <iostream>",
		"#include <vector>",
		"#include <string>",
		"#include <algorithm>",
		"namespace ",
		"class ",
		"template<",
		"std::",
		"extern \"C\"",
		"using namespace",
		"public:",
		"private:",
		"protected:",
	}

	for _, indicator := range cppIndicators {
		if strings.Contains(content, indicator) {
			return "C++"
		}
	}

	return "C"
}

// detectObjectiveCOrMatlab distinguishes between Objective-C and MATLAB
func (ld *LanguageDetector) detectObjectiveCOrMatlab(path string) string {
	content, err := ld.readFileContent(path, 4096)
	if err != nil {
		return langObjectiveC
	}

	// Look for Objective-C indicators
	objcIndicators := []string{
		"#import",
		"@interface",
		"@implementation",
		"@end",
		"@property",
		"@synthesize",
		"@class",
		"NSString",
		"NSArray",
		"NSObject",
	}

	for _, indicator := range objcIndicators {
		if strings.Contains(content, indicator) {
			return langObjectiveC
		}
	}

	// Look for MATLAB indicators
	matlabIndicators := []string{
		"function ",
		"end\n",
		"fprintf(",
		"plot(",
		"figure(",
		"disp(",
		"clc;",
		"clear;",
	}

	for _, indicator := range matlabIndicators {
		if strings.Contains(content, indicator) {
			return "MATLAB"
		}
	}

	return langObjectiveC // Default
}

// detectROrRebol distinguishes between R and REBOL
func (ld *LanguageDetector) detectROrRebol(path string) string {
	content, err := ld.readFileContent(path, 2048)
	if err != nil {
		return "R"
	}

	if strings.Contains(content, "REBOL") {
		return "REBOL"
	}

	return "R"
}

// detectPerlOrProlog distinguishes between Perl and Prolog
func (ld *LanguageDetector) detectPerlOrProlog(path string) string {
	content, err := ld.readFileContent(path, 2048)
	if err != nil {
		return langPerl
	}

	// Look for Prolog indicators
	if strings.Contains(content, ":-") ||
		strings.Contains(content, "?-") ||
		regexp.MustCompile(`^[a-z][a-zA-Z0-9_]*\(`).MatchString(content) {
		return "Prolog"
	}

	return langPerl
}

// detectPerlTest detects Perl test files
func (ld *LanguageDetector) detectPerlTest(path string) string {
	if strings.Contains(path, "/t/") || strings.HasSuffix(path, ".t") {
		return langPerl
	}
	return ""
}

// addFileToLanguage adds a file to the language statistics
func (ld *LanguageDetector) addFileToLanguage(language, path string, info os.FileInfo) {
	if lang, exists := ld.languageMap[language]; exists {
		lang.FileCount++
		lang.LineCount += ld.countLines(path)
	} else {
		lineCount := ld.countLines(path)
		ld.languageMap[language] = &LanguageInfo{
			Name:        language,
			FileCount:   1,
			LineCount:   lineCount,
			Extensions:  []string{strings.ToLower(filepath.Ext(path))},
			Conventions: &LanguageConventions{},
		}
	}
}

// countLines counts the number of lines in a file
func (ld *LanguageDetector) countLines(path string) int {
	// #nosec G304 - path comes from validated file system walk, not user input
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer func() { _ = file.Close() }() //nolint:errcheck // Close in defer is best effort

	lineCount := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineCount++
	}

	return lineCount
}

// finalizeLanguages converts the language map to a sorted slice with percentages
func (ld *LanguageDetector) finalizeLanguages() []LanguageInfo {
	totalLines := 0
	for _, lang := range ld.languageMap {
		totalLines += lang.LineCount
	}

	var languages []LanguageInfo
	for _, lang := range ld.languageMap {
		if totalLines > 0 {
			lang.Percentage = float64(lang.LineCount) / float64(totalLines) * 100
		}
		languages = append(languages, *lang)
	}

	// Sort by line count (descending)
	sort.Slice(languages, func(i, j int) bool {
		return languages[i].LineCount > languages[j].LineCount
	})

	return languages
}

// detectLanguageSpecifics detects language-specific configuration and conventions
func (ld *LanguageDetector) detectLanguageSpecifics(projectRoot string, lang *LanguageInfo) {
	switch lang.Name {
	case "JavaScript", "TypeScript":
		ld.detectJavaScriptSpecifics(projectRoot, lang)
	case "Python":
		ld.detectPythonSpecifics(projectRoot, lang)
	case "Go":
		ld.detectGoSpecifics(projectRoot, lang)
	case "Java":
		ld.detectJavaSpecifics(projectRoot, lang)
	case "Rust":
		ld.detectRustSpecifics(projectRoot, lang)
	case "PHP":
		ld.detectPHPSpecifics(projectRoot, lang)
	case "Ruby":
		ld.detectRubySpecifics(projectRoot, lang)
	}
}

// detectJavaScriptSpecifics detects JavaScript/TypeScript specific information
func (ld *LanguageDetector) detectJavaScriptSpecifics(projectRoot string, lang *LanguageInfo) {
	configFiles := []string{
		"package.json", "package-lock.json", "yarn.lock", "pnpm-lock.yaml",
		"tsconfig.json", "jsconfig.json", "babel.config.js", ".babelrc",
		"webpack.config.js", "rollup.config.js", "vite.config.js",
		".eslintrc.js", ".eslintrc.json", "prettier.config.js", ".prettierrc",
		"jest.config.js", "vitest.config.js", "cypress.json",
	}

	for _, file := range configFiles {
		if fileExists(filepath.Join(projectRoot, file)) {
			lang.ConfigFiles = append(lang.ConfigFiles, file)
		}
	}

	// Detect Node.js version from .nvmrc
	if fileExists(filepath.Join(projectRoot, ".nvmrc")) {
		lang.ConfigFiles = append(lang.ConfigFiles, ".nvmrc")
		// #nosec G304 - filepath constructed from validated projectRoot and known config file name
		if content, err := os.ReadFile(filepath.Join(projectRoot, ".nvmrc")); err == nil {
			lang.Version = strings.TrimSpace(string(content))
		}
	}

	lang.Conventions = &LanguageConventions{
		NamingStyle:        "camelCase",
		FileStructure:      "flat-modules",
		ImportStyle:        "ES6-modules",
		ErrorHandling:      "try-catch",
		TestingPattern:     "jest",
		DocumentationStyle: "JSDoc",
		CommonPatterns:     []string{"async-await", "promises", "event-driven"},
	}
}

// detectPythonSpecifics detects Python specific information
func (ld *LanguageDetector) detectPythonSpecifics(projectRoot string, lang *LanguageInfo) {
	configFiles := []string{
		"requirements.txt", "requirements-dev.txt", "setup.py", "setup.cfg",
		"pyproject.toml", "poetry.lock", "Pipfile", "Pipfile.lock",
		"pytest.ini", "tox.ini", ".pylintrc", "mypy.ini",
		".python-version", "runtime.txt",
	}

	for _, file := range configFiles {
		if fileExists(filepath.Join(projectRoot, file)) {
			lang.ConfigFiles = append(lang.ConfigFiles, file)
		}
	}

	// Detect Python version from .python-version
	if fileExists(filepath.Join(projectRoot, ".python-version")) {
		// #nosec G304 - filepath constructed from validated projectRoot and known config file name
		if content, err := os.ReadFile(filepath.Join(projectRoot, ".python-version")); err == nil {
			lang.Version = strings.TrimSpace(string(content))
		}
	}

	lang.Conventions = &LanguageConventions{
		NamingStyle:        "snake_case",
		FileStructure:      "packages",
		ImportStyle:        "absolute-imports",
		ErrorHandling:      "exceptions",
		TestingPattern:     "pytest",
		DocumentationStyle: "docstrings",
		CommonPatterns:     []string{"decorators", "context-managers", "generators"},
	}
}

// detectGoSpecifics detects Go specific information
func (ld *LanguageDetector) detectGoSpecifics(projectRoot string, lang *LanguageInfo) {
	configFiles := []string{"go.mod", "go.sum", "go.work", "go.work.sum"}

	for _, file := range configFiles {
		if fileExists(filepath.Join(projectRoot, file)) {
			lang.ConfigFiles = append(lang.ConfigFiles, file)
		}
	}

	// Extract Go version from go.mod
	// #nosec G304 - filepath constructed from validated projectRoot and known config file name
	if content, err := os.ReadFile(filepath.Join(projectRoot, "go.mod")); err == nil {
		goVersionRegex := regexp.MustCompile(`go\s+(\d+\.\d+(?:\.\d+)?)`)
		if match := goVersionRegex.FindStringSubmatch(string(content)); len(match) > 1 {
			lang.Version = match[1]
		}
	}

	lang.Conventions = &LanguageConventions{
		NamingStyle:        "mixedCase",
		FileStructure:      "packages",
		ImportStyle:        "package-imports",
		ErrorHandling:      "explicit-errors",
		TestingPattern:     "testing-package",
		DocumentationStyle: "godoc",
		CommonPatterns:     []string{"interfaces", "goroutines", "channels"},
	}
}

// detectJavaSpecifics detects Java specific information
func (ld *LanguageDetector) detectJavaSpecifics(projectRoot string, lang *LanguageInfo) {
	configFiles := []string{
		"pom.xml", "build.gradle", "build.gradle.kts", "settings.gradle",
		"gradle.properties", "gradlew", "mvnw", "ivy.xml",
		"application.properties", "application.yml",
	}

	for _, file := range configFiles {
		if fileExists(filepath.Join(projectRoot, file)) {
			lang.ConfigFiles = append(lang.ConfigFiles, file)
		}
	}

	lang.Conventions = &LanguageConventions{
		NamingStyle:        "camelCase",
		FileStructure:      "packages",
		ImportStyle:        "package-imports",
		ErrorHandling:      "exceptions",
		TestingPattern:     "junit",
		DocumentationStyle: "javadoc",
		CommonPatterns:     []string{"spring-framework", "dependency-injection", "annotations"},
	}
}

// detectRustSpecifics detects Rust specific information
func (ld *LanguageDetector) detectRustSpecifics(projectRoot string, lang *LanguageInfo) {
	configFiles := []string{"Cargo.toml", "Cargo.lock", "rust-toolchain", "rustfmt.toml"}

	for _, file := range configFiles {
		if fileExists(filepath.Join(projectRoot, file)) {
			lang.ConfigFiles = append(lang.ConfigFiles, file)
		}
	}

	lang.Conventions = &LanguageConventions{
		NamingStyle:        "snake_case",
		FileStructure:      "modules",
		ImportStyle:        "use-statements",
		ErrorHandling:      "result-option",
		TestingPattern:     "built-in-tests",
		DocumentationStyle: "doc-comments",
		CommonPatterns:     []string{"ownership", "borrowing", "traits"},
	}
}

// detectPHPSpecifics detects PHP specific information
func (ld *LanguageDetector) detectPHPSpecifics(projectRoot string, lang *LanguageInfo) {
	configFiles := []string{
		"composer.json", "composer.lock", "composer.phar",
		"phpunit.xml", "phpunit.xml.dist", "phpcs.xml", "psalm.xml",
		".php_cs", ".php_cs.dist", "phpstan.neon",
	}

	for _, file := range configFiles {
		if fileExists(filepath.Join(projectRoot, file)) {
			lang.ConfigFiles = append(lang.ConfigFiles, file)
		}
	}

	lang.Conventions = &LanguageConventions{
		NamingStyle:        "camelCase",
		FileStructure:      "namespaces",
		ImportStyle:        "use-statements",
		ErrorHandling:      "exceptions",
		TestingPattern:     "phpunit",
		DocumentationStyle: "phpdoc",
		CommonPatterns:     []string{"mvc", "dependency-injection", "traits"},
	}
}

// detectRubySpecifics detects Ruby specific information
func (ld *LanguageDetector) detectRubySpecifics(projectRoot string, lang *LanguageInfo) {
	configFiles := []string{
		"Gemfile", "Gemfile.lock", ".ruby-version", ".rvmrc",
		"config.ru", "Rakefile", ".rspec", "spec_helper.rb",
	}

	for _, file := range configFiles {
		if fileExists(filepath.Join(projectRoot, file)) {
			lang.ConfigFiles = append(lang.ConfigFiles, file)
		}
	}

	// Detect Ruby version from .ruby-version
	if fileExists(filepath.Join(projectRoot, ".ruby-version")) {
		// #nosec G304 - filepath constructed from validated projectRoot and known config file name
		if content, err := os.ReadFile(filepath.Join(projectRoot, ".ruby-version")); err == nil {
			lang.Version = strings.TrimSpace(string(content))
		}
	}

	lang.Conventions = &LanguageConventions{
		NamingStyle:        "snake_case",
		FileStructure:      "gems",
		ImportStyle:        "require-statements",
		ErrorHandling:      "exceptions",
		TestingPattern:     "rspec",
		DocumentationStyle: "yard",
		CommonPatterns:     []string{"metaprogramming", "blocks", "mixins"},
	}
}

// shouldIgnoreFile checks if a file should be ignored based on patterns
func (ld *LanguageDetector) shouldIgnoreFile(relPath string) bool {
	for _, pattern := range ld.config.IgnorePatterns {
		if matched, err := filepath.Match(pattern, relPath); err == nil && matched {
			return true
		}
		// Handle directory patterns
		if strings.HasSuffix(pattern, "/**") {
			dirPattern := strings.TrimSuffix(pattern, "/**")
			if strings.HasPrefix(relPath, dirPattern+"/") {
				return true
			}
		}
	}
	return false
}

// readFileContent reads a limited amount of content from a file
func (ld *LanguageDetector) readFileContent(path string, maxBytes int) (string, error) {
	// #nosec G304 - path comes from validated file system walk, not user input
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }() //nolint:errcheck // Close in defer is best effort

	buffer := make([]byte, maxBytes)
	n, err := file.Read(buffer)
	if err != nil && err.Error() != "EOF" {
		return "", err
	}

	return string(buffer[:n]), nil
}

// getExtensionToLanguageMap returns a map of file extensions to languages
func getExtensionToLanguageMap() map[string]string {
	return map[string]string{
		// Popular languages
		".js":    "JavaScript",
		".jsx":   "JavaScript",
		".ts":    "TypeScript",
		".tsx":   "TypeScript",
		".py":    "Python",
		".pyw":   "Python",
		".go":    "Go",
		".java":  "Java",
		".kt":    "Kotlin",
		".rs":    "Rust",
		".php":   "PHP",
		".rb":    "Ruby",
		".cpp":   "C++",
		".cc":    "C++",
		".cxx":   "C++",
		".c++":   "C++",
		".c":     "C",
		".cs":    "C#",
		".swift": "Swift",
		".m":     langObjectiveC,
		".mm":    "Objective-C++",
		".h":     "C",
		".hpp":   "C++",
		".hxx":   "C++",
		".scala": "Scala",
		".clj":   "Clojure",
		".cljs":  "ClojureScript",
		".hs":    "Haskell",
		".ml":    "OCaml",
		".fs":    "F#",
		".ex":    "Elixir",
		".exs":   "Elixir",
		".erl":   "Erlang",
		".dart":  "Dart",
		".lua":   "Lua",
		".r":     "R",
		".R":     "R",
		".perl":  langPerl,
		".pl":    langPerl,
		".pm":    langPerl,
		".sh":    "Shell",
		".bash":  "Bash",
		".zsh":   "Zsh",
		".fish":  "Fish",
		".ps1":   "PowerShell",

		// Web technologies
		".html":   "HTML",
		".htm":    "HTML",
		".css":    "CSS",
		".scss":   "SCSS",
		".sass":   "Sass",
		".less":   "Less",
		".vue":    "Vue",
		".svelte": "Svelte",

		// Data formats
		".json": "JSON",
		".xml":  "XML",
		".yaml": "YAML",
		".yml":  "YAML",
		".toml": "TOML",
		".ini":  "INI",
		".cfg":  "INI",
		".conf": "Configuration",

		// Documentation
		".md":       "Markdown",
		".markdown": "Markdown",
		".rst":      "reStructuredText",
		".tex":      "TeX",
		".adoc":     "AsciiDoc",

		// Scripts and config
		".sql":        "SQL",
		".dockerfile": "Dockerfile",
		".makefile":   "Makefile",
		".cmake":      "CMake",
		".gradle":     "Gradle",
		".groovy":     "Groovy",

		// Others
		".vim":     "Vim Script",
		".proto":   "Protocol Buffer",
		".thrift":  "Thrift",
		".graphql": "GraphQL",
		".gql":     "GraphQL",
	}
}

// getShebangPatterns returns regex patterns for detecting languages from shebangs
func getShebangPatterns() map[string]*regexp.Regexp {
	patterns := make(map[string]*regexp.Regexp)

	patterns["Python"] = regexp.MustCompile(`python[0-9.]*`)
	patterns["Ruby"] = regexp.MustCompile(`ruby`)
	patterns[langPerl] = regexp.MustCompile(`perl`)
	patterns["Shell"] = regexp.MustCompile(`/bin/(ba)?sh`)
	patterns["Bash"] = regexp.MustCompile(`/bin/bash`)
	patterns["Zsh"] = regexp.MustCompile(`/bin/zsh`)
	patterns["Fish"] = regexp.MustCompile(`/bin/fish`)
	patterns["Node.js"] = regexp.MustCompile(`node`)
	patterns["PHP"] = regexp.MustCompile(`php`)
	patterns["Lua"] = regexp.MustCompile(`lua`)
	patterns["AWK"] = regexp.MustCompile(`awk`)
	patterns["Tcl"] = regexp.MustCompile(`tclsh`)

	return patterns
}
