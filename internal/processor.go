package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Processor handles the main workflow for issue-to-PR automation
type Processor struct{}

// NewProcessor creates a new processor instance
func NewProcessor() *Processor {
	return &Processor{}
}

// ProcessIssue handles the complete workflow from issue to PR
func (p *Processor) ProcessIssue(issueURL string) error {
	// 1. Fetch issue
	fmt.Println("🔍 Fetching issue...")
	issue, err := p.fetchIssue(issueURL)
	if err != nil {
		return fmt.Errorf("failed to fetch issue: %w", err)
	}
	fmt.Printf("✅ Issue fetched: \"%s\"\n\n", issue.Title)

	// 2. Generate code
	fmt.Println("🤖 Generating code with Claude...")
	changes, err := p.generateCode(issue)
	if err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}
	fmt.Printf("✅ Code generated: %d files changed\n\n", len(changes.Files))

	// 3. Verification loop (max 3 retries)
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Apply changes
		if err := p.applyChanges(changes); err != nil {
			return fmt.Errorf("failed to apply changes: %w", err)
		}

		// Run verification
		fmt.Println("🔧 Running verification (.cca/verify.sh)...")
		verifyErr := p.runVerification()
		if verifyErr == nil {
			fmt.Println("✅ Verification passed")
			break
		}

		// Verification failed
		if attempt == maxRetries {
			return fmt.Errorf("verification failed after %d attempts: %w", maxRetries, verifyErr)
		}

		fmt.Printf("❌ Verification failed: %v\n\n", verifyErr)
		fmt.Printf("🔄 Verification failed (attempt %d/%d), asking Claude to fix...\n", attempt, maxRetries)

		// Ask Claude to fix
		fmt.Println("🤖 Claude fixing verification errors...")
		changes, err = p.fixWithClaude(changes, verifyErr.Error())
		if err != nil {
			return fmt.Errorf("failed to fix with Claude: %w", err)
		}
		fmt.Printf("✅ Code updated: %d files changed\n\n", len(changes.Files))
	}

	// 4. Git operations
	fmt.Printf("📝 Creating branch cca/issue-%d...\n", issue.Number)
	if err := p.gitOperations(issue); err != nil {
		return fmt.Errorf("git operations failed: %w", err)
	}
	fmt.Println("✅ Changes committed and pushed")

	// 5. Create PR
	fmt.Println("🎯 Creating pull request...")
	prURL, err := p.createPR(issue)
	if err != nil {
		return fmt.Errorf("failed to create PR: %w", err)
	}
	fmt.Printf("✅ Pull request created: %s\n\n", prURL)

	return nil
}

// fetchIssue uses GitHub CLI to fetch issue details
func (p *Processor) fetchIssue(issueURL string) (*Issue, error) {
	cmd := exec.Command("gh", "issue", "view", issueURL, "--json", "number,title,body,url")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh command failed: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal(output, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse issue JSON: %w", err)
	}

	// Extract repository from URL
	parts := strings.Split(issueURL, "/")
	if len(parts) >= 5 {
		issue.Repository = fmt.Sprintf("%s/%s", parts[3], parts[4])
	}

	return &issue, nil
}

// generateCode uses Claude CLI to generate implementation
func (p *Processor) generateCode(issue *Issue) (*CodeChanges, error) {
	prompt := fmt.Sprintf(`Implement a solution for this GitHub issue:

Issue: %s
Description: %s
Repository: %s

Analyze the issue and provide a complete implementation including:
1. All necessary code changes
2. Tests for the implementation  
3. Any documentation updates needed

Return the implementation as file paths and their complete content.

Format as JSON:
{
  "files": {
    "path/to/file.go": "complete file content...",
    "path/to/test.go": "test file content..."
  },
  "new_files": ["list", "of", "new", "files"],
  "deleted_files": ["list", "of", "deleted", "files"],
  "summary": "Brief description of changes made"
}`, issue.Title, issue.Body, issue.Repository)

	cmd := exec.Command("claude", "--no-confirmation", "-p", prompt) // #nosec G204 - command arguments are controlled
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("claude command failed: %w", err)
	}

	// Parse JSON from Claude's response
	var changes CodeChanges
	if err := json.Unmarshal(output, &changes); err != nil {
		return nil, fmt.Errorf("failed to parse Claude response: %w", err)
	}

	return &changes, nil
}

// applyChanges writes the generated code to disk
func (p *Processor) applyChanges(changes *CodeChanges) error {
	// Delete files first
	for _, path := range changes.DeletedFiles {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete %s: %w", path, err)
		}
	}

	// Write/update files
	for path, content := range changes.Files {
		// Create directory if needed
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Write file
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to write file %s: %w", path, err)
		}
	}

	return nil
}

// runVerification executes the verification script
func (p *Processor) runVerification() error {
	verifyPath := ".cca/verify.sh"

	// Check if verify.sh exists, create if not
	if _, err := os.Stat(verifyPath); os.IsNotExist(err) {
		if err := p.createVerificationScript(); err != nil {
			return fmt.Errorf("failed to create verification script: %w", err)
		}
	}

	// Run verification script
	cmd := exec.Command("bash", verifyPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(output))
	}

	return nil
}

// createVerificationScript creates a default verification script
func (p *Processor) createVerificationScript() error {
	verifyDir := ".cca"
	verifyPath := filepath.Join(verifyDir, "verify.sh")

	// Create directory
	if err := os.MkdirAll(verifyDir, 0750); err != nil {
		return err
	}

	// Default verification script
	content := `#!/bin/bash
# Add your build, test, and lint commands here
# Examples:
# go build ./...
# go test ./...
# golangci-lint run

echo "No verification script configured - skipping checks"
exit 0
`

	// Write script
	if err := os.WriteFile(verifyPath, []byte(content), 0700); err != nil {
		return err
	}

	return nil
}

// fixWithClaude asks Claude to fix verification errors
func (p *Processor) fixWithClaude(currentChanges *CodeChanges, verifyErrors string) (*CodeChanges, error) {
	// Serialize current changes
	changesJSON, err := json.MarshalIndent(currentChanges, "", "  ")
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf(`The verification script failed with these errors:

%s

Here are the current code changes:
%s

Please fix the code to resolve these verification errors. Return the corrected implementation.

Format as JSON with the same structure as before:
{
  "files": {...},
  "new_files": [...],
  "deleted_files": [...],
  "summary": "Description of fixes applied"
}`, verifyErrors, string(changesJSON))

	cmd := exec.Command("claude", "--no-confirmation", "-p", prompt) // #nosec G204 - command arguments are controlled
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("claude command failed: %w", err)
	}

	// Parse JSON from Claude's response
	var changes CodeChanges
	if err := json.Unmarshal(output, &changes); err != nil {
		return nil, fmt.Errorf("failed to parse Claude response: %w", err)
	}

	return &changes, nil
}

// gitOperations handles git branch, commit, and push
func (p *Processor) gitOperations(issue *Issue) error {
	branchName := fmt.Sprintf("cca/issue-%d", issue.Number)

	// Create and checkout branch
	cmd := exec.Command("git", "checkout", "-b", branchName) // #nosec G204 - branchName is controlled
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Add all changes
	cmd = exec.Command("git", "add", ".")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add files: %w", err)
	}

	// Commit
	commitMsg := fmt.Sprintf("Implement: %s", issue.Title)
	cmd = exec.Command("git", "commit", "-m", commitMsg)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Push to origin
	cmd = exec.Command("git", "push", "origin", branchName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}

// createPR uses GitHub CLI to create a pull request
func (p *Processor) createPR(issue *Issue) (string, error) {
	title := fmt.Sprintf("Fix: %s", issue.Title)
	body := fmt.Sprintf("Resolves: %s", issue.URL)

	cmd := exec.Command("gh", "pr", "create", "--draft", "--title", title, "--body", body)
	
	// Capture output to get PR URL
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create PR: %w\nOutput: %s", err, out.String())
	}

	// Extract PR URL from output
	output := out.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) > 0 {
		lastLine := lines[len(lines)-1]
		if strings.Contains(lastLine, "github.com") {
			return lastLine, nil
		}
	}

	return output, nil
}