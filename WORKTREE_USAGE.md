# Enhanced Git Worktree Support for CCA

## Overview

The CCA project now supports creating git worktrees within the project directory at `{project_root}/.cca/worktrees/`. This provides isolated workspaces for different operations while keeping everything project-local.

## Key Features

### 1. Project-Local Worktrees
- Creates worktrees in `{project_root}/.cca/worktrees/`
- Keeps all agent operations within the project
- Easy to manage and clean up
- No interference with the main codebase

### 2. Specialized Worktree Types

#### Agent Worktrees
```go
wm := git.NewProjectWorktreeManager(projectRoot)
worktree, err := wm.CreateAgentWorktree("security", "vulnerability-scan")
// Creates: .cca/worktrees/agent-security-vulnerability-scan-20240103-143052/
// Branch: cca/agent-security-vulnerability-scan-20240103-143052
```

#### Experiment Worktrees
```go
worktree, err := wm.CreateExperimentWorktree("claude-integration-v2")
// Creates: .cca/worktrees/experiment-claude-integration-v2-20240103-143052/
// Branch: cca/experiment-claude-integration-v2-20240103-143052
```

#### Task Worktrees
```go
worktree, err := wm.CreateTaskWorktree("TASK-123")
// Creates: .cca/worktrees/task-TASK-123-20240103-143052/
// Branch: cca/task-TASK-123-20240103-143052
```

#### Custom Worktrees
```go
worktree, err := wm.CreateProjectWorktree("refactor-agents", "main")
// Creates: .cca/worktrees/refactor-agents-20240103-143052/
// Branch: cca/refactor-agents-20240103-143052
```

## Project Structure

```
project-root/
├── .cca/
│   └── worktrees/
│       ├── agent-security-vulnerability-scan-20240103-143052/
│       ├── experiment-claude-integration-v2-20240103-143052/
│       ├── task-TASK-123-20240103-143052/
│       └── refactor-agents-20240103-143052/
├── src/
├── pkg/
└── other project files...
```

## Use Cases

### 1. Agent Isolation
Each agent can work in its own dedicated worktree:
- Security agent scans for vulnerabilities
- Architecture agent analyzes code structure  
- Documentation agent updates docs
- No conflicts between concurrent operations

### 2. Experimental Development
Test new features safely:
- Create experiment worktrees for new ideas
- Multiple experiments can run in parallel
- Easy to discard failed experiments
- Merge successful changes back to main

### 3. Task-Specific Development
Organize development by tasks:
- Each feature/bug gets its own worktree
- Switch between tasks without losing work
- Clean separation of concerns
- Easy merge when task is complete

### 4. CI/CD Operations
Isolated build/test environments:
- Build in dedicated worktrees
- No interference with ongoing development
- Parallel builds for different branches
- Clean state for each operation

## API Reference

### WorktreeManager Creation
```go
// Global worktree manager (uses ~/.ccagents/worktrees/)
wm := git.NewWorktreeManager(basePath)

// Project-local worktree manager (uses .cca/worktrees/)
wm := git.NewProjectWorktreeManager(projectRoot)
```

### Worktree Creation Methods
```go
// General project worktree
CreateProjectWorktree(name, baseBranch string) (*Worktree, error)

// Specialized worktree types
CreateAgentWorktree(agentName, taskName string) (*Worktree, error)
CreateExperimentWorktree(experimentName string) (*Worktree, error)
CreateTaskWorktree(taskID string) (*Worktree, error)

// Original issue-based worktree (for external repos)
CreateWorktree(issueRef *types.IssueReference) (*Worktree, error)
```

### Worktree Management
```go
// List all worktrees
ListWorktrees() ([]*Worktree, error)

// Clean up specific worktree
CleanupWorktree(worktree *Worktree) error

// Clean up old worktrees (older than specified duration)
CleanupOldWorktrees(maxAge time.Duration) error
```

### Worktree Information
```go
// Get worktree details
worktree.GetWorktreeInfo() map[string]interface{}

// Validate worktree state
worktree.ValidateWorktree() error

// Get current branch in worktree
worktree.GetCurrentBranch() (string, error)
```

## Benefits

### Project Organization
- **Local Management**: All worktrees stay within the project
- **Clear Structure**: Organized under `.cca/worktrees/`
- **Easy Discovery**: All team members can see active worktrees
- **Simple Cleanup**: Remove `.cca/` directory to clean everything

### Development Workflow
- **Parallel Work**: Multiple developers/agents can work simultaneously
- **Isolation**: Changes don't interfere with each other
- **Safety**: Experiments can't break the main codebase
- **Flexibility**: Easy to switch between different workstreams

### Performance
- **Efficient**: Git worktrees share object database
- **Fast**: No need to clone repositories multiple times
- **Space-Efficient**: Minimal disk usage compared to full clones
- **Quick Switching**: Instant workspace changes

## Integration with CCA Agents

Each agent can create its own worktree for isolation:

```go
// Security Agent
securityWorktree, err := wm.CreateAgentWorktree("security", "audit-dependencies")
// Work in: .cca/worktrees/agent-security-audit-dependencies-{timestamp}/

// Architecture Agent  
archWorktree, err := wm.CreateAgentWorktree("architecture", "analyze-patterns")
// Work in: .cca/worktrees/agent-architecture-analyze-patterns-{timestamp}/

// Documentation Agent
docsWorktree, err := wm.CreateAgentWorktree("documentation", "update-api-docs")
// Work in: .cca/worktrees/agent-documentation-update-api-docs-{timestamp}/
```

This ensures that:
1. Agents never interfere with each other
2. Main codebase remains stable during agent operations
3. Agent work can be reviewed before merging
4. Failed agent operations don't affect the project
5. Multiple agents can work on the same repository simultaneously

## Git Ignore Recommendations

Add to your `.gitignore`:
```
# CCA worktrees and temporary files
.cca/
```

This ensures worktrees aren't accidentally committed while allowing them to be shared if needed.