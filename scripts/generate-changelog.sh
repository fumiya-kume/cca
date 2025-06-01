#!/bin/bash
# Changelog Generation Script
# Automatically generates changelog entries from git commits

set -e

# Configuration
REPO="fumiya-kume/cca"
CHANGELOG_FILE="CHANGELOG.md"
TEMP_CHANGELOG="/tmp/changelog-temp.md"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging functions
info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Show help
show_help() {
    echo "Changelog Generation Script"
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --version TAG        Generate changelog for specific version"
    echo "  --from TAG           Start from specific tag"
    echo "  --to TAG             End at specific tag (default: HEAD)"
    echo "  --output FILE        Output file (default: CHANGELOG.md)"
    echo "  --format FORMAT      Output format (markdown, json, text)"
    echo "  --help, -h           Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Generate unreleased changes"
    echo "  $0 --version v1.0.0                  # Generate changelog for v1.0.0"
    echo "  $0 --from v0.9.0 --to v1.0.0         # Generate changes between versions"
    echo "  $0 --format json --output changes.json  # Output as JSON"
}

# Parse command line arguments
parse_args() {
    VERSION=""
    FROM_TAG=""
    TO_TAG="HEAD"
    OUTPUT_FILE="$CHANGELOG_FILE"
    FORMAT="markdown"
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --version)
                VERSION="$2"
                shift 2
                ;;
            --from)
                FROM_TAG="$2"
                shift 2
                ;;
            --to)
                TO_TAG="$2"
                shift 2
                ;;
            --output)
                OUTPUT_FILE="$2"
                shift 2
                ;;
            --format)
                FORMAT="$2"
                shift 2
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                ;;
        esac
    done
}

# Get git information
get_git_info() {
    # Check if we're in a git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        error "Not in a git repository"
    fi
    
    # Get tags
    if [ -z "$FROM_TAG" ]; then
        # Get the latest tag
        FROM_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
        if [ -z "$FROM_TAG" ]; then
            # If no tags, use initial commit
            FROM_TAG=$(git rev-list --max-parents=0 HEAD)
            info "No tags found, using initial commit: $FROM_TAG"
        else
            info "Using latest tag as start: $FROM_TAG"
        fi
    fi
    
    info "Generating changelog from $FROM_TAG to $TO_TAG"
}

# Categorize commit by type
categorize_commit() {
    local message="$1"
    
    case "$message" in
        feat:*|feat\(*\):*)
            echo "Added"
            ;;
        fix:*|fix\(*\):*)
            echo "Fixed"
            ;;
        docs:*|docs\(*\):*)
            echo "Documentation"
            ;;
        style:*|style\(*\):*)
            echo "Style"
            ;;
        refactor:*|refactor\(*\):*)
            echo "Changed"
            ;;
        perf:*|perf\(*\):*)
            echo "Performance"
            ;;
        test:*|test\(*\):*)
            echo "Testing"
            ;;
        chore:*|chore\(*\):*)
            echo "Maintenance"
            ;;
        ci:*|ci\(*\):*)
            echo "CI/CD"
            ;;
        build:*|build\(*\):*)
            echo "Build"
            ;;
        revert:*|revert\(*\):*)
            echo "Reverted"
            ;;
        security:*|security\(*\):*)
            echo "Security"
            ;;
        *)
            echo "Other"
            ;;
    esac
}

# Clean commit message
clean_commit_message() {
    local message="$1"
    
    # Remove conventional commit prefix
    message=$(echo "$message" | sed -E 's/^(feat|fix|docs|style|refactor|perf|test|chore|ci|build|revert|security)(\([^)]+\))?: //')
    
    # Capitalize first letter
    message="$(echo "${message:0:1}" | tr '[:lower:]' '[:upper:]')${message:1}"
    
    echo "$message"
}

# Generate changelog in markdown format
generate_markdown_changelog() {
    local temp_file="/tmp/commits-temp.txt"
    
    # Get commits between tags
    if [ "$FROM_TAG" = "$TO_TAG" ]; then
        info "Generating changelog for single commit/tag"
        git log --pretty=format:"%H|%s|%an|%ad" --date=short -1 "$TO_TAG" > "$temp_file"
    else
        git log --pretty=format:"%H|%s|%an|%ad" --date=short "$FROM_TAG..$TO_TAG" > "$temp_file"
    fi
    
    # Check if there are any commits
    if [ ! -s "$temp_file" ]; then
        warning "No commits found between $FROM_TAG and $TO_TAG"
        rm -f "$temp_file"
        return 1
    fi
    
    # Count commits by category
    declare -A categories
    declare -A category_commits
    
    while IFS='|' read -r hash subject author date; do
        local category=$(categorize_commit "$subject")
        local clean_message=$(clean_commit_message "$subject")
        
        if [ -z "${categories[$category]}" ]; then
            categories[$category]=1
            category_commits[$category]="- $clean_message ($hash)"
        else
            categories[$category]=$((categories[$category] + 1))
            category_commits[$category]="${category_commits[$category]}
- $clean_message ($hash)"
        fi
    done < "$temp_file"
    
    # Generate changelog
    local changelog_content=""
    
    if [ -n "$VERSION" ]; then
        changelog_content+="## [$VERSION] - $(date +%Y-%m-%d)\n\n"
    else
        changelog_content+="## [Unreleased]\n\n"
    fi
    
    # Sort categories by importance
    local ordered_categories=("Added" "Changed" "Fixed" "Security" "Performance" "Documentation" "Testing" "Build" "CI/CD" "Maintenance" "Style" "Reverted" "Other")
    
    for category in "${ordered_categories[@]}"; do
        if [ -n "${categories[$category]}" ]; then
            changelog_content+="### $category\n\n"
            changelog_content+="${category_commits[$category]}\n\n"
        fi
    done
    
    # Add statistics
    local total_commits=$(wc -l < "$temp_file")
    changelog_content+="**Total Changes:** $total_commits commits\n"
    changelog_content+="**Contributors:** $(cut -d'|' -f3 "$temp_file" | sort -u | wc -l)\n\n"
    
    echo -e "$changelog_content"
    
    rm -f "$temp_file"
}

# Generate changelog in JSON format
generate_json_changelog() {
    local temp_file="/tmp/commits-temp.txt"
    
    # Get commits
    if [ "$FROM_TAG" = "$TO_TAG" ]; then
        git log --pretty=format:"%H|%s|%an|%ae|%ad|%ad" --date=short --date=iso -1 "$TO_TAG" > "$temp_file"
    else
        git log --pretty=format:"%H|%s|%an|%ae|%ad|%ad" --date=short --date=iso "$FROM_TAG..$TO_TAG" > "$temp_file"
    fi
    
    # Generate JSON
    echo "{"
    echo "  \"version\": \"${VERSION:-unreleased}\","
    echo "  \"date\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\","
    echo "  \"range\": \"$FROM_TAG..$TO_TAG\","
    echo "  \"commits\": ["
    
    local first=true
    while IFS='|' read -r hash subject author email date iso_date; do
        if [ "$first" = true ]; then
            first=false
        else
            echo ","
        fi
        
        local category=$(categorize_commit "$subject")
        local clean_message=$(clean_commit_message "$subject")
        
        echo -n "    {"
        echo -n "\"hash\": \"$hash\", "
        echo -n "\"subject\": \"$subject\", "
        echo -n "\"message\": \"$clean_message\", "
        echo -n "\"category\": \"$category\", "
        echo -n "\"author\": \"$author\", "
        echo -n "\"email\": \"$email\", "
        echo -n "\"date\": \"$date\", "
        echo -n "\"iso_date\": \"$iso_date\""
        echo -n "}"
    done < "$temp_file"
    
    echo ""
    echo "  ]"
    echo "}"
    
    rm -f "$temp_file"
}

# Generate changelog in text format
generate_text_changelog() {
    local temp_file="/tmp/commits-temp.txt"
    
    # Get commits
    if [ "$FROM_TAG" = "$TO_TAG" ]; then
        git log --pretty=format:"%s (%h)" -1 "$TO_TAG" > "$temp_file"
    else
        git log --pretty=format:"%s (%h)" "$FROM_TAG..$TO_TAG" > "$temp_file"
    fi
    
    if [ -n "$VERSION" ]; then
        echo "ccAgents $VERSION - $(date +%Y-%m-%d)"
    else
        echo "ccAgents Unreleased Changes"
    fi
    echo "================================="
    echo ""
    
    cat "$temp_file"
    
    rm -f "$temp_file"
}

# Update existing changelog
update_changelog() {
    local new_content="$1"
    
    if [ ! -f "$CHANGELOG_FILE" ]; then
        error "Changelog file not found: $CHANGELOG_FILE"
    fi
    
    info "Updating $CHANGELOG_FILE..."
    
    # Create backup
    cp "$CHANGELOG_FILE" "${CHANGELOG_FILE}.backup"
    
    # Find insertion point (after [Unreleased] section)
    local unreleased_line=$(grep -n "## \[Unreleased\]" "$CHANGELOG_FILE" | head -1 | cut -d: -f1)
    
    if [ -z "$unreleased_line" ]; then
        error "Could not find [Unreleased] section in $CHANGELOG_FILE"
    fi
    
    # Find next section
    local next_section_line=$(tail -n +$((unreleased_line + 1)) "$CHANGELOG_FILE" | grep -n "^## " | head -1 | cut -d: -f1)
    
    if [ -n "$next_section_line" ]; then
        next_section_line=$((unreleased_line + next_section_line))
    else
        next_section_line=$(($(wc -l < "$CHANGELOG_FILE") + 1))
    fi
    
    # Insert new content
    head -n "$unreleased_line" "$CHANGELOG_FILE" > "$TEMP_CHANGELOG"
    echo "" >> "$TEMP_CHANGELOG"
    echo -e "$new_content" >> "$TEMP_CHANGELOG"
    tail -n +$next_section_line "$CHANGELOG_FILE" >> "$TEMP_CHANGELOG"
    
    mv "$TEMP_CHANGELOG" "$CHANGELOG_FILE"
    success "Changelog updated successfully"
}

# Main function
main() {
    parse_args "$@"
    get_git_info
    
    info "Generating changelog in $FORMAT format..."
    
    local changelog_content=""
    
    case "$FORMAT" in
        markdown)
            changelog_content=$(generate_markdown_changelog)
            ;;
        json)
            changelog_content=$(generate_json_changelog)
            ;;
        text)
            changelog_content=$(generate_text_changelog)
            ;;
        *)
            error "Unsupported format: $FORMAT"
            ;;
    esac
    
    if [ -z "$changelog_content" ]; then
        error "Failed to generate changelog content"
    fi
    
    # Output results
    if [ "$OUTPUT_FILE" = "$CHANGELOG_FILE" ] && [ "$FORMAT" = "markdown" ] && [ -f "$CHANGELOG_FILE" ]; then
        update_changelog "$changelog_content"
    else
        echo -e "$changelog_content" > "$OUTPUT_FILE"
        success "Changelog generated: $OUTPUT_FILE"
    fi
}

# Run main function
main "$@"