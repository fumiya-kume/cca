# Claude Code Assistant (CCA)

CCA is a command-line tool written in TypeScript for Deno that automates the process of implementing GitHub issue fixes. It leverages Claude AI to generate code changes, applies them locally, runs verification tests, and creates pull requests automatically.

## Features

- 🤖 **AI-Powered Code Generation**: Uses Claude Code JS SDK to analyze GitHub issues and generate implementation code
- 🔄 **Automatic Retry Logic**: If verification fails, Claude will attempt to fix the errors (up to 3 attempts)
- 🧪 **Built-in Verification**: Runs custom verification scripts to ensure code quality before committing
- 🌿 **Automated Git Workflow**: Creates branches, commits changes, and opens draft pull requests
- 📝 **Full Code Coverage**: Includes comprehensive test suite with mocked dependencies

## Requirements

- [Deno](https://deno.land/) v1.35+ (install via [installation guide](https://deno.land/#installation))
- [`gh`](https://cli.github.com/) GitHub CLI authenticated and configured for your repository
- `git` with push access to the target repository
- `bash` for running verification scripts

## Installation

Clone this repository:

```bash
git clone https://github.com/fumiya-kume/cca.git
cd cca
```

## Usage

Run CCA with a GitHub issue URL:

```bash
deno run -A src/main.ts https://github.com/owner/repo/issues/123
```

### What CCA Does

1. **Fetches Issue Details**: Uses `gh issue view` to retrieve the issue information
2. **Generates Code**: Sends the issue details to Claude Code JS to generate a solution
3. **Applies Changes**: Writes the generated files to your local repository
4. **Runs Verification**: Executes `.cca/verify.sh` to validate the changes
5. **Handles Failures**: If verification fails, asks Claude to fix the errors
6. **Creates Branch**: Commits changes to `cca/issue-<number>` branch
7. **Opens Pull Request**: Creates a draft PR that links back to the original issue

## Verification Script

CCA looks for a verification script at `.cca/verify.sh`. This script should:

- Run your build process
- Execute tests
- Perform linting
- Exit with status 0 on success, non-zero on failure

If the script doesn't exist, CCA creates a stub that always passes:

```bash
#!/bin/bash
# Add your build, test, and lint commands here
# Examples:
# deno task build
# deno test

echo "No verification script configured - skipping checks"
exit 0
```

### Example Verification Script

```bash
#!/bin/bash
set -e

# Run formatter check
echo "Checking code formatting..."
deno fmt --check src/

# Run linter
echo "Running linter..."
deno lint src/

# Run tests
echo "Running tests..."
deno test --allow-all

# Build project (if applicable)
# echo "Building project..."
# deno task build

echo "All checks passed!"
```

## Development

### Project Structure

```
cca/
├── src/
│   ├── main.ts        # Entry point and CLI argument handling
│   ├── processor.ts   # Core logic for processing issues
│   ├── git.ts         # Git operations (branch, commit, push, PR)
│   └── types.ts       # TypeScript interfaces
├── tests/
│   └── processor.test.ts # Comprehensive test suite
└── .github/
    └── workflows/
        └── ci.yml     # GitHub Actions CI pipeline
```

### Running Locally

Format code:
```bash
deno fmt src/main.ts src/processor.ts src/types.ts src/git.ts
```

Lint code:
```bash
deno lint src/main.ts src/processor.ts src/types.ts src/git.ts
```

Run tests:
```bash
DENO_TLS_CA_STORE=system deno test --allow-all --coverage=cov
```

Generate coverage report:
```bash
deno coverage cov --lcov > coverage.lcov
```

### Key Components

#### Processor Class (`src/processor.ts`)
- Main orchestrator for the issue processing workflow
- Handles Claude API communication
- Manages file operations and verification
- Implements retry logic for failed verifications

#### Git Helpers (`src/git.ts`)
- Provides git operations: branch creation, committing, pushing
- Creates pull requests using GitHub CLI
- Mockable helpers for testing

#### Type Definitions (`src/types.ts`)
- `Issue`: GitHub issue structure
- `CodeChanges`: Claude's response format for file modifications

### Testing

The test suite uses Deno's built-in testing framework with mocked external dependencies:

- Mock `Deno.Command` for shell command simulation
- Mock file system operations
- Test coverage for error handling and edge cases

## Continuous Integration

The repository includes GitHub Actions workflow that runs on every pull request:

1. Code formatting check
2. Linting
3. Full test suite with coverage
4. Coverage report generation

## Error Handling

CCA provides clear error messages for common issues:

- Invalid GitHub issue URLs
- GitHub CLI authentication failures
- Verification script failures
- Git operation errors
- Claude API errors

## Security Considerations

- Never commit sensitive data or credentials
- Review generated code before merging pull requests
- Use the verification script to enforce security policies
- The tool requires full file system access (`-A` flag)

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Ensure tests pass and coverage is maintained
5. Submit a pull request

## License

See the LICENSE file for details.

## Troubleshooting

### Common Issues

**"gh command failed"**
- Ensure `gh` is installed and authenticated: `gh auth login`

**"failed to create branch"**
- Check if the branch already exists
- Ensure you have write access to the repository

**"Invalid GitHub issue URL"**
- URL must contain 'github.com' and '/issues/'
- Format: `https://github.com/owner/repo/issues/number`

**Verification keeps failing**
- Check `.cca/verify.sh` for proper error reporting
- Ensure the script exits with proper status codes
- Review Claude's generated code for syntax errors

### Debug Mode

For verbose output during processing, check the console output which includes:
- Issue fetching status
- Claude prompt and response indicators
- File operation logs
- Git command execution
- Verification script output