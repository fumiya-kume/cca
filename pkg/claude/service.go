package claude

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/internal/types"
)

// Service provides high-level Claude Code operations
type Service struct {
	client              *Client
	promptTemplates     map[string]PromptTemplate
	conversationHistory map[string][]ConversationEntry
	historyMutex        sync.RWMutex

	// Configuration
	defaultTimeout time.Duration
	retryAttempts  int
	retryDelay     time.Duration

	// Session pools for different contexts
	issueAnalysisSessions  map[string]*Session
	codeGenerationSessions map[string]*Session
	reviewSessions         map[string]*Session
	sessionPoolMutex       sync.RWMutex
}

// PromptTemplate defines reusable prompt templates
type PromptTemplate struct {
	Name        string
	Template    string
	Variables   []string
	Description string
	Category    PromptCategory
}

// PromptCategory categorizes different types of prompts
type PromptCategory int

const (
	PromptIssueAnalysis PromptCategory = iota
	PromptCodeGeneration
	PromptCodeReview
	PromptTesting
	PromptDocumentation
	PromptDebugging
	PromptRefactoring
)

// ConversationEntry represents a single interaction in a conversation
type ConversationEntry struct {
	Timestamp time.Time
	Role      ConversationRole
	Content   string
	Metadata  map[string]interface{}
}

// ConversationRole defines who sent the message
type ConversationRole int

const (
	RoleUser ConversationRole = iota
	RoleAssistant
	RoleSystem
)

// ServiceConfig configures the Claude service
type ServiceConfig struct {
	ClientConfig   ClientConfig
	DefaultTimeout time.Duration
	RetryAttempts  int
	RetryDelay     time.Duration
}

// Default service configuration values
const (
	DefaultServiceTimeout = 5 * time.Minute
	DefaultRetryAttempts  = 3
	DefaultRetryDelay     = 2 * time.Second
)

// NewService creates a new Claude service
func NewService(config ServiceConfig) (*Service, error) {
	// Set defaults
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = DefaultServiceTimeout
	}
	if config.RetryAttempts == 0 {
		config.RetryAttempts = DefaultRetryAttempts
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = DefaultRetryDelay
	}

	// Create client
	client, err := NewClient(config.ClientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create claude client: %w", err)
	}

	service := &Service{
		client:                 client,
		promptTemplates:        getDefaultPromptTemplates(),
		conversationHistory:    make(map[string][]ConversationEntry),
		defaultTimeout:         config.DefaultTimeout,
		retryAttempts:          config.RetryAttempts,
		retryDelay:             config.RetryDelay,
		issueAnalysisSessions:  make(map[string]*Session),
		codeGenerationSessions: make(map[string]*Session),
		reviewSessions:         make(map[string]*Session),
	}

	return service, nil
}

// AnalyzeIssue analyzes a GitHub issue and provides development guidance
func (s *Service) AnalyzeIssue(ctx context.Context, issueData *types.IssueData, codeContext *types.CodeContext) (*IssueAnalysis, error) {
	fmt.Println("🔍 Debug: AnalyzeIssue starting...")
	sessionKey := fmt.Sprintf("issue_%s_%d", issueData.Repository, issueData.Number)
	fmt.Printf("🔍 Debug: Session key: %s\n", sessionKey)

	// Get or create dedicated session for issue analysis
	fmt.Println("🔍 Debug: Getting or creating session...")
	session, err := s.getOrCreateSession(ctx, sessionKey, s.issueAnalysisSessions, issueData.WorkingDirectory)
	if err != nil {
		fmt.Printf("❌ Failed to get analysis session: %v\n", err)
		return nil, fmt.Errorf("failed to get analysis session: %w", err)
	}
	fmt.Printf("✅ Session created: %s\n", session.ID)

	// Build analysis prompt
	fmt.Println("🔍 Debug: Building analysis prompt...")
	prompt, err := s.buildIssueAnalysisPrompt(issueData, codeContext)
	if err != nil {
		fmt.Printf("❌ Failed to build analysis prompt: %v\n", err)
		return nil, fmt.Errorf("failed to build analysis prompt: %w", err)
	}
	fmt.Printf("✅ Prompt built (%d chars)\n", len(prompt))

	// Execute analysis
	fmt.Println("🔍 Debug: Executing prompt with Claude...")
	response, err := s.executeWithRetry(ctx, session, prompt)
	if err != nil {
		fmt.Printf("❌ Failed to execute issue analysis: %v\n", err)
		return nil, fmt.Errorf("failed to execute issue analysis: %w", err)
	}
	fmt.Printf("✅ Got response (%d chars)\n", len(response))

	// Parse analysis response
	fmt.Println("🔍 Debug: Parsing analysis response...")
	analysis, err := s.parseIssueAnalysis(response)
	if err != nil {
		fmt.Printf("❌ Failed to parse analysis response: %v\n", err)
		return nil, fmt.Errorf("failed to parse analysis response: %w", err)
	}
	fmt.Println("✅ Analysis parsed")

	// Update conversation history
	s.updateConversationHistory(sessionKey, prompt, response)

	return analysis, nil
}

// GenerateCode generates code based on issue requirements and context
func (s *Service) GenerateCode(ctx context.Context, request *CodeGenerationRequest) (*CodeGenerationResult, error) {
	sessionKey := fmt.Sprintf("codegen_%s_%d", request.Repository, request.IssueNumber)

	// Get or create dedicated session for code generation
	session, err := s.getOrCreateSession(ctx, sessionKey, s.codeGenerationSessions, request.WorkingDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to get generation session: %w", err)
	}

	// Build generation prompt
	prompt, err := s.buildCodeGenerationPrompt(request)
	if err != nil {
		return nil, fmt.Errorf("failed to build generation prompt: %w", err)
	}

	// Execute generation
	response, err := s.executeWithRetry(ctx, session, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to execute code generation: %w", err)
	}

	// Parse generation response
	result, err := s.parseCodeGenerationResult(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse generation response: %w", err)
	}

	// Update conversation history
	s.updateConversationHistory(sessionKey, prompt, response)

	return result, nil
}

// ReviewCode performs code review using Claude
func (s *Service) ReviewCode(ctx context.Context, request *CodeReviewRequest) (*CodeReviewResult, error) {
	sessionKey := fmt.Sprintf("review_%s_%s", request.Repository, request.Branch)

	// Get or create dedicated session for code review
	session, err := s.getOrCreateSession(ctx, sessionKey, s.reviewSessions, request.WorkingDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to get review session: %w", err)
	}

	// Build review prompt
	prompt, err := s.buildCodeReviewPrompt(request)
	if err != nil {
		return nil, fmt.Errorf("failed to build review prompt: %w", err)
	}

	// Execute review
	response, err := s.executeWithRetry(ctx, session, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to execute code review: %w", err)
	}

	// Parse review response
	result, err := s.parseCodeReviewResult(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse review response: %w", err)
	}

	// Update conversation history
	s.updateConversationHistory(sessionKey, prompt, response)

	return result, nil
}

// SendInteractiveCommand sends a command and handles interactive prompts
func (s *Service) SendInteractiveCommand(ctx context.Context, sessionID, command string, inputHandler InputHandler) (*CommandResult, error) {
	// Get session
	session, err := s.getSessionByID(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Send command
	if err := s.client.ExecuteCommand(session, command); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	result := &CommandResult{
		Command:   command,
		StartTime: time.Now(),
		Output:    []string{},
		Errors:    []string{},
	}

	// Monitor output and handle interactive prompts
	for {
		select {
		case <-ctx.Done():
			result.EndTime = time.Now()
			result.Success = false
			result.Error = ctx.Err()
			return result, ctx.Err()

		case output := <-s.client.GetSessionOutput(session):
			result.Output = append(result.Output, output.Content)

			// Check if it's a prompt requiring input
			if output.Type == OutputPrompt && inputHandler != nil {
				input, err := inputHandler.HandlePrompt(output.Content)
				if err != nil {
					result.EndTime = time.Now()
					result.Success = false
					result.Error = err
					return result, err
				}

				// Send input
				if err := s.client.sendInput(session, input); err != nil {
					result.EndTime = time.Now()
					result.Success = false
					result.Error = err
					return result, err
				}
			}

		case err := <-s.client.GetSessionErrors(session):
			result.Errors = append(result.Errors, err.Error())

		case status := <-s.client.GetSessionStatus(session):
			switch status {
			case SessionIdle:
				// Command completed
				result.EndTime = time.Now()
				result.Success = len(result.Errors) == 0
				return result, nil
			case SessionError, SessionClosed:
				result.EndTime = time.Now()
				result.Success = false
				if len(result.Errors) > 0 {
					result.Error = fmt.Errorf("session error: %s", strings.Join(result.Errors, "; "))
				}
				return result, result.Error
			}
		}
	}
}

// GetConversationHistory retrieves conversation history for a session
func (s *Service) GetConversationHistory(sessionKey string) []ConversationEntry {
	s.historyMutex.RLock()
	defer s.historyMutex.RUnlock()

	history, exists := s.conversationHistory[sessionKey]
	if !exists {
		return []ConversationEntry{}
	}

	// Return a copy to prevent external modification
	result := make([]ConversationEntry, len(history))
	copy(result, history)
	return result
}

// AddPromptTemplate adds a custom prompt template
func (s *Service) AddPromptTemplate(template PromptTemplate) {
	s.promptTemplates[template.Name] = template
}

// GetPromptTemplate retrieves a prompt template by name
func (s *Service) GetPromptTemplate(name string) (PromptTemplate, bool) {
	template, exists := s.promptTemplates[name]
	return template, exists
}

// Shutdown gracefully shuts down the service
func (s *Service) Shutdown(ctx context.Context) error {
	return s.client.Shutdown(ctx)
}

// Helper methods

func (s *Service) getOrCreateSession(ctx context.Context, sessionKey string, sessionPool map[string]*Session, workingDir string) (*Session, error) {
	s.sessionPoolMutex.Lock()
	defer s.sessionPoolMutex.Unlock()

	// Check if session already exists
	if session, exists := sessionPool[sessionKey]; exists {
		// Check if session is still healthy
		if session.Status != SessionError && session.Status != SessionClosed {
			return session, nil
		}
		// Remove unhealthy session
		delete(sessionPool, sessionKey)
	}

	// Create new session
	session, err := s.client.CreateSession(ctx, workingDir)
	if err != nil {
		return nil, err
	}

	sessionPool[sessionKey] = session
	return session, nil
}

func (s *Service) getSessionByID(sessionID string) (*Session, error) {
	s.sessionPoolMutex.RLock()
	defer s.sessionPoolMutex.RUnlock()

	// Check all session pools
	pools := []map[string]*Session{
		s.issueAnalysisSessions,
		s.codeGenerationSessions,
		s.reviewSessions,
	}

	for _, pool := range pools {
		for _, session := range pool {
			if session.ID == sessionID {
				return session, nil
			}
		}
	}

	return nil, fmt.Errorf("session %s not found", sessionID)
}

func (s *Service) executeWithRetry(ctx context.Context, session *Session, prompt string) (string, error) {
	fmt.Printf("🔍 Debug: executeWithRetry starting (attempts: %d)\n", s.retryAttempts)
	var lastErr error

	for attempt := 0; attempt < s.retryAttempts; attempt++ {
		fmt.Printf("🔍 Debug: Attempt %d/%d\n", attempt+1, s.retryAttempts)
		
		if attempt > 0 {
			fmt.Printf("🔍 Debug: Waiting %v before retry...\n", s.retryDelay)
			select {
			case <-time.After(s.retryDelay):
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}

		// Execute Claude in non-interactive mode
		fmt.Println("🔍 Debug: Calling Claude in non-interactive mode...")
		response, err := s.executeClaudeNonInteractive(ctx, prompt, session.WorkingDir)
		if err != nil {
			fmt.Printf("❌ Claude execution failed on attempt %d/%d: %v\n", attempt+1, s.retryAttempts, err)
			lastErr = err
			
			// If it's the last attempt, don't continue to retry
			if attempt == s.retryAttempts-1 {
				fmt.Printf("❌ All retry attempts exhausted. Final error: %v\n", err)
			}
			continue
		}
		fmt.Printf("✅ Got response from Claude (%d chars)\n", len(response))

		return response, nil
	}

	return "", fmt.Errorf("command failed after %d attempts: %w", s.retryAttempts, lastErr)
}

// executeClaudeNonInteractive executes Claude in non-interactive mode with -p flag
func (s *Service) executeClaudeNonInteractive(ctx context.Context, prompt, workingDir string) (string, error) {
	// Get Claude command from client
	claudeCommand := s.client.claudeCommand
	fmt.Printf("🔍 Debug: Using Claude command: %s\n", claudeCommand)
	
	// Use the original context without additional timeout
	cmdCtx := ctx
	
	// Execute Claude command with timeout and proper error handling
	fmt.Printf("🔍 Debug: Executing Claude command: %s\n", claudeCommand)
	
	// Add timeout to prevent hanging
	timeoutCtx, cancel := context.WithTimeout(cmdCtx, 45*time.Second)
	defer cancel()
	
	// Build command: claude -p "prompt"
	var cmd *exec.Cmd
	if strings.Contains(claudeCommand, "/") {
		// Direct execution for explicit paths - pass prompt as separate argument
		// #nosec G204 - claudeCommand is from validated config, not user input
		cmd = exec.CommandContext(timeoutCtx, claudeCommand, "-p", prompt)
	} else {
		// Use shell for aliases or commands in PATH with proper quoting
		fullCommand := fmt.Sprintf("%s -p %q", claudeCommand, prompt)
		// #nosec G204 - claudeCommand is from validated config, prompt is escaped
		cmd = exec.CommandContext(timeoutCtx, "sh", "-c", fullCommand)
	}
	
	if workingDir != "" {
		cmd.Dir = workingDir
	}
	
	// Set environment variables for Claude
	cmd.Env = append(os.Environ(), 
		"CLAUDE_TIMEOUT=30s",
		"CLAUDE_OUTPUT_FORMAT=text",
		"CLAUDE_FAST_MODE=true",
		"CLAUDE_MAX_TOKENS=2048",
	)
	
	// Execute command and capture output
	fmt.Printf("🔍 Debug: Starting Claude execution...\n")
	if strings.Contains(claudeCommand, "/") {
		fmt.Printf("🔍 Debug: Executing command: %s -p \"<prompt>\"\n", claudeCommand)
		fmt.Printf("🔍 Debug: Working directory: %s\n", workingDir)
		// For direct execution, prompt is passed as separate arg (automatically quoted by exec)
		fmt.Printf("🔍 Debug: Command args: [\"%s\", \"-p\", \"<prompt_content>\"]\n", claudeCommand)
	} else {
		// Show the full command as it will be executed by shell
		fullCommand := fmt.Sprintf("%s -p %q", claudeCommand, prompt)
		fmt.Printf("🔍 Debug: Executing shell command: sh -c \"%s\"\n", fullCommand)
		fmt.Printf("🔍 Debug: Working directory: %s\n", workingDir)
		fmt.Printf("🔍 Debug: Command args: [\"sh\", \"-c\", \"%s\"]\n", fullCommand)
	}
	
	// Show prompt content if verbose debug is enabled
	if os.Getenv("CCAGENTS_VERBOSE_DEBUG") == "true" {
		separator := "=" + strings.Repeat("=", 78)
		fmt.Printf("🔍 Debug: Full Claude Prompt:\n")
		fmt.Printf("%s\n", separator)
		fmt.Printf("%s\n", prompt)
		fmt.Printf("%s\n", separator)
		fmt.Printf("🔍 Debug: Prompt length: %d characters\n", len(prompt))
	}
	output, err := cmd.Output()
	if err != nil {
		// If Claude execution fails, provide helpful error context
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Printf("❌ Claude Error: Command timed out after 45 seconds\n")
			return "", fmt.Errorf("claude command timed out after 45 seconds")
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			fmt.Printf("❌ Claude Error: Command failed with exit code %d\n", exitErr.ExitCode())
			fmt.Printf("❌ Claude Stderr: %s\n", stderr)
			fmt.Printf("❌ Claude Command: %s\n", claudeCommand)
			if workingDir != "" {
				fmt.Printf("❌ Claude Working Directory: %s\n", workingDir)
			}
			return "", fmt.Errorf("claude command failed with exit code %d: %s", exitErr.ExitCode(), stderr)
		}
		fmt.Printf("❌ Claude Error: Command execution failed: %v\n", err)
		fmt.Printf("❌ Claude Command: %s\n", claudeCommand)
		if workingDir != "" {
			fmt.Printf("❌ Claude Working Directory: %s\n", workingDir)
		}
		return "", fmt.Errorf("claude command execution failed: %w", err)
	}
	
	response := strings.TrimSpace(string(output))
	fmt.Printf("✅ Claude execution completed (%d chars)\n", len(response))
	
	if len(response) == 0 {
		return "", fmt.Errorf("claude returned empty response")
	}
	
	return response, nil
}

func (s *Service) waitForResponse(ctx context.Context, session *Session) (string, error) { //nolint:unused
	fmt.Printf("🔍 Debug: waitForResponse starting (timeout: %v)\n", s.defaultTimeout)
	var output []string
	var errors []string

	timeout := time.NewTimer(s.defaultTimeout)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("❌ Context canceled while waiting for response")
			return "", ctx.Err()

		case <-timeout.C:
			fmt.Printf("❌ Command timeout after %v\n", s.defaultTimeout)
			return "", fmt.Errorf("command timeout after %v", s.defaultTimeout)

		case msg := <-s.client.GetSessionOutput(session):
			fmt.Printf("📤 Got output: %s\n", msg.Content)
			output = append(output, msg.Content)

		case err := <-s.client.GetSessionErrors(session):
			fmt.Printf("📤 Got error: %v\n", err)
			errors = append(errors, err.Error())

		case status := <-s.client.GetSessionStatus(session):
			fmt.Printf("📤 Got status: %v\n", status)
			switch status {
			case SessionIdle:
				if len(errors) > 0 {
					fmt.Printf("❌ Command completed with errors: %v\n", errors)
					return "", fmt.Errorf("command failed: %s", strings.Join(errors, "; "))
				}
				fmt.Printf("✅ Command completed successfully with %d output lines\n", len(output))
				return strings.Join(output, "\n"), nil
			case SessionError, SessionClosed:
				fmt.Printf("❌ Session error/closed with errors: %v\n", errors)
				return "", fmt.Errorf("session error: %s", strings.Join(errors, "; "))
			}
		}
	}
}

func (s *Service) updateConversationHistory(sessionKey, prompt, response string) {
	s.historyMutex.Lock()
	defer s.historyMutex.Unlock()

	if s.conversationHistory[sessionKey] == nil {
		s.conversationHistory[sessionKey] = []ConversationEntry{}
	}

	// Add user message
	s.conversationHistory[sessionKey] = append(s.conversationHistory[sessionKey], ConversationEntry{
		Timestamp: time.Now(),
		Role:      RoleUser,
		Content:   prompt,
	})

	// Add assistant response
	s.conversationHistory[sessionKey] = append(s.conversationHistory[sessionKey], ConversationEntry{
		Timestamp: time.Now(),
		Role:      RoleAssistant,
		Content:   response,
	})
}

// InputHandler defines an interface for interactive commands
type InputHandler interface {
	HandlePrompt(prompt string) (string, error)
}

// Data structures for service operations

type IssueAnalysis struct {
	Summary          string
	Requirements     []string
	TechnicalDetails []string
	Complexity       ComplexityLevel
	EstimatedEffort  time.Duration
	Dependencies     []string
	RiskFactors      []string
	Recommendations  []string
}

type ComplexityLevel int

const (
	ComplexityLow ComplexityLevel = iota
	ComplexityMedium
	ComplexityHigh
	ComplexityCritical
)

type CodeGenerationRequest struct {
	Repository       string
	IssueNumber      int
	WorkingDirectory string
	Requirements     []string
	Context          *types.CodeContext
	Constraints      []string
	Framework        string
	Language         string
}

type CodeGenerationResult struct {
	Files         []GeneratedFile
	Tests         []GeneratedFile
	Documentation string
	Instructions  []string
	Warnings      []string
}

type GeneratedFile struct {
	Path        string
	Content     string
	Language    string
	Description string
}

type CodeReviewRequest struct {
	Repository       string
	Branch           string
	WorkingDirectory string
	Files            []string
	Context          *types.CodeContext
	ReviewType       ReviewType
}

type ReviewType int

const (
	ReviewSecurity ReviewType = iota
	ReviewPerformance
	ReviewMaintainability
	ReviewFull
)

type CodeReviewResult struct {
	Summary     string
	Issues      []ReviewIssue
	Suggestions []ReviewSuggestion
	Score       int
	Approved    bool
}

type ReviewIssue struct {
	File        string
	Line        int
	Type        IssueType
	Severity    IssueSeverity
	Description string
	Suggestion  string
}

type ReviewSuggestion struct {
	File        string
	Line        int
	Type        SuggestionType
	Description string
	Code        string
}

type IssueType int

const (
	IssueSecurity IssueType = iota
	IssuePerformance
	IssueBug
	IssueStyle
	IssueMaintainability
)

type IssueSeverity int

const (
	SeverityInfo IssueSeverity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

type SuggestionType int

const (
	SuggestionImprovement SuggestionType = iota
	SuggestionOptimization
	SuggestionRefactoring
	SuggestionBestPractice
)

type CommandResult struct {
	Command   string
	StartTime time.Time
	EndTime   time.Time
	Output    []string
	Errors    []string
	Success   bool
	Error     error
}
