# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Build and Development
```bash
# Build the application
make build

# Build with debug symbols for development
make build-dev

# Install the application
make install

# Run complete development workflow (format, lint, test)
make dev
```

### Code Quality
```bash
# Format Go code
make fmt

# Run linters
make lint

# Tidy Go modules
make mod-tidy
```

### Testing
```bash
# Run all tests
make test

# Run tests with verbose output
make test-verbose

# Generate test coverage report
make test-coverage

# Generate HTML coverage report
make test-coverage-html

# CI-optimized test targets (fast execution)
make ci-test-fast      # Ultra-fast unit tests (<2 minutes)
make ci-test-unit      # Unit tests with coverage (~2 minutes)
make ci-test-integration # Integration tests (~3 minutes)
```

### Cross-platform Builds
```bash
# Build for all platforms
make build-all

# Build for specific platforms
make build-linux
make build-darwin
make build-windows

# Create release archives and checksums
make release-prep
```

### Development Dependencies
```bash
# Install development dependencies (golangci-lint, goimports, gofumpt)
make dev-deps
```

## Architecture Overview

ccAgents is an AI-powered GitHub automation tool built with a multi-agent architecture:

### Core Architecture Components

**Multi-Agent System** (`pkg/agents/`):
- `Agent` interface defines all specialized agents (Security, Architecture, Documentation, Testing, Performance)
- Agent communication via message bus with typed messages (`AgentMessage`, `MessageType`)
- Agent registry manages agent lifecycle and health monitoring
- Result aggregation handles conflicts and prioritizes recommendations

**Workflow Engine** (`pkg/workflow/`):
- Orchestrates complex automation workflows with stage-based execution
- Supports parallel and sequential stage execution with dependency management
- State management with persistence and recovery capabilities
- Event-driven architecture with comprehensive monitoring

**Claude Code Integration** (`pkg/claude/`):
- Primary interface to Claude Code for AI-powered code generation
- Process management with timeout and retry handling
- Prompt engineering for different automation contexts

**Analysis Framework** (`pkg/analysis/`):
- Context-aware codebase analysis with framework/language detection
- Dependency analysis for understanding code relationships
- Smart caching to optimize repeated operations

### Key Patterns

**Agent Communication**: All agents communicate through typed messages on a central message bus. Use `AgentMessage` with appropriate `MessageType` for coordination.

**Workflow Stages**: Complex operations are broken into workflow stages with dependency tracking. Each stage can execute in parallel or sequentially based on configuration.

**Configuration Management** (`pkg/config/`): Uses hierarchical YAML configuration with environment-specific overrides and validation.

**Error Handling**: Comprehensive error types in `pkg/errors/` with retry mechanisms and context preservation.

## Project Structure

- `cmd/ccagents/` - CLI entry point and command definitions
- `pkg/agents/` - Multi-agent system implementation
- `pkg/workflow/` - Workflow engine and orchestration
- `pkg/claude/` - Claude Code client and process management
- `pkg/analysis/` - Codebase analysis and context building
- `pkg/git/` - Git operations and repository management
- `pkg/github/` - GitHub API integration with rate limiting
- `pkg/config/` - Configuration management system
- `pkg/observability/` - Metrics, logging, and monitoring

## Testing Patterns

- Unit tests alongside source files (e.g., `analyzer_test.go`)
- Integration tests in `test/` directory
- Use testify for assertions: `github.com/stretchr/testify`
- Mock interfaces for external dependencies
- Table-driven tests for comprehensive coverage

## Key Dependencies

- **Cobra** (`github.com/spf13/cobra`) - CLI framework
- **Bubble Tea** (`github.com/charmbracelet/bubbletea`) - TUI framework
- **go-git** (`github.com/go-git/go-git/v5`) - Git operations
- **GitHub API** (`github.com/google/go-github/v60`) - GitHub integration
- **GitHub CLI** (`github.com/cli/go-gh/v2`) - GitHub CLI integration

## Configuration

Uses YAML configuration files (`.ccagents.yaml`) with sections for:
- `claude` - Claude Code settings (model, tokens, temperature)
- `github` - GitHub API configuration
- `workflow` - Automation workflow settings
- `ui` - Terminal UI preferences

### Debug Mode

To enable enhanced debugging with full Claude prompt display:

```bash
# Option 1: Use the --debug flag (automatically enables verbose Claude debugging)
./bin/ccagents --debug [command]

# Option 2: Manually set the environment variable
export CCAGENTS_VERBOSE_DEBUG=true
./bin/ccagents [command]
```

When debug mode is enabled, the application will display:
- Full Claude prompt content with clear separator lines
- Prompt character count
- Command execution details
- Working directory and execution context
- Additional workflow debugging information

**Note**: The `--debug` flag automatically enables `CCAGENTS_VERBOSE_DEBUG` for comprehensive debugging.

## Common Development Tasks

When working on new features:
1. Implement agent interfaces if adding new specialized functionality
2. Use workflow stages for multi-step operations
3. Add appropriate configuration options with validation
4. Include comprehensive error handling with context
5. Add observability metrics for monitoring
6. Write tests covering both success and error cases

Claude Code Reliability-First Development Guide
🎯 Core Principle: Certainty Over Speed
Priority Order:

Task Success Rate - 100% completion is mandatory
Code Quality - Production-ready output required
Error Prevention - Comprehensive validation at each step
Speed/Cost - Secondary considerations
🛡️ Mandatory Safety Protocols
Pre-Execution Requirements
xml
<mandatory>Always perform these steps before any code changes</mandatory>

<step>1</step>
<action>Create backup branch or commit current state</action>
<verification>Confirm git status is clean</verification>

<step>2</step>  
<action>Analyze complete task scope and dependencies</action>
<verification>Document all affected files and systems</verification>

<step>3</step>
<action>Plan rollback strategy for each change</action>
<verification>Test rollback procedure before implementation</verification>
Execution Validation Steps
xml
<priority>1</priority>
<rule>After each file modification, immediately verify syntax</rule>

<priority>2</priority>
<rule>Run relevant tests after each logical change</rule>

<priority>3</priority>
<rule>Confirm build passes before proceeding to next step</rule>
📋 Step-by-Step Execution Protocol
Level 1: Analysis Phase
bash
# ALWAYS start with comprehensive analysis
"Before making any changes, please:
1. Read and understand all relevant existing code
2. Identify potential conflicts and dependencies  
3. Create a detailed step-by-step plan
4. Estimate risk level for each step
5. Wait for my approval before proceeding"
Level 2: Implementation Phase
xml
<approach>Incremental changes with validation</approach>
<pattern>
  Change → Test → Verify → Commit → Next Change
</pattern>
<forbidden>
  - Multiple file changes without testing
  - Large refactoring in single step
  - Skipping verification steps
</forbidden>
Level 3: Verification Phase
bash
# Required after each implementation step
1. Syntax check (linting, compilation)
2. Unit test execution
3. Integration test execution  
4. Manual verification of core functionality
5. Performance impact assessment
🔧 Error Prevention Strategies
Input Validation
xml
<rule>Validate all inputs before processing</rule>
<example>
  Before: user.email = input
  After: user.email = validateEmail(input) || throw Error
</example>
Defensive Programming
xml
<mandatory>Always include error handling</mandatory>
<pattern>
try {
  // Implementation
} catch (error) {
  // Specific error handling
  // Logging
  // Graceful degradation
}
</pattern>
Dependency Management
xml
<rule>Explicitly check all dependencies before use</rule>
<example>
  if (!requiredService.isAvailable()) {
    throw new Error('Required service unavailable');
  }
</example>
🧪 Test-Driven Development Requirements
Mandatory Testing Sequence
bash
1. Write failing test first
2. Implement minimal code to pass test
3. Refactor while maintaining test pass
4. Add edge case tests
5. Verify all existing tests still pass
Test Coverage Requirements
 Unit tests: All functions/methods
 Integration tests: All API endpoints
 Edge cases: Error conditions, boundary values
 Regression tests: Previous bug scenarios
🔄 Rollback and Recovery
Automatic Checkpoints
xml
<rule>Create git commits at each major step</rule>
<pattern>
git add .
git commit -m "Step X: [specific change description]"
</pattern>
Recovery Procedures
bash
# If any step fails:
1. Stop immediately
2. Assess failure cause
3. Rollback to last known good state
4. Re-analyze approach
5. Implement alternative solution
6. Never force through errors
📊 Quality Gates
Code Quality Checklist
 Readability: Clear variable names, proper comments
 Performance: No obvious performance regressions
 Security: No security vulnerabilities introduced
 Maintainability: Follows project conventions
 Documentation: Updated for any API changes
Completion Criteria
xml
<mandatory>Task is NOT complete until ALL conditions met</mandatory>
<criteria>
  ✅ All tests pass
  ✅ Build succeeds
  ✅ No linting errors
  ✅ Documentation updated
  ✅ Manual verification completed
  ✅ Performance acceptable
  ✅ Security review passed
</criteria>
🎯 Execution Commands
High-Reliability Mode
bash
# Use these phrases to enforce reliability-first approach:
"Please implement this with maximum reliability, including all safety checks"
"Prioritize correctness over speed, with comprehensive testing at each step"
"Use defensive programming and include all error handling"
Verification Commands
bash
"After each change, verify the build still works"
"Run all relevant tests before proceeding"
"Create a checkpoint commit before the next step"
🚫 Prohibited Practices
Never Do
❌ Skip testing phases for "simple" changes
❌ Make multiple unrelated changes simultaneously ❌ Proceed if any verification step fails ❌ Assume dependencies will work without checking ❌ Implement without understanding existing code

Always Do
✅ Plan before implementing ✅ Test incrementally ✅ Verify at each step
✅ Document changes ✅ Create recovery points ✅ Think through edge cases

🔍 Mandatory Post-Implementation Review
Self-Review Protocol
xml
<mandatory>After completing any code implementation, ALWAYS perform comprehensive self-review</mandatory>

<phase>1. Code Quality Review</phase>
<actions>
  - Review variable naming conventions
  - Check function/method complexity
  - Validate error handling coverage
  - Assess code readability and maintainability
</actions>

<phase>2. Architecture Review</phase>
<actions>
  - Evaluate design patterns usage
  - Check separation of concerns
  - Assess coupling and cohesion
  - Validate SOLID principles adherence
</actions>

<phase>3. Performance & Security Review</phase>
<actions>
  - Identify potential performance bottlenecks
  - Check for security vulnerabilities
  - Validate input sanitization
  - Assess resource usage efficiency
</actions>
Automated Review Questions
bash
# ALWAYS ask these questions after implementation:

1. "Can this code be simplified or made more readable?"
2. "Does this follow the established architecture patterns?"
3. "Are there any potential security issues?"
4. "Could this be more performant?"
5. "Is this maintainable by other developers?"
6. "Does this violate any SOLID principles?"
7. "Are there opportunities for better abstraction?"
Architecture Improvement Process
xml
<step>1</step>
<action>Analyze current implementation against architecture principles</action>
<check>Single Responsibility, Open/Closed, Liskov Substitution, Interface Segregation, Dependency Inversion</check>

<step>2</step>
<action>Identify refactoring opportunities</action>
<check>Extract methods, move responsibilities, improve abstractions</check>

<step>3</step>
<action>Suggest design pattern improvements</action>
<check>Strategy, Factory, Observer, Decorator, etc.</check>

<step>4</step>
<action>Implement improvements incrementally</action>
<check>Test after each refactoring step</check>
🔄 Post-Code Review Workflow
Phase 1: Immediate Self-Review
bash
# After completing implementation:
"Now I'll perform a self-review of the implemented code:

1. Code Quality Assessment
   - Checking naming conventions
   - Reviewing function complexity
   - Validating error handling

2. Architecture Review
   - Evaluating design patterns
   - Checking separation of concerns
   - Assessing maintainability

3. Improvement Identification
   - Finding refactoring opportunities
   - Suggesting architecture enhancements"
Phase 2: Architecture Assessment
xml
<evaluation>
  <aspect>Modularity</aspect>
  <question>Is the code properly modularized?</question>
  <action>Extract modules if needed</action>

  <aspect>Coupling</aspect>
  <question>Are components loosely coupled?</question>
  <action>Introduce interfaces/abstractions</action>

  <aspect>Testability</aspect>
  <question>Is the code easily testable?</question>
  <action>Refactor for dependency injection</action>

  <aspect>Extensibility</aspect>
  <question>Can new features be added easily?</question>
  <action>Apply Open/Closed principle</action>
</evaluation>
Phase 3: Improvement Implementation
bash
# If improvements are identified:
"I've identified several improvement opportunities:

1. Architecture Enhancement: [specific suggestion]
2. Code Quality Improvement: [specific suggestion]  
3. Performance Optimization: [specific suggestion]

Implementing these improvements now..."
📊 Review Checklist Template
Code Quality Metrics
 Readability: Clear and self-documenting code
 Complexity: Functions under 20 lines, low cyclomatic complexity
 Naming: Descriptive variable and function names
 Comments: Meaningful comments for complex logic
 Consistency: Follows project coding standards
Architecture Metrics
 Single Responsibility: Each class/function has one reason to change
 Open/Closed: Open for extension, closed for modification
 Dependency Inversion: Depend on abstractions, not concretions
 Interface Segregation: No forced dependency on unused methods
 DRY Principle: No code duplication
Refactoring Opportunities
 Extract Method: Break down large functions
 Extract Class: Separate responsibilities
 Move Method: Place methods in appropriate classes
 Replace Magic Numbers: Use named constants
 Simplify Conditionals: Reduce nested if statements
🛠️ Automated Improvement Commands
Built-in Review Triggers
bash
# These phrases automatically trigger post-implementation review:
"Please review and improve the implemented code"
"Perform architecture assessment and suggest improvements"
"Apply best practices and refactor if needed"
Specific Review Types
bash
# Architecture-focused review
"Review the architecture and suggest design pattern improvements"

# Performance-focused review  
"Analyze performance and optimize if necessary"

# Security-focused review
"Perform security review and address any vulnerabilities"

# Maintainability-focused review
"Review for maintainability and long-term code health"
⚡ Context Window Management Strategy
AgentTool-First Approach for Reliability
xml
<principle>Distribute complex tasks across AgentTools to maintain main agent context clarity</principle>

<benefits>
  - Preserved main agent focus
  - Specialized task execution
  - Reduced context pollution
  - Enhanced error isolation
  - Improved long-term reliability
</benefits>
Mandatory AgentTool Usage
xml
<rule>Launch AgentTool for ANY task meeting these criteria</rule>

<criteria>
  - Multi-file operations (3+ files)
  - Complex refactoring tasks
  - Comprehensive code reviews
  - Architecture analysis
  - Independent module development
  - Self-contained feature implementation
</criteria>
Context Preservation Protocol
bash
# Main Agent: High-level coordination only
Main Agent Tasks:
- Overall strategy planning
- AgentTool coordination
- Final integration review
- Cross-module validation

# AgentTool: Specialized execution
AgentTool Tasks:
- Detailed implementation
- Component-specific review
- Isolated testing
- Architecture compliance check
AgentTool Review Distribution
xml
<strategy>Distribute post-implementation reviews across specialized AgentTools</strategy>

<distribution>
  AgentTool1: Code Quality Review
    - Syntax and style validation
    - Naming convention checks
    - Function complexity analysis

  AgentTool2: Architecture Review  
    - SOLID principles validation
    - Design pattern assessment
    - Coupling/cohesion analysis

  AgentTool3: Security & Performance Review
    - Vulnerability scanning
    - Performance bottleneck identification
    - Resource usage optimization
</distribution>
🔄 Enhanced Execution Workflow
Phase 1: Task Distribution (Main Agent)
bash
"I'll distribute this complex task across multiple AgentTools to ensure reliability:

Planning:
1. Analyzing task complexity and scope
2. Identifying independent components  
3. Planning AgentTool distribution strategy
4. Setting up coordination framework

AgentTool Assignment:
- AgentTool1: Component A implementation + review
- AgentTool2: Component B implementation + review  
- AgentTool3: Integration testing + architecture review
- Main Agent: Final coordination and validation"
Phase 2: Specialized Execution (AgentTools)
xml
<pattern>Each AgentTool performs complete cycle</pattern>

<cycle>
  Implementation → Self-Review → Architecture Assessment → Testing → Report
</cycle>

<isolation>
  Each AgentTool maintains focused context on specific component
  No cross-contamination of complex requirements
  Specialized attention to component-specific concerns
</isolation>
Phase 3: Integration & Final Review (Main Agent)
bash
# Main Agent coordinates final steps with clean context:
"Collecting AgentTool results:

AgentTool1 Report: ✅ Component A completed with architecture improvements
AgentTool2 Report: ✅ Component B completed with performance optimizations  
AgentTool3 Report: ✅ Integration tests pass, architecture validated

Final Integration:
1. Cross-component compatibility check
2. System-level architecture review
3. End-to-end testing coordination
4. Documentation consolidation"
📊 Context Window Optimization Benefits
Main Agent Context Stays Clean
xml
<before>
  Main context: Task details + File1 + File2 + File3 + Review1 + Review2 + Errors + Fixes...
  Result: Context pollution, reduced performance
</before>

<after>
  Main context: High-level coordination only
  AgentTool contexts: Specialized, focused, efficient
  Result: Enhanced reliability and performance
</after>
AgentTool Specialization Advantages
bash
# Each AgentTool maintains laser focus:

AgentTool (Auth Module):
- Deep context on authentication patterns
- Specialized security review
- Auth-specific performance optimization

AgentTool (Database Layer):  
- Deep context on data patterns
- Specialized query optimization
- Database-specific architecture review

AgentTool (API Layer):
- Deep context on endpoint patterns  
- Specialized API design review
- REST/GraphQL specific optimizations
🎯 Strategic AgentTool Deployment
Reliability-First AgentTool Usage
xml
<deployment>Always prefer AgentTool over main agent for</deployment>

<scenarios>
  ✅ Multi-step refactoring requiring deep context
  ✅ Component-specific architecture reviews
  ✅ Isolated feature development with testing
  ✅ Complex debugging requiring focused analysis
  ✅ Performance optimization for specific modules
  ✅ Security review of authentication/authorization
</scenarios>
Context Window Triggers
bash
# Automatically launch AgentTool when:
IF (task_complexity > simple) AND (files_affected > 2) THEN
    launch_agent_tool_with_specialized_context()

IF (main_context_length > 70%) THEN  
    delegate_to_agent_tool_and_clear_main_context()

IF (review_depth == comprehensive) THEN
    launch_specialized_review_agent_tools()
💡 Complete Context-Optimized Workflow
bash
User: "Implement comprehensive user management system with authentication"

Main Agent Response:
"I'll implement this using AgentTool distribution for maximum reliability:

TASK ANALYSIS:
- Complex multi-component system
- Requires specialized reviews
- Benefits from context distribution

AGENTOOL DEPLOYMENT PLAN:
1. AgentTool1: User model + validation + review
2. AgentTool2: Authentication service + security review  
3. AgentTool3: API endpoints + performance review
4. AgentTool4: Integration tests + architecture review
5. Main Agent: Final coordination + system validation

EXECUTION:
[Launching AgentTool1 for User model...]
[Launching AgentTool2 for Authentication...]
[Launching AgentTool3 for API layer...]
[Launching AgentTool4 for Integration...]

COORDINATION:
✅ AgentTool1: User model completed with SOLID compliance
✅ AgentTool2: Auth service completed with security hardening
✅ AgentTool3: API layer completed with performance optimization  
✅ AgentTool4: Integration tests pass, architecture validated

FINAL INTEGRATION:
[Main Agent performing final system review with clean context]
✅ Cross-component compatibility verified
✅ System-level architecture approved
✅ End-to-end functionality confirmed
✅ Production-ready implementation delivered"
🔄 Context Management Best Practices
Context Pollution Prevention
xml
<rule>Use /clear command before major context shifts</rule>
<rule>Delegate detailed work to AgentTools</rule>
<rule>Keep main agent focused on coordination</rule>
<rule>Consolidate AgentTool results efficiently</rule>
</xml>

### Long-Term Reliability Benefits
```bash
Benefits of AgentTool Distribution:
✅ Sustained high-quality output throughout long sessions
✅ Reduced context-related errors and omissions  
✅ Enhanced specialization in component reviews
✅ Improved error isolation and debugging
✅ Better scalability for complex projects
💡 Complete Example Workflow
bash
User: "Add user authentication to the API"

Expected Claude Code Response:
"I'll implement user authentication with maximum reliability. Let me start by:

1. Analyzing the current API structure
2. Reading existing auth-related code
3. Creating a detailed implementation plan
4. Setting up a backup branch
5. Writing tests first

Please confirm this approach before I proceed.

[After implementation completion]

Now I'll perform comprehensive self-review:

CODE QUALITY REVIEW:
✅ Function complexity: Average 8 lines per function
✅ Naming conventions: Clear and descriptive
✅ Error handling: Comprehensive try-catch blocks
⚠️  Opportunity: Extract validation logic to separate module

ARCHITECTURE REVIEW:
✅ Single Responsibility: Each class has clear purpose
✅ Dependency Injection: Properly implemented
⚠️  Improvement: Could apply Strategy pattern for auth methods
⚠️  Enhancement: Consider separating JWT logic into service

IMPLEMENTING IMPROVEMENTS:
1. Extracting AuthValidator class...
2. Implementing Strategy pattern for auth methods...
3. Creating JWTService abstraction...

FINAL VERIFICATION:
✅ All tests pass
✅ Architecture improved
✅ Code quality enhanced
✅ Ready for production"
Remember: Code is written once but read many times. Post-implementation review ensures long-term maintainability and quality.

