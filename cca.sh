#!/usr/bin/env bash
set -euo pipefail

claude_chat() {
  local prompt="$1"
  if [ -z "${ANTHROPIC_API_KEY:-}" ]; then
    echo "ANTHROPIC_API_KEY not set" >&2
    return 1
  fi

  curl -sS https://api.anthropic.com/v1/messages \
    -H "x-api-key: $ANTHROPIC_API_KEY" \
    -H "anthropic-version: 2023-06-01" \
    -H "content-type: application/json" \
    -d "$(jq -n --arg p "$prompt" '{model:"claude-3-opus-20240229",max_tokens:4096,messages:[{role:"user",content:$p}] }')" |
    jq -r '.content[0].text'
}

apply_changes() {
  local file="$1"
  jq -r '.deleted_files[]?' "$file" | while read -r path; do
    rm -f "$path" && echo "Deleted $path"
  done

  jq -r '.files | keys[]' "$file" | while read -r path; do
    content=$(jq -r --arg p "$path" '.files[$p]' "$file")
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
if ! command -v curl >/dev/null; then
  echo "curl command not found" >&2
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

changes_json=$(claude_chat "$(cat "$prompt_file")")
rm "$prompt_file"

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
  changes_json=$(claude_chat "$(cat "$fix_prompt_file")")
  rm "$fix_prompt_file"
  attempt=$((attempt + 1))
done

rand=$(tr -dc 'a-z0-9' </dev/urandom | head -c 6)
branch="cca/issue-$number-$rand"

git checkout -b "$branch"
git add .
git commit -m "Implement: $title"
git push origin "$branch"

pr_url=$(gh pr create --draft --title "Fix: $title" --body "Resolves: $ISSUE_URL")

echo "Pull request created: $pr_url"
