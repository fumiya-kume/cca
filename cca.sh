#!/usr/bin/env bash
set -euo pipefail

claude_chat() {
  local prompt_file="$1"
  local mode="${2:-with-p}"
  local prompt
  prompt=$(cat "$prompt_file")
  if [ "$mode" = "with-p" ]; then
    claude -p "$prompt"
  else
    claude "$prompt"
  fi
}

apply_changes() {
  local file="$1"
  jq -r '.deleted_files[]?' "$file" | while read -r path; do
    rm -f "$path"
    echo "Deleted $path"
  done

  jq -r '.files | to_entries[] | [.key, (.value|@base64)] | @tsv' "$file" 2>/dev/null | \
  while IFS=$'\t' read -r path b64; do
    content=$(echo "$b64" | base64 --decode)
    mkdir -p "$(dirname "$path")"
    printf '%s' "$content" > "$path"
    echo "Wrote $path"
  done
}

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <github-issue-url>" >&2
  exit 1
fi

ISSUE_URL="$1"
if [[ "$ISSUE_URL" != *github.com* || "$ISSUE_URL" != */issues/* ]]; then
  echo "Invalid GitHub issue URL: $ISSUE_URL" >&2
  exit 1
fi

if ! command -v gh >/dev/null; then
  echo "gh command not found" >&2
  exit 1
fi
if ! command -v jq >/dev/null; then
  echo "jq command not found" >&2
  exit 1
fi
if ! command -v claude >/dev/null; then
  echo "claude command not found" >&2
  exit 1
fi

if [ -z "${ANTHROPIC_API_KEY:-}" ]; then
  echo "ANTHROPIC_API_KEY environment variable not set" >&2
  exit 1
fi

# fetch issue details
echo "Fetching issue..."
issue_json=$(gh issue view "$ISSUE_URL" --json number,title,body,url)
number=$(echo "$issue_json" | jq -r '.number')
title=$(echo "$issue_json" | jq -r '.title')
body=$(echo "$issue_json" | jq -r '.body')
repo=$(echo "$ISSUE_URL" | awk -F/ '{print $4"/"$5}')

prompt_file=$(mktemp)
cat >"$prompt_file" <<EOF2
Implement a solution for this GitHub issue:

Issue: $title
Description: $body
Repository: $repo

Analyze the issue and provide a complete implementation including:
1. All necessary code changes
2. Tests for the implementation
3. Any documentation updates needed

Return the implementation as file paths and their complete content.

Format as JSON:
{
  "files": {"path/to/file.ts": "complete file content..."},
  "new_files": ["list", "of", "new", "files"],
  "deleted_files": ["list", "of", "deleted", "files"],
  "summary": "Brief description of changes made"
}
EOF2


changes_json=$(claude_chat "$prompt_file" "no-p")
rm "$prompt_file"

rand=$(tr -dc 'a-z0-9' </dev/urandom | head -c 6)
branch="cca/issue-$number-$rand"
root_dir=$(git rev-parse --show-toplevel)
work_dir="$root_dir/.cca/worktrees/$branch"
mkdir -p "$root_dir/.cca/worktrees"
git worktree add "$work_dir" -b "$branch"
pushd "$work_dir" >/dev/null

max_retries=3
attempt=1

while true; do
  tmp_changes=$(mktemp)
  echo "$changes_json" > "$tmp_changes"

  apply_changes "$tmp_changes"
  rm "$tmp_changes"

  echo "Running verification..."
  verify_output=$(bash .cca/verify.sh 2>&1)
  verify_code=$?

  if [ $verify_code -eq 0 ]; then
    echo "Verification passed"
    break
  fi

  if [ $attempt -ge $max_retries ]; then
    echo "Verification failed after $max_retries attempts" >&2
    echo "$verify_output" >&2
    exit 1
  fi

  echo "Verification failed: $verify_output"

  fix_prompt_file=$(mktemp)
  cat >"$fix_prompt_file" <<EOF3
The verification script failed with these errors:

$verify_output

Here are the current code changes:
$changes_json

Please fix the code to resolve these verification errors. Return the corrected implementation.

Format as JSON with the same structure as before:
{
  "files": {"path": "content"},
  "new_files": [],
  "deleted_files": [],
  "summary": "..."
}
EOF3
  changes_json=$(claude_chat "$fix_prompt_file" "with-p")
  rm "$fix_prompt_file"
  attempt=$((attempt + 1))
done


git add .
git commit -m "Implement: $title"
git push origin "$branch"

pr_url=$(gh pr create --draft --title "Fix: $title" --body "Resolves: $ISSUE_URL")

popd >/dev/null
git worktree remove "$work_dir"

echo "Pull request created: $pr_url"
