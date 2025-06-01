package comments

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Status constants
const (
	statusFailed    = "failed"
	statusCompleted = "completed"
)

// Severity constants
const (
	severityCritical = "critical"
	statusUnknown    = "unknown"
)

// CommentHandler handles pull request comments and reviews
type CommentHandler struct {
	config    CommentHandlerConfig
	analyzer  *CommentAnalyzer
	responder *CommentResponder
	monitor   *CommentMonitor
}

// CommentHandlerConfig configures the comment handler
type CommentHandlerConfig struct {
	AutoRespond       bool
	ResponseDelay     time.Duration
	MaxResponseLength int
	EnableMentions    bool
	AlertOnFailure    bool
	TrackResolution   bool
	EscalationRules   []EscalationRule
	IgnoreUsers       []string
	RequiredApprovals int
}

// Comment represents a PR comment or review comment
type Comment struct {
	ID          int                `json:"id"`
	Type        CommentType        `json:"type"`
	Author      string             `json:"author"`
	Body        string             `json:"body"`
	File        string             `json:"file,omitempty"`
	Line        int                `json:"line,omitempty"`
	Position    int                `json:"position,omitempty"`
	Intent      CommentIntent      `json:"intent"`
	Priority    CommentPriority    `json:"priority"`
	Status      CommentStatus      `json:"status"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	ResolvedAt  *time.Time         `json:"resolved_at,omitempty"`
	Metadata    CommentMetadata    `json:"metadata"`
	Responses   []CommentResponse  `json:"responses"`
	Suggestions []CodeSuggestion   `json:"suggestions"`
	References  []CommentReference `json:"references"`
}

// CommentMetadata contains additional comment information
type CommentMetadata struct {
	Sentiment      float64         `json:"sentiment"`
	Confidence     float64         `json:"confidence"`
	Keywords       []string        `json:"keywords"`
	MentionedCode  []string        `json:"mentioned_code"`
	Complexity     ComplexityLevel `json:"complexity"`
	Urgency        UrgencyLevel    `json:"urgency"`
	ActionRequired bool            `json:"action_required"`
	ReviewRound    int             `json:"review_round"`
	ThreadID       string          `json:"thread_id"`
	ParentID       *int            `json:"parent_id,omitempty"`
}

// CommentResponse represents a response to a comment
type CommentResponse struct {
	ID        int              `json:"id"`
	Type      ResponseType     `json:"type"`
	Content   string           `json:"content"`
	Author    string           `json:"author"`
	CreatedAt time.Time        `json:"created_at"`
	Status    ResponseStatus   `json:"status"`
	Actions   []ResponseAction `json:"actions"`
}

// CodeSuggestion represents a code change suggestion
type CodeSuggestion struct {
	File        string     `json:"file"`
	StartLine   int        `json:"start_line"`
	EndLine     int        `json:"end_line"`
	OldCode     string     `json:"old_code"`
	NewCode     string     `json:"new_code"`
	Description string     `json:"description"`
	Confidence  float64    `json:"confidence"`
	Applied     bool       `json:"applied"`
	AppliedAt   *time.Time `json:"applied_at,omitempty"`
}

// CommentReference represents references to issues, PRs, or commits
type CommentReference struct {
	Type   ReferenceType `json:"type"`
	Target string        `json:"target"`
	URL    string        `json:"url"`
}

// EscalationRule defines when to escalate comments
type EscalationRule struct {
	Name       string        `json:"name"`
	Condition  string        `json:"condition"`
	Threshold  int           `json:"threshold"`
	Action     string        `json:"action"`
	Recipients []string      `json:"recipients"`
	Delay      time.Duration `json:"delay"`
	MaxRetries int           `json:"max_retries"`
}

// ResponseAction represents an action taken in response to a comment
type ResponseAction struct {
	Type        ActionType `json:"type"`
	Description string     `json:"description"`
	File        string     `json:"file,omitempty"`
	Command     string     `json:"command,omitempty"`
	Status      string     `json:"status"`
	Result      string     `json:"result,omitempty"`
	ExecutedAt  time.Time  `json:"executed_at"`
}

// Enums

type CommentType int

const (
	CommentTypeReview CommentType = iota
	CommentTypeInline
	CommentTypeGeneral
	CommentTypeApproval
	CommentTypeDismissal
	CommentTypeRequest
)

type CommentIntent int

const (
	CommentIntentQuestion CommentIntent = iota
	CommentIntentSuggestion
	CommentIntentRequest
	CommentIntentApproval
	CommentIntentBlocking
	CommentIntentPraise
	CommentIntentClarification
	CommentIntentConcern
)

type CommentPriority int

const (
	CommentPriorityLow CommentPriority = iota
	CommentPriorityMedium
	CommentPriorityHigh
	CommentPriorityCritical
	CommentPriorityBlocking
)

type CommentStatus int

const (
	CommentStatusPending CommentStatus = iota
	CommentStatusAcknowledged
	CommentStatusInProgress
	CommentStatusResolved
	CommentStatusDismissed
	CommentStatusEscalated
)

type ResponseType int

const (
	ResponseTypeAcknowledgment ResponseType = iota
	ResponseTypeClarification
	ResponseTypeImplementation
	ResponseTypeExplanation
	ResponseTypeCounterproposal
	ResponseTypeEscalation
)

type ResponseStatus int

const (
	ResponseStatusDraft ResponseStatus = iota
	ResponseStatusSent
	ResponseStatusDelivered
	ResponseStatusRead
	ResponseStatusActioned
)

type ActionType int

const (
	ActionTypeCodeChange ActionType = iota
	ActionTypeFileModify
	ActionTypeTestAdd
	ActionTypeDocUpdate
	ActionTypeRefactor
	ActionTypeInvestigate
	ActionTypeDiscuss
)

type ComplexityLevel int

const (
	ComplexityLevelSimple ComplexityLevel = iota
	ComplexityLevelModerate
	ComplexityLevelComplex
	ComplexityLevelVeryComplex
)

type UrgencyLevel int

const (
	UrgencyLevelLow UrgencyLevel = iota
	UrgencyLevelMedium
	UrgencyLevelHigh
	UrgencyLevelUrgent
)

type ReferenceType int

const (
	ReferenceTypeIssue ReferenceType = iota
	ReferenceTypePR
	ReferenceTypeCommit
	ReferenceTypeFile
	ReferenceTypeUser
)

// GitHubClient interface for GitHub comment operations
type GitHubClient interface {
	ListComments(ctx context.Context, prNumber int) ([]*Comment, error)
	GetComment(ctx context.Context, commentID int) (*Comment, error)
	CreateComment(ctx context.Context, prNumber int, body string) (*Comment, error)
	UpdateComment(ctx context.Context, commentID int, body string) (*Comment, error)
	ReplyToComment(ctx context.Context, commentID int, body string) (*Comment, error)
	ResolveComment(ctx context.Context, commentID int) error
	DismissReview(ctx context.Context, prNumber int, reviewID int, message string) error
}

// NewCommentHandler creates a new comment handler
func NewCommentHandler(config CommentHandlerConfig, client GitHubClient) *CommentHandler {
	// Set defaults
	if config.ResponseDelay == 0 {
		config.ResponseDelay = time.Minute * 5
	}
	if config.MaxResponseLength == 0 {
		config.MaxResponseLength = 2000
	}
	if config.RequiredApprovals == 0 {
		config.RequiredApprovals = 1
	}

	ch := &CommentHandler{
		config: config,
	}

	// Initialize sub-components
	ch.analyzer = NewCommentAnalyzer(CommentAnalyzerConfig{
		EnableSentimentAnalysis: true,
		EnableIntentDetection:   true,
		EnableKeywordExtraction: true,
		ConfidenceThreshold:     0.7,
	})

	ch.responder = NewCommentResponder(CommentResponderConfig{
		MaxLength:         config.MaxResponseLength,
		EnableSuggestions: true,
		EnableMentions:    config.EnableMentions,
		ResponseDelay:     config.ResponseDelay,
	})

	ch.monitor = NewCommentMonitor(CommentMonitorConfig{
		PollInterval:    time.Minute * 2,
		TrackResolution: config.TrackResolution,
		EscalationRules: config.EscalationRules,
		AlertOnFailure:  config.AlertOnFailure,
	})

	return ch
}

// HandleComments processes comments for a pull request
func (ch *CommentHandler) HandleComments(ctx context.Context, prNumber int, client GitHubClient) error {
	// Get all comments
	comments, err := client.ListComments(ctx, prNumber)
	if err != nil {
		return fmt.Errorf("failed to list comments: %w", err)
	}

	// Process each comment
	for _, comment := range comments {
		if ch.shouldIgnoreComment(comment) {
			continue
		}

		// Analyze comment
		if err := ch.analyzer.AnalyzeComment(ctx, comment); err != nil {
			fmt.Printf("Warning: failed to analyze comment %d: %v\n", comment.ID, err)
			continue
		}

		// Handle based on intent and priority
		if err := ch.handleSingleComment(ctx, comment, client); err != nil {
			fmt.Printf("Warning: failed to handle comment %d: %v\n", comment.ID, err)
			continue
		}
	}

	return nil
}

// MonitorComments continuously monitors for new comments
func (ch *CommentHandler) MonitorComments(ctx context.Context, prNumber int, client GitHubClient) error {
	return ch.monitor.StartMonitoring(ctx, prNumber, client, ch.handleNewComment)
}

// handleSingleComment processes an individual comment
func (ch *CommentHandler) handleSingleComment(ctx context.Context, comment *Comment, client GitHubClient) error {
	// Skip if already handled
	if comment.Status == CommentStatusResolved || comment.Status == CommentStatusDismissed {
		return nil
	}

	// Mark as acknowledged
	comment.Status = CommentStatusAcknowledged

	// Handle based on intent
	switch comment.Intent {
	case CommentIntentQuestion:
		return ch.handleQuestion(ctx, comment, client)
	case CommentIntentSuggestion:
		return ch.handleSuggestion(ctx, comment, client)
	case CommentIntentRequest:
		return ch.handleRequest(ctx, comment, client)
	case CommentIntentBlocking:
		return ch.handleBlockingComment(ctx, comment, client)
	case CommentIntentApproval:
		return ch.handleApproval(ctx, comment, client)
	case CommentIntentPraise:
		return ch.handlePraise(ctx, comment, client)
	default:
		return ch.handleGenericComment(ctx, comment, client)
	}
}

// handleQuestion responds to questions in comments
func (ch *CommentHandler) handleQuestion(ctx context.Context, comment *Comment, client GitHubClient) error {
	// Generate response
	response, err := ch.responder.GenerateQuestionResponse(ctx, comment)
	if err != nil {
		return fmt.Errorf("failed to generate question response: %w", err)
	}

	// Send response
	if ch.config.AutoRespond {
		select {
		case <-time.After(ch.config.ResponseDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
		_, err := client.ReplyToComment(ctx, comment.ID, response.Content)
		if err != nil {
			return fmt.Errorf("failed to send response: %w", err)
		}
		comment.Responses = append(comment.Responses, *response)
	}

	return nil
}

// handleSuggestion processes code suggestions
func (ch *CommentHandler) handleSuggestion(ctx context.Context, comment *Comment, client GitHubClient) error {
	// Extract suggestions
	suggestions := ch.analyzer.ExtractCodeSuggestions(comment)
	comment.Suggestions = suggestions

	// Evaluate suggestions
	for i, suggestion := range suggestions {
		if suggestion.Confidence > 0.8 {
			// High confidence - auto-apply if enabled
			if ch.config.AutoRespond {
				if err := ch.applySuggestion(ctx, &suggestions[i]); err != nil {
					fmt.Printf("Warning: failed to apply suggestion: %v\n", err)
					continue
				}
				suggestions[i].Applied = true
				now := time.Now()
				suggestions[i].AppliedAt = &now
			}
		}
	}

	// Generate response
	response, err := ch.responder.GenerateSuggestionResponse(ctx, comment, suggestions)
	if err != nil {
		return fmt.Errorf("failed to generate suggestion response: %w", err)
	}

	// Send response
	if ch.config.AutoRespond {
		select {
		case <-time.After(ch.config.ResponseDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
		_, err := client.ReplyToComment(ctx, comment.ID, response.Content)
		if err != nil {
			return fmt.Errorf("failed to send response: %w", err)
		}
		comment.Responses = append(comment.Responses, *response)
	}

	return nil
}

// handleRequest processes change requests
func (ch *CommentHandler) handleRequest(ctx context.Context, comment *Comment, client GitHubClient) error {
	comment.Status = CommentStatusInProgress

	// Analyze the request
	actions := ch.analyzer.ExtractActionItems(comment)

	// Execute actions
	var completedActions []ResponseAction
	for _, action := range actions {
		if err := ch.executeAction(ctx, &action); err != nil {
			fmt.Printf("Warning: failed to execute action %s: %v\n", action.Description, err)
			action.Status = statusFailed
			action.Result = err.Error()
		} else {
			action.Status = statusCompleted
			completedActions = append(completedActions, action)
		}
		action.ExecutedAt = time.Now()
	}

	// Generate response
	response, err := ch.responder.GenerateRequestResponse(ctx, comment, completedActions)
	if err != nil {
		return fmt.Errorf("failed to generate request response: %w", err)
	}

	// Send response
	if ch.config.AutoRespond {
		select {
		case <-time.After(ch.config.ResponseDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
		_, err := client.ReplyToComment(ctx, comment.ID, response.Content)
		if err != nil {
			return fmt.Errorf("failed to send response: %w", err)
		}
		comment.Responses = append(comment.Responses, *response)
	}

	// Mark as resolved if all actions completed
	if len(completedActions) == len(actions) {
		comment.Status = CommentStatusResolved
		now := time.Now()
		comment.ResolvedAt = &now
	}

	return nil
}

// handleBlockingComment handles blocking comments that prevent merge
func (ch *CommentHandler) handleBlockingComment(ctx context.Context, comment *Comment, client GitHubClient) error {
	// Mark as high priority
	comment.Priority = CommentPriorityBlocking

	// Generate detailed response
	response, err := ch.responder.GenerateBlockingResponse(ctx, comment)
	if err != nil {
		return fmt.Errorf("failed to generate blocking response: %w", err)
	}

	// Send response immediately
	_, err = client.ReplyToComment(ctx, comment.ID, response.Content)
	if err != nil {
		return fmt.Errorf("failed to send response: %w", err)
	}
	comment.Responses = append(comment.Responses, *response)

	// Escalate if configured
	if ch.shouldEscalate(comment) {
		return ch.escalateComment(ctx, comment, client)
	}

	return nil
}

// handleApproval processes approval comments
func (ch *CommentHandler) handleApproval(ctx context.Context, comment *Comment, client GitHubClient) error {
	comment.Status = CommentStatusResolved
	now := time.Now()
	comment.ResolvedAt = &now

	// Send acknowledgment
	if ch.config.AutoRespond {
		response, err := ch.responder.GenerateApprovalResponse(ctx, comment)
		if err == nil {
			_, err = client.ReplyToComment(ctx, comment.ID, response.Content)
			if err != nil {
				fmt.Printf("Warning: failed to send approval response: %v\n", err)
			} else {
				comment.Responses = append(comment.Responses, *response)
			}
		}
	}

	return nil
}

// handlePraise responds to positive feedback
func (ch *CommentHandler) handlePraise(ctx context.Context, comment *Comment, client GitHubClient) error {
	comment.Status = CommentStatusResolved
	now := time.Now()
	comment.ResolvedAt = &now

	// Send thank you response
	if ch.config.AutoRespond {
		response, err := ch.responder.GeneratePraiseResponse(ctx, comment)
		if err == nil {
			select {
			case <-time.After(ch.config.ResponseDelay):
			case <-ctx.Done():
				return ctx.Err()
			}
			_, err = client.ReplyToComment(ctx, comment.ID, response.Content)
			if err != nil {
				fmt.Printf("Warning: failed to send praise response: %v\n", err)
			} else {
				comment.Responses = append(comment.Responses, *response)
			}
		}
	}

	return nil
}

// handleGenericComment handles comments with unclear intent
func (ch *CommentHandler) handleGenericComment(ctx context.Context, comment *Comment, client GitHubClient) error {
	// Generate clarification request
	response, err := ch.responder.GenerateClarificationResponse(ctx, comment)
	if err != nil {
		return fmt.Errorf("failed to generate clarification response: %w", err)
	}

	// Send response
	if ch.config.AutoRespond {
		select {
		case <-time.After(ch.config.ResponseDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
		_, err := client.ReplyToComment(ctx, comment.ID, response.Content)
		if err != nil {
			return fmt.Errorf("failed to send response: %w", err)
		}
		comment.Responses = append(comment.Responses, *response)
	}

	return nil
}

// Helper methods

func (ch *CommentHandler) shouldIgnoreComment(comment *Comment) bool {
	// Check ignore list
	for _, user := range ch.config.IgnoreUsers {
		if comment.Author == user {
			return true
		}
	}

	// Ignore bot comments
	if strings.Contains(strings.ToLower(comment.Author), "bot") {
		return true
	}

	// Ignore very old comments
	if time.Since(comment.CreatedAt) > time.Hour*24*7 { // 1 week
		return true
	}

	return false
}

func (ch *CommentHandler) handleNewComment(ctx context.Context, comment *Comment, client GitHubClient) error {
	// Process new comment immediately
	return ch.handleSingleComment(ctx, comment, client)
}

func (ch *CommentHandler) applySuggestion(ctx context.Context, suggestion *CodeSuggestion) error {
	// This would apply the code suggestion to the file
	// Implementation depends on the specific suggestion type
	fmt.Printf("Applying suggestion to %s:%d-%d\n",
		suggestion.File, suggestion.StartLine, suggestion.EndLine)
	return nil // Placeholder
}

func (ch *CommentHandler) executeAction(ctx context.Context, action *ResponseAction) error {
	// Execute the requested action
	switch action.Type {
	case ActionTypeCodeChange:
		return ch.executeCodeChange(ctx, action)
	case ActionTypeFileModify:
		return ch.executeFileModify(ctx, action)
	case ActionTypeTestAdd:
		return ch.executeTestAdd(ctx, action)
	case ActionTypeDocUpdate:
		return ch.executeDocUpdate(ctx, action)
	default:
		return fmt.Errorf("unsupported action type: %v", action.Type)
	}
}

func (ch *CommentHandler) executeCodeChange(ctx context.Context, action *ResponseAction) error {
	// Execute code change
	fmt.Printf("Executing code change: %s\n", action.Description)
	return nil // Placeholder
}

func (ch *CommentHandler) executeFileModify(ctx context.Context, action *ResponseAction) error {
	// Execute file modification
	fmt.Printf("Executing file modification: %s\n", action.Description)
	return nil // Placeholder
}

func (ch *CommentHandler) executeTestAdd(ctx context.Context, action *ResponseAction) error {
	// Execute test addition
	fmt.Printf("Executing test addition: %s\n", action.Description)
	return nil // Placeholder
}

func (ch *CommentHandler) executeDocUpdate(ctx context.Context, action *ResponseAction) error {
	// Execute documentation update
	fmt.Printf("Executing documentation update: %s\n", action.Description)
	return nil // Placeholder
}

func (ch *CommentHandler) shouldEscalate(comment *Comment) bool {
	// Check escalation rules
	for _, rule := range ch.config.EscalationRules {
		if ch.matchesEscalationRule(comment, rule) {
			return true
		}
	}
	return false
}

func (ch *CommentHandler) matchesEscalationRule(comment *Comment, rule EscalationRule) bool {
	switch rule.Condition {
	case keywordBlocking:
		return comment.Priority == CommentPriorityBlocking
	case severityCritical:
		return comment.Priority == CommentPriorityCritical
	case "unresolved_time":
		return time.Since(comment.CreatedAt) > rule.Delay
	default:
		return false
	}
}

func (ch *CommentHandler) escalateComment(ctx context.Context, comment *Comment, client GitHubClient) error {
	comment.Status = CommentStatusEscalated

	// Find matching escalation rule
	for _, rule := range ch.config.EscalationRules {
		if ch.matchesEscalationRule(comment, rule) {
			// Execute escalation action
			return ch.executeEscalation(ctx, comment, rule, client)
		}
	}

	return nil
}

func (ch *CommentHandler) executeEscalation(ctx context.Context, comment *Comment, rule EscalationRule, client GitHubClient) error {
	// Create escalation comment
	escalationMsg := fmt.Sprintf("⚠️ **Escalation**: Comment requires attention\n\n"+
		"Original comment by @%s needs resolution:\n> %s\n\n"+
		"Escalation rule: %s\n"+
		"Recipients: %s",
		comment.Author,
		comment.Body,
		rule.Name,
		strings.Join(rule.Recipients, ", "))

	_, err := client.CreateComment(ctx, 0, escalationMsg) // Would need PR number
	return err
}

// String methods for enums

func (ct CommentType) String() string {
	switch ct {
	case CommentTypeReview:
		return "review"
	case CommentTypeInline:
		return "inline"
	case CommentTypeGeneral:
		return "general"
	case CommentTypeApproval:
		return "approval"
	case CommentTypeDismissal:
		return "dismissal"
	case CommentTypeRequest:
		return "request"
	default:
		return statusUnknown
	}
}

func (ci CommentIntent) String() string {
	switch ci {
	case CommentIntentQuestion:
		return "question"
	case CommentIntentSuggestion:
		return "suggestion"
	case CommentIntentRequest:
		return "request"
	case CommentIntentApproval:
		return "approval"
	case CommentIntentBlocking:
		return keywordBlocking
	case CommentIntentPraise:
		return "praise"
	case CommentIntentClarification:
		return "clarification"
	case CommentIntentConcern:
		return "concern"
	default:
		return statusUnknown
	}
}

func (cp CommentPriority) String() string {
	switch cp {
	case CommentPriorityLow:
		return "low"
	case CommentPriorityMedium:
		return "medium"
	case CommentPriorityHigh:
		return "high"
	case CommentPriorityCritical:
		return severityCritical
	case CommentPriorityBlocking:
		return keywordBlocking
	default:
		return statusUnknown
	}
}

func (cs CommentStatus) String() string {
	switch cs {
	case CommentStatusPending:
		return "pending"
	case CommentStatusAcknowledged:
		return "acknowledged"
	case CommentStatusInProgress:
		return "in_progress"
	case CommentStatusResolved:
		return "resolved"
	case CommentStatusDismissed:
		return "dismissed"
	case CommentStatusEscalated:
		return "escalated"
	default:
		return statusUnknown
	}
}
