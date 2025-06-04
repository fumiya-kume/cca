#!/usr/bin/env deno run --allow-read --allow-write --allow-run --allow-env

/**
 * ccAgents - GitHub Issue to PR Automation Tool
 * 
 * A simple CLI tool that takes a GitHub issue URL and automatically:
 * 1. Fetches the issue via GitHub CLI
 * 2. Generates code implementation via Claude CLI
 * 3. Runs verification script with intelligent retry loop
 * 4. Creates git branch, commits, and pushes
 * 5. Creates a pull request via GitHub CLI
 * 
 * Usage: deno run --allow-read --allow-write --allow-run --allow-env cca.ts <github-issue-url>
 * Example: deno run --allow-read --allow-write --allow-run --allow-env cca.ts https://github.com/owner/repo/issues/123
 */

// ============================================================================
// Types
// ============================================================================

interface Issue {
  number: number;
  title: string;
  body: string;
  repository: string;
  url: string;
}

interface CodeChanges {
  files: Record<string, string>;          // path -> content
  new_files: string[];
  deleted_files: string[];
  summary: string;
}

interface ProcessorConfig {
  maxRetries: number;
  verifyScript: string;
}

interface VerificationResult {
  success: boolean;
  error: string;
}

interface ClaudeResponse {
  conversation_id?: string;
  messages?: Array<{
    type: string;
    content: string;
  }>;
  usage?: {
    input_tokens: number;
    output_tokens: number;
  };
}

// ============================================================================
// Processor Class
// ============================================================================

class Processor {
  private config: ProcessorConfig;

  constructor(config: ProcessorConfig = { maxRetries: 3, verifyScript: '.cca/verify.sh' }) {
    this.config = config;
  }

  async processIssue(issueURL: string): Promise<void> {
    try {
      // 1. Fetch issue via GitHub CLI
      console.log(`🔍 Fetching issue from ${issueURL}...`);
      const issue = await this.fetchIssue(issueURL);
      console.log(`✅ Issue fetched: "${issue.title}"`);

      // 2. Generate code via Claude CLI
      console.log('🤖 Generating code with Claude...');
      let codeChanges = await this.generateCode(issue);
      console.log(`✅ Code generated: ${Object.keys(codeChanges.files).length} files changed`);

      // 3. Verification loop with retries
      const success = await this.verificationLoop(codeChanges, issue);
      if (!success) {
        throw new Error(`Verification failed after ${this.config.maxRetries} attempts`);
      }

      // 4. Git operations
      await this.performGitOperations(issue, codeChanges);

      // 5. Create PR
      console.log('🎯 Creating pull request...');
      const prURL = await this.createPullRequest(issue);
      console.log(`✅ Pull request created: ${prURL}`);
      console.log('✅ Pull request created successfully!');

    } catch (error) {
      throw new Error(`Failed to process issue: ${error.message}`);
    }
  }

  private async fetchIssue(issueURL: string): Promise<Issue> {
    const command = new Deno.Command('gh', {
      args: ['issue', 'view', issueURL, '--json', 'number,title,body,url'],
      stdout: 'piped',
      stderr: 'piped',
    });

    const { success, stdout, stderr } = await command.output();
    
    if (!success) {
      const errorText = new TextDecoder().decode(stderr);
      throw new Error(`Failed to fetch issue: ${errorText}`);
    }

    const output = new TextDecoder().decode(stdout);
    const issue = JSON.parse(output) as Issue;
    
    // Extract repository from URL
    const urlMatch = issueURL.match(/github\.com\/([^\/]+\/[^\/]+)/);
    if (urlMatch) {
      issue.repository = urlMatch[1];
    }

    return issue;
  }

  private async generateCode(issue: Issue): Promise<CodeChanges> {
    const prompt = this.createGenerationPrompt(issue);
    
    const command = new Deno.Command('claude', {
      args: ['-p', prompt, '--output-format', 'json'],
      stdout: 'piped',
      stderr: 'piped',
    });

    const { success, stdout, stderr } = await command.output();
    
    if (!success) {
      const errorText = new TextDecoder().decode(stderr);
      throw new Error(`Failed to generate code: ${errorText}`);
    }

    const output = new TextDecoder().decode(stdout);
    
    try {
      // Parse Claude's JSON response
      const claudeResponse = JSON.parse(output) as ClaudeResponse;
      
      // Extract the assistant's message content
      let assistantContent = '';
      if (claudeResponse.messages) {
        for (const message of claudeResponse.messages) {
          if (message.type === 'assistant' || message.type === 'text') {
            assistantContent += message.content;
          }
        }
      } else {
        // Fallback: try to find JSON in raw output
        assistantContent = output;
      }

      // Extract JSON from the assistant's response
      const jsonMatch = assistantContent.match(/\{[\s\S]*\}/);
      if (!jsonMatch) {
        throw new Error('No JSON found in Claude response');
      }
      
      return JSON.parse(jsonMatch[0]) as CodeChanges;
    } catch (error) {
      throw new Error(`Failed to parse Claude response: ${error.message}\n\nRaw output: ${output.substring(0, 500)}...`);
    }
  }

  private async verificationLoop(codeChanges: CodeChanges, issue: Issue): Promise<boolean> {
    let currentChanges = codeChanges;
    
    for (let attempt = 1; attempt <= this.config.maxRetries; attempt++) {
      // Apply changes to disk
      await this.applyChanges(currentChanges);
      
      // Ensure verification script exists
      await this.ensureVerificationScript();
      
      // Run verification
      console.log(`🔧 Running verification (${this.config.verifyScript})...`);
      const verificationResult = await this.runVerification();
      
      if (verificationResult.success) {
        console.log('✅ Verification passed');
        return true;
      }
      
      if (attempt < this.config.maxRetries) {
        console.log(`❌ Verification failed: "${verificationResult.error}"`);
        console.log(`🔄 Verification failed (attempt ${attempt}/${this.config.maxRetries}), asking Claude to fix...`);
        
        // Ask Claude to fix
        console.log('🤖 Claude fixing verification errors...');
        currentChanges = await this.fixCode(currentChanges, verificationResult.error);
        console.log(`✅ Code updated: ${Object.keys(currentChanges.files).length} files changed`);
      } else {
        console.log(`❌ Verification failed: "${verificationResult.error}"`);
        console.log(`❌ Max retries (${this.config.maxRetries}) exceeded`);
      }
    }
    
    return false;
  }

  private async applyChanges(changes: CodeChanges): Promise<void> {
    // Delete files first
    for (const filePath of changes.deleted_files) {
      try {
        await Deno.remove(filePath);
        console.log(`🗑️  Deleted: ${filePath}`);
      } catch (error) {
        // File might not exist, continue
        console.warn(`Warning: Could not delete ${filePath}: ${error.message}`);
      }
    }

    // Create/update files
    for (const [filePath, content] of Object.entries(changes.files)) {
      // Ensure directory exists
      const dir = filePath.substring(0, filePath.lastIndexOf('/'));
      if (dir) {
        await Deno.mkdir(dir, { recursive: true }).catch(() => {});
      }
      
      await Deno.writeTextFile(filePath, content);
      
      // Check if this is a new file
      if (changes.new_files.includes(filePath)) {
        console.log(`📄 Created: ${filePath}`);
      } else {
        console.log(`📝 Updated: ${filePath}`);
      }
    }
  }

  private async ensureVerificationScript(): Promise<void> {
    try {
      await Deno.stat(this.config.verifyScript);
    } catch {
      // Script doesn't exist, create template
      const template = `#!/bin/bash
# Add your build, test, and lint commands here
# Examples:
# go build ./...
# go test ./...
# golangci-lint run
# npm test
# deno test

echo "No verification script configured - skipping checks"
exit 0
`;
      
      // Ensure directory exists
      const dir = this.config.verifyScript.substring(0, this.config.verifyScript.lastIndexOf('/'));
      if (dir) {
        await Deno.mkdir(dir, { recursive: true }).catch(() => {});
      }
      
      await Deno.writeTextFile(this.config.verifyScript, template);
      await Deno.chmod(this.config.verifyScript, 0o755);
      console.log(`📋 Created verification script template: ${this.config.verifyScript}`);
    }
  }

  private async runVerification(): Promise<VerificationResult> {
    const command = new Deno.Command('bash', {
      args: [this.config.verifyScript],
      stdout: 'piped',
      stderr: 'piped',
    });

    const { success, stdout, stderr } = await command.output();
    
    if (success) {
      return { success: true, error: '' };
    } else {
      const errorText = new TextDecoder().decode(stderr);
      const outputText = new TextDecoder().decode(stdout);
      return { 
        success: false, 
        error: errorText || outputText || 'Verification script failed'
      };
    }
  }

  private async fixCode(currentChanges: CodeChanges, verificationErrors: string): Promise<CodeChanges> {
    const prompt = this.createFixPrompt(currentChanges, verificationErrors);
    
    const command = new Deno.Command('claude', {
      args: ['-p', prompt, '--output-format', 'json'],
      stdout: 'piped',
      stderr: 'piped',
    });

    const { success, stdout, stderr } = await command.output();
    
    if (!success) {
      const errorText = new TextDecoder().decode(stderr);
      throw new Error(`Failed to fix code: ${errorText}`);
    }

    const output = new TextDecoder().decode(stdout);
    
    try {
      // Parse Claude's JSON response
      const claudeResponse = JSON.parse(output) as ClaudeResponse;
      
      // Extract the assistant's message content
      let assistantContent = '';
      if (claudeResponse.messages) {
        for (const message of claudeResponse.messages) {
          if (message.type === 'assistant' || message.type === 'text') {
            assistantContent += message.content;
          }
        }
      } else {
        // Fallback: try to find JSON in raw output
        assistantContent = output;
      }

      // Extract JSON from the assistant's response
      const jsonMatch = assistantContent.match(/\{[\s\S]*\}/);
      if (!jsonMatch) {
        throw new Error('No JSON found in Claude fix response');
      }
      
      return JSON.parse(jsonMatch[0]) as CodeChanges;
    } catch (error) {
      throw new Error(`Failed to parse Claude fix response: ${error.message}\n\nRaw output: ${output.substring(0, 500)}...`);
    }
  }

  private async performGitOperations(issue: Issue, changes: CodeChanges): Promise<void> {
    const branchName = `cca/issue-${issue.number}`;
    
    // Create branch
    console.log(`📝 Creating branch ${branchName}...`);
    await this.runGitCommand(['checkout', '-b', branchName]);
    
    // Add changes
    await this.runGitCommand(['add', '.']);
    
    // Commit
    const commitMessage = `Implement: ${issue.title}

${changes.summary}

Resolves: ${issue.url}`;
    await this.runGitCommand(['commit', '-m', commitMessage]);
    
    // Push
    await this.runGitCommand(['push', 'origin', branchName]);
    
    console.log('✅ Changes committed and pushed');
  }

  private async runGitCommand(args: string[]): Promise<void> {
    const command = new Deno.Command('git', {
      args,
      stdout: 'piped',
      stderr: 'piped',
    });

    const { success, stderr } = await command.output();
    
    if (!success) {
      const errorText = new TextDecoder().decode(stderr);
      throw new Error(`Git command failed: ${errorText}`);
    }
  }

  private async createPullRequest(issue: Issue): Promise<string> {
    const title = `Fix: ${issue.title}`;
    const body = `Resolves: ${issue.url}

## Changes Made
This PR automatically implements a solution for the GitHub issue.

## Verification
The implementation has been verified using the project's verification script.

---
*Generated by ccAgents - GitHub Issue to PR Automation Tool*`;
    
    const command = new Deno.Command('gh', {
      args: ['pr', 'create', '--draft', '--title', title, '--body', body],
      stdout: 'piped',
      stderr: 'piped',
    });

    const { success, stdout, stderr } = await command.output();
    
    if (!success) {
      const errorText = new TextDecoder().decode(stderr);
      throw new Error(`Failed to create PR: ${errorText}`);
    }

    const output = new TextDecoder().decode(stdout).trim();
    return output;
  }

  private createGenerationPrompt(issue: Issue): string {
    return `Implement a solution for this GitHub issue:

Issue: ${issue.title}
Description: ${issue.body}
Repository: ${issue.repository}

Analyze the issue and provide a complete implementation including:
1. All necessary code changes
2. Tests for the implementation  
3. Any documentation updates needed

Return the implementation as file paths and their complete content.

IMPORTANT: You must return your response as valid JSON in exactly this format:
{
  "files": {
    "path/to/file.go": "complete file content...",
    "path/to/test.go": "test file content..."
  },
  "new_files": ["list", "of", "new", "files"],
  "deleted_files": ["list", "of", "deleted", "files"],
  "summary": "Brief description of changes made"
}

Do not include any explanation or markdown - only return the JSON object.`;
  }

  private createFixPrompt(currentChanges: CodeChanges, verificationErrors: string): string {
    return `The verification script failed with these errors:

${verificationErrors}

Here are the current code changes:
${JSON.stringify(currentChanges, null, 2)}

Please fix the code to resolve these verification errors. Return the corrected implementation.

IMPORTANT: You must return your response as valid JSON in exactly this format:
{
  "files": {...},
  "new_files": [...],
  "deleted_files": [...],
  "summary": "Description of fixes applied"
}

Do not include any explanation or markdown - only return the JSON object.`;
  }
}

// ============================================================================
// CLI Functions
// ============================================================================

function validateGitHubIssueURL(url: string): boolean {
  try {
    const urlObj = new URL(url);
    return urlObj.hostname === 'github.com' && 
           urlObj.pathname.includes('/issues/') &&
           /\/issues\/\d+/.test(urlObj.pathname);
  } catch {
    return false;
  }
}

function printUsage(): void {
  console.log('ccAgents - GitHub Issue to PR Automation Tool');
  console.log('');
  console.log('Usage: deno run --allow-read --allow-write --allow-run --allow-env cca.ts <github-issue-url>');
  console.log('');
  console.log('Example:');
  console.log('  deno run --allow-read --allow-write --allow-run --allow-env cca.ts https://github.com/owner/repo/issues/123');
  console.log('');
  console.log('Alternatively, compile to executable:');
  console.log('  deno compile --allow-read --allow-write --allow-run --allow-env --output cca cca.ts');
  console.log('  ./cca https://github.com/owner/repo/issues/123');
  console.log('');
  console.log('The URL must be a valid GitHub issue URL.');
  console.log('');
  console.log('Required tools: gh (GitHub CLI), claude (Claude Code), git, bash');
  console.log('');
  console.log('Authentication setup:');
  console.log('  gh auth login');
  console.log('  claude login');
}

async function checkRequiredTools(): Promise<void> {
  const requiredTools = [
    { name: 'gh', description: 'GitHub CLI', checkArgs: ['--version'], installUrl: 'https://cli.github.com/' },
    { name: 'claude', description: 'Claude Code', checkArgs: ['--version'], installUrl: 'https://claude.ai/code' },
    { name: 'git', description: 'Git', checkArgs: ['--version'], installUrl: 'https://git-scm.com/downloads' },
    { name: 'bash', description: 'Bash shell', checkArgs: ['--version'], installUrl: 'Usually pre-installed' }
  ];
  
  const missingTools: Array<{name: string, description: string, installUrl: string}> = [];
  
  for (const tool of requiredTools) {
    try {
      const command = new Deno.Command(tool.name, {
        args: tool.checkArgs,
        stdout: 'null',
        stderr: 'null',
      });
      
      const { success } = await command.output();
      
      if (!success) {
        missingTools.push(tool);
      }
    } catch {
      missingTools.push(tool);
    }
  }
  
  if (missingTools.length > 0) {
    console.error('Missing required tools:');
    for (const tool of missingTools) {
      console.error(`  ❌ ${tool.name} (${tool.description}) - Install from: ${tool.installUrl}`);
    }
    console.error('');
    console.error('After installation, make sure to authenticate:');
    console.error('  gh auth login');
    console.error('  claude login');
    
    throw new Error(`Missing ${missingTools.length} required tool(s)`);
  }
}

// ============================================================================
// Main Function
// ============================================================================

async function main(): Promise<void> {
  const args = Deno.args;
  
  // Show help if no arguments or help flag
  if (args.length === 0 || args.includes('--help') || args.includes('-h')) {
    printUsage();
    Deno.exit(0);
  }
  
  // Validate arguments
  if (args.length !== 1) {
    console.error('Error: Exactly one argument (GitHub issue URL) is required.');
    printUsage();
    Deno.exit(1);
  }
  
  const issueURL = args[0];
  
  // Validate URL format
  if (!validateGitHubIssueURL(issueURL)) {
    console.error('Error: Invalid GitHub issue URL format.');
    console.error('URL must contain "github.com" and "/issues/" with a valid issue number.');
    console.error('');
    console.error('Valid format: https://github.com/owner/repo/issues/123');
    Deno.exit(1);
  }
  
  // Check if required external tools are available
  try {
    console.log('🔧 Checking required tools...');
    await checkRequiredTools();
    console.log('✅ All required tools are available');
  } catch (error) {
    console.error(`Error: ${error.message}`);
    Deno.exit(1);
  }
  
  // Process the issue
  try {
    const processor = new Processor();
    await processor.processIssue(issueURL);
  } catch (error) {
    console.error(`❌ Error: ${error.message}`);
    Deno.exit(1);
  }
}

// ============================================================================
// Entry Point
// ============================================================================

// Handle uncaught errors gracefully
globalThis.addEventListener('unhandledrejection', (event) => {
  console.error(`❌ Unhandled error: ${event.reason?.message || event.reason}`);
  Deno.exit(1);
});

globalThis.addEventListener('error', (event) => {
  console.error(`❌ Unhandled error: ${event.error?.message || event.error}`);
  Deno.exit(1);
});

// Run main function if this file is executed directly
if (import.meta.main) {
  await main();
}