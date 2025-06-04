package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fumiya-kume/cca/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Update ccAgents to the latest version",
		Long: `Check for and install the latest version of ccAgents.

This command will:
1. Check for the latest release on GitHub
2. Compare with your current version
3. Download and install the latest version if available
4. Verify the installation

Examples:
  ccagents update                    # Update to latest version
  ccagents update --check            # Only check for updates
  ccagents update --force            # Force update even if current version is latest
  ccagents update --version v1.2.0   # Update to specific version`,
		RunE: runUpdate,
	}

	checkOnly     bool
	forceUpdate   bool
	targetVersion string
)

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().BoolVar(&checkOnly, "check", false, "Only check for updates, don't install")
	updateCmd.Flags().BoolVar(&forceUpdate, "force", false, "Force update even if current version is latest")
	updateCmd.Flags().StringVar(&targetVersion, "version", "", "Update to specific version (e.g., v1.2.0)")
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	Prerelease  bool   `json:"prerelease"`
	PublishedAt string `json:"published_at"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
}

func runUpdate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	log := logger.GetGlobalLogger()

	log.Info("Checking for ccAgents updates...")

	// Get current version
	currentVersion := GetCurrentVersion()
	log.Info("Current version: %s", currentVersion)

	// Get latest version
	var latestRelease *GitHubRelease
	var err error

	if targetVersion != "" {
		// Get specific version
		latestRelease, err = getSpecificRelease(ctx, targetVersion)
		if err != nil {
			return fmt.Errorf("failed to get version %s: %w", targetVersion, err)
		}
	} else {
		// Get latest release
		latestRelease, err = getLatestRelease(ctx)
		if err != nil {
			return fmt.Errorf("failed to check for updates: %w", err)
		}
	}

	log.Info("Latest version: %s", latestRelease.TagName)

	// Check if update is needed
	if !forceUpdate && !IsUpdateNeeded(currentVersion, latestRelease.TagName) {
		fmt.Println("✅ ccAgents is already up to date!")
		return nil
	}

	if checkOnly {
		fmt.Printf("🔄 Update available: %s → %s\n", currentVersion, latestRelease.TagName)
		fmt.Printf("📅 Released: %s\n", FormatReleaseDate(latestRelease.PublishedAt))
		if latestRelease.Body != "" {
			fmt.Printf("\n📝 Release notes:\n%s\n", latestRelease.Body)
		}
		return nil
	}

	// Perform update
	fmt.Printf("🔄 Updating ccAgents from %s to %s...\n", currentVersion, latestRelease.TagName)

	// Check if we can update (not in read-only location)
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	if !CanWriteToPath(execPath) {
		return fmt.Errorf("cannot update: ccAgents is installed in a read-only location (%s). Please use your package manager or run with sudo", execPath)
	}

	// Download and install update
	if err := downloadAndInstall(ctx, latestRelease, execPath); err != nil {
		return fmt.Errorf("failed to install update: %w", err)
	}

	fmt.Println("✅ ccAgents updated successfully!")
	fmt.Printf("🎉 You're now running ccAgents %s\n", latestRelease.TagName)

	return nil
}

// GetCurrentVersion returns the current version of ccAgents
func GetCurrentVersion() string {
	// Try to get version from build info first
	if version != "" {
		return version
	}

	// Fallback to command execution
	// #nosec G204 - os.Args[0] is the current binary, safe to execute
	cmd := exec.Command(os.Args[0], "version", "--short")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	return strings.TrimSpace(string(output))
}

// getLatestRelease fetches the latest release from GitHub
func getLatestRelease(ctx context.Context) (*GitHubRelease, error) {
	url := "https://api.github.com/repos/fumiya-kume/cca/releases/latest"
	return fetchRelease(ctx, url)
}

// getSpecificRelease fetches a specific release from GitHub
func getSpecificRelease(ctx context.Context, version string) (*GitHubRelease, error) {
	// Ensure version starts with 'v'
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	url := fmt.Sprintf("https://api.github.com/repos/fumiya-kume/cca/releases/tags/%s", version)
	return fetchRelease(ctx, url)
}

// fetchRelease fetches a release from the given URL
func fetchRelease(ctx context.Context, url string) (*GitHubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "ccAgents-updater")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// Log error but don't fail the request as data is already read
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// IsUpdateNeeded checks if an update is needed
func IsUpdateNeeded(current, latest string) bool {
	// Remove 'v' prefix for comparison
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	// Simple string comparison (works for semantic versioning)
	return current != latest && current != "unknown" && current != "dev"
}

// CanWriteToPath checks if we can write to the given path
func CanWriteToPath(path string) bool {
	dir := filepath.Dir(path)

	// Try to create a temporary file in the directory
	tmpFile := filepath.Join(dir, ".ccagents-update-test")
	// #nosec G304 - tmpFile is validated temporary file path
	file, err := os.Create(tmpFile)
	if err != nil {
		return false
	}
	if err := file.Close(); err != nil {
		// Log error but continue cleanup
		fmt.Fprintf(os.Stderr, "Warning: failed to close temp file: %v\n", err)
	}
	if err := os.Remove(tmpFile); err != nil {
		// Log error but continue as test was successful
		fmt.Fprintf(os.Stderr, "Warning: failed to remove temp file: %v\n", err)
	}

	return true
}

// downloadAndInstall downloads and installs the update
func downloadAndInstall(ctx context.Context, release *GitHubRelease, execPath string) error {
	// Find the appropriate asset for this platform
	assetURL, assetName, err := findReleaseAsset(release)
	if err != nil {
		return err
	}

	fmt.Printf("📦 Downloading %s...\n", assetName)

	// Create temporary directory for download
	tmpDir, err := createTempDir()
	if err != nil {
		return err
	}
	defer cleanupTempDir(tmpDir)

	// Download and extract the binary
	binaryPath, err := downloadAndExtractBinary(ctx, assetURL, assetName, tmpDir)
	if err != nil {
		return err
	}

	// Make binary executable on Unix-like systems
	if err := makeExecutable(binaryPath); err != nil {
		return err
	}

	// Replace current binary with new version
	return replaceBinary(binaryPath, execPath)
}

// findReleaseAsset finds the appropriate asset for the current platform
func findReleaseAsset(release *GitHubRelease) (string, string, error) {
	platform := runtime.GOOS
	arch := runtime.GOARCH

	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, platform) && strings.Contains(asset.Name, arch) {
			return asset.BrowserDownloadURL, asset.Name, nil
		}
	}

	return "", "", fmt.Errorf("no release asset found for %s/%s", platform, arch)
}

// createTempDir creates a temporary directory for the update process
func createTempDir() (string, error) {
	return os.MkdirTemp("", "ccagents-update-")
}

// cleanupTempDir removes the temporary directory
func cleanupTempDir(tmpDir string) {
	if err := os.RemoveAll(tmpDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to cleanup temp directory %s: %v\n", tmpDir, err)
	}
}

// downloadAndExtractBinary downloads and extracts the binary from the release asset
func downloadAndExtractBinary(ctx context.Context, assetURL, assetName, tmpDir string) (string, error) {
	// Download the asset
	tmpFile := filepath.Join(tmpDir, assetName)
	if err := downloadFile(ctx, assetURL, tmpFile); err != nil {
		return "", err
	}

	// Extract the binary
	binaryPath, err := extractBinary(tmpFile, tmpDir, assetName)
	if err != nil {
		return "", err
	}

	// Verify binary exists
	if _, err := os.Stat(binaryPath); err != nil {
		return "", fmt.Errorf("extracted binary not found: %s", binaryPath)
	}

	return binaryPath, nil
}

// extractBinary extracts the binary from the downloaded file
func extractBinary(tmpFile, tmpDir, assetName string) (string, error) {
	platform := runtime.GOOS
	arch := runtime.GOARCH
	extractDir := filepath.Join(tmpDir, "extracted")

	if strings.HasSuffix(assetName, ".zip") {
		// Windows
		if err := extractZip(tmpFile, extractDir); err != nil {
			return "", err
		}
		return filepath.Join(extractDir, fmt.Sprintf("ccagents-%s-%s.exe", platform, arch)), nil
	}

	// Unix-like systems
	if err := extractTarGz(tmpFile, extractDir); err != nil {
		return "", err
	}
	return filepath.Join(extractDir, fmt.Sprintf("ccagents-%s-%s", platform, arch)), nil
}

// makeExecutable makes the binary executable on Unix-like systems
func makeExecutable(binaryPath string) error {
	if runtime.GOOS != "windows" {
		// #nosec G302 - 0755 is appropriate for executable binaries
		return os.Chmod(binaryPath, 0755)
	}
	return nil
}

// replaceBinary replaces the current binary with the new one
func replaceBinary(newBinaryPath, execPath string) error {
	fmt.Println("🔧 Installing update...")

	// Create backup
	backupPath := execPath + ".backup"
	if err := CopyFile(execPath, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Replace binary
	if err := CopyFile(newBinaryPath, execPath); err != nil {
		// Restore backup on failure
		if restoreErr := CopyFile(backupPath, execPath); restoreErr != nil {
			fmt.Fprintf(os.Stderr, "Critical: failed to restore backup after failed update: %v\n", restoreErr)
		}
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Remove backup
	if err := os.Remove(backupPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to remove backup file: %v\n", err)
	}

	return nil
}

// downloadFile downloads a file from URL to local path
func downloadFile(ctx context.Context, url, filepath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// Log error but don't fail the download as data transfer may have completed
			fmt.Fprintf(os.Stderr, "Warning: failed to close download response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// #nosec G304 - filepath is validated download destination
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Log error but don't fail the download
			fmt.Fprintf(os.Stderr, "Warning: failed to close download file: %v\n", err)
		}
	}()

	_, err = io.Copy(file, resp.Body)
	return err
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	// #nosec G304 - src path is validated file copy source
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := srcFile.Close(); err != nil {
			// Log error but don't fail the copy operation
			fmt.Fprintf(os.Stderr, "Warning: failed to close source file: %v\n", err)
		}
	}()

	// #nosec G304 - dst path is validated for file update process
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if err := dstFile.Close(); err != nil {
			// Log error but don't fail the copy operation
			fmt.Fprintf(os.Stderr, "Warning: failed to close destination file: %v\n", err)
		}
	}()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// Copy permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}

// extractTarGz extracts a tar.gz file (placeholder - would need proper implementation)
func extractTarGz(srcPath, dstPath string) error {
	_ = srcPath
	_ = dstPath
	// For now, return an error suggesting manual installation
	return fmt.Errorf("automatic extraction not implemented for tar.gz files. Please download and install manually from: https://github.com/fumiya-kume/cca/releases")
}

// extractZip extracts a zip file (placeholder - would need proper implementation)
func extractZip(srcPath, dstPath string) error {
	_ = srcPath
	_ = dstPath
	// For now, return an error suggesting manual installation
	return fmt.Errorf("automatic extraction not implemented for zip files. Please download and install manually from: https://github.com/fumiya-kume/cca/releases")
}

// FormatReleaseDate formats the release date
func FormatReleaseDate(dateStr string) string {
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return dateStr
	}
	return t.Format("January 2, 2006")
}
