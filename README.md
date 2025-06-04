# ccAgents (CCA) - GitHub Issue to PR Automation Tool

ccAgents (formerly Claude Code Assistant) is a CLI tool that automates the process of turning GitHub issues into pull requests. It fetches a GitHub issue, uses Claude AI to generate the required code changes, runs verification scripts, and creates a pull request with the implementation.

## Features

- 🤖 Automated code generation using Claude AI
- 🔄 Intelligent retry loop that fixes verification errors
- 🧪 Automatic verification script execution
- 🌿 Git branch creation and management
- 🔧 Pull request creation via GitHub CLI
- 📝 Multiple implementation options (TypeScript/Deno and Go binary)

## Requirements

- [Deno](https://deno.land/) v1.35+ (for TypeScript version)
- [`claude`](https://claude.ai) CLI tool configured with API access
- [`gh`](https://cli.github.com/) GitHub CLI configured for the target repository
- `git` with push access to the repository
- `bash` for running verification scripts

## Installation

### Option 1: Run directly with Deno

No installation needed. Run directly from the source:

```bash
deno run --allow-read --allow-write --allow-run --allow-env cca.ts <github-issue-url>
```

### Option 2: Compile to executable

```bash
deno compile --allow-read --allow-write --allow-run --allow-env cca.ts
./cca <github-issue-url>
```

### Option 3: Use the Go binary

A pre-compiled Go binary is available in the repository:

```bash
./cca <github-issue-url>
```

## Usage

Run ccAgents with a GitHub issue URL:

```bash
# Using the TypeScript version
deno run --allow-read --allow-write --allow-run --allow-env cca.ts https://github.com/owner/repo/issues/123

# Or using the compiled executable
./cca https://github.com/owner/repo/issues/123
```

### What ccAgents does:

1. **Validates environment** - Checks that all required tools are available
2. **Fetches issue details** - Uses `gh issue view` to get issue information
3. **Generates implementation** - Asks Claude to create code changes based on the issue
4. **Applies changes** - Writes new files, updates existing files, and deletes files as needed
5. **Runs verification** - Executes `.cca/verify.sh` with intelligent retry on failure
6. **Creates branch** - Makes a new git branch named `cca/issue-<number>`
7. **Commits changes** - Creates a commit with the issue details
8. **Pushes to remote** - Pushes the branch to GitHub
9. **Creates pull request** - Opens a draft PR linking back to the original issue

## Verification Script

Place your build, test, and validation commands in `.cca/verify.sh`. The script should:
- Exit with status 0 on success
- Exit with non-zero status on failure
- Output clear error messages for debugging

Example `.cca/verify.sh`:

```bash
#!/bin/bash
set -e

# Run tests
npm test

# Type checking
npm run typecheck

# Linting
npm run lint
```

If no verification script exists, ccAgents creates a simple stub that always succeeds.

## Intelligent Retry Loop

When verification fails, ccAgents will:
1. Capture the error output
2. Send it back to Claude with the current implementation
3. Ask Claude to fix the issues
4. Apply the fixes and retry verification
5. Continue for up to 5 iterations (configurable)

## Development

### Project Structure

```
cca/
├── cca.ts           # Main TypeScript implementation (standalone)
├── cca              # Go binary implementation
├── src/             # Original modular TypeScript implementation
│   ├── main.ts      # Entry point
│   ├── processor.ts # Core logic
│   ├── git.ts       # Git operations
│   └── types.ts     # Type definitions
├── tests/           # Test files
├── cmd/ccagents/    # Go source code
└── go.mod           # Go module file
```

### Running Tests

```bash
DENO_TLS_CA_STORE=system deno test --allow-all --coverage=cov
```

### Code Formatting

```bash
# Format all TypeScript files
deno fmt

# Or format specific files
deno fmt cca.ts src/*.ts tests/*.ts
```

### Linting

```bash
# Lint all TypeScript files
deno lint

# Or lint specific files
deno lint cca.ts src/*.ts tests/*.ts
```

## Configuration

ccAgents uses the following environment variables (all optional):
- `CLAUDE_MODEL` - Claude model to use (defaults to system default)
- `GH_TOKEN` - GitHub token (usually set by gh CLI)

## Troubleshooting

### Common Issues

1. **"Error: Required tool not found"**
   - Ensure all required tools (claude, gh, git) are installed and in PATH

2. **"Error fetching issue"**
   - Verify you're authenticated with `gh auth status`
   - Check the issue URL is correct and accessible

3. **"Claude API error"**
   - Ensure the claude CLI is configured with valid credentials
   - Check your API quota hasn't been exceeded

4. **Verification keeps failing**
   - Review the `.cca/verify.sh` script for issues
   - Check the error output for specific problems
   - Consider simplifying verification requirements

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Ensure all tests pass
6. Submit a pull request

## License

[Add license information here]