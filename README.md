# Claude Code Assistant (CCA)

CCA is a shell-based tool that automates the process of implementing GitHub issue fixes. It leverages Claude AI to generate code changes, applies them locally, runs verification tests, and creates pull requests automatically.

## Features

- 🤖 **AI-Powered Code Generation**: Uses the `claude-code-js` library to analyze GitHub issues and generate implementation code
- 🔄 **Automatic Retry Logic**: If verification fails, Claude will attempt to fix the errors (up to 3 attempts)
- 🧪 **Built-in Verification**: Runs custom verification scripts to ensure code quality before committing
- 🌿 **Automated Git Workflow**: Uses a temporary git worktree to create branches, commit changes, and open draft pull requests

## Requirements

- [`gh`](https://cli.github.com/) GitHub CLI authenticated and configured for your repository
- `git` with push access to the target repository
- `bash` for running verification scripts
- `node` to run Claude helpers
- `jq` for JSON parsing

## Installation

Clone this repository:

```bash
git clone https://github.com/fumiya-kume/cca.git
cd cca
npm install
```

## Usage

Run CCA with a GitHub issue URL using the shell script:

```bash
export ANTHROPIC_API_KEY=your-key
./cca.sh https://github.com/owner/repo/issues/123
```


1. **Fetches Issue Details**: Uses `gh issue view` to retrieve the issue information
2. **Generates Code**: Uses the `claude-code-js` library to produce a solution based on the issue details
3. **Applies Changes**: Writes the generated files to your local repository
4. **Runs Verification**: Executes `.cca/verify.sh` to validate the changes
5. **Handles Failures**: If verification fails, asks Claude to fix the errors
6. **Creates Worktree**: Checks out a new worktree in `.cca/worktrees/` for branch `cca/issue-<number>` and commits the changes there
7. **Opens Pull Request**: Creates a draft PR that links back to the original issue and then removes the temporary worktree

The worktree is created under `.cca/worktrees/` using the issue number and a random suffix. It is automatically cleaned up after the pull request is opened.

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

echo "No verification script configured - skipping checks"
exit 0
```

## Error Handling

CCA provides clear error messages for common issues:

- Invalid GitHub issue URLs
- GitHub CLI authentication failures
- Verification script failures
- Git operation errors
- Claude Code errors

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
