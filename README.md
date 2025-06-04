# Claude Code Assistant (CCA)

CCA is a command line tool written in TypeScript for Deno. It fetches a GitHub
issue, asks Claude to generate the required code changes, runs a verification
script, and then commits and pushes a new branch with a draft pull request.

## Requirements

- [Deno](https://deno.land/) v1.35+ (the install script or package manager can
  be used)
- [`gh`](https://cli.github.com/) GitHub CLI configured for the target
  repository
- `git` with push access

## Usage

Run the tool with a GitHub issue URL:

```bash
deno run -A src/main.ts https://github.com/owner/repo/issues/123
```

CCA will:

1. Fetch issue details using `gh issue view`
2. Ask Claude Code JS to generate a patch
3. Apply the changes locally
4. Execute `.cca/verify.sh` if present (otherwise a stub is created)
5. Commit and push the changes to a branch named `cca/issue-<number>`
6. Create a draft pull request linking back to the issue

## Verification Script

Place your build/test commands in `.cca/verify.sh`. The script should exit with
a non-zero status on failure. If the file is missing, CCA writes a stub that
simply succeeds.

## Development

Format the code using:

```bash
deno fmt src/main.ts src/processor.ts src/types.ts
```

Lint with:

```bash
deno lint src/main.ts src/processor.ts src/types.ts
```

## Continuous Integration

The repository includes a GitHub Actions workflow that runs formatting, linting,
and the test suite for every pull request.
