package comments

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// CommentResponder generates responses to comments
type CommentResponder struct {
	config CommentResponderConfig
}

// CommentResponderConfig configures the comment responder
type CommentResponderConfig struct {
	MaxLength         int
	EnableSuggestions bool
	EnableMentions    bool
	ResponseDelay     time.Duration
	Templates         map[string]string
	CustomResponses   map[string][]string
	Personality       string
}

// NewCommentResponder creates a new comment responder
func NewCommentResponder(config CommentResponderConfig) *CommentResponder {
	if config.MaxLength == 0 {
		config.MaxLength = 2000
	}
	if config.ResponseDelay == 0 {
		config.ResponseDelay = time.Millisecond * 2 // Fast for tests
	}
	if config.Personality == "" {
		config.Personality = "helpful"
	}

	// Set default templates
	if config.Templates == nil {
		config.Templates = getDefaultTemplates()
	}

	return &CommentResponder{
		config: config,
	}
}

// GenerateQuestionResponse generates a response to a question
func (cr *CommentResponder) GenerateQuestionResponse(ctx context.Context, comment *Comment) (*CommentResponse, error) {
	var content strings.Builder

	// Acknowledge the question
	content.WriteString(fmt.Sprintf("Thanks for the question, @%s! ", comment.Author))

	// Analyze what kind of question it is
	questionType := cr.analyzeQuestionType(comment.Body)

	switch questionType {
	case "implementation":
		content.WriteString("I'll explain the implementation approach:\n\n")
		content.WriteString(cr.generateImplementationExplanation(comment))
	case "reasoning":
		content.WriteString("Here's the reasoning behind this change:\n\n")
		content.WriteString(cr.generateReasoningExplanation(comment))
	case "usage":
		content.WriteString("Here's how to use this:\n\n")
		content.WriteString(cr.generateUsageExplanation(comment))
	case "alternatives":
		content.WriteString("Let me explain the alternatives considered:\n\n")
		content.WriteString(cr.generateAlternativesExplanation(comment))
	default:
		content.WriteString("Let me provide some clarification:\n\n")
		content.WriteString(cr.generateGenericExplanation(comment))
	}

	// Add follow-up invitation
	content.WriteString("\n\nFeel free to ask if you need any additional clarification!")

	return &CommentResponse{
		Type:      ResponseTypeClarification,
		Content:   cr.truncateIfNeeded(content.String()),
		Author:    "ccagents-ai",
		CreatedAt: time.Now(),
		Status:    ResponseStatusDraft,
	}, nil
}

// GenerateSuggestionResponse generates a response to suggestions
func (cr *CommentResponder) GenerateSuggestionResponse(ctx context.Context, comment *Comment, suggestions []CodeSuggestion) (*CommentResponse, error) {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("Thank you for the suggestion, @%s!\n\n", comment.Author))

	if len(suggestions) == 0 {
		content.WriteString("I've noted your suggestion and will consider it for future improvements.")
	} else {
		appliedCount := 0
		for _, suggestion := range suggestions {
			if suggestion.Applied {
				appliedCount++
			}
		}

		if appliedCount > 0 {
			content.WriteString(fmt.Sprintf("✅ I've applied %d of your suggestions:\n\n", appliedCount))
			for _, suggestion := range suggestions {
				if suggestion.Applied {
					content.WriteString(fmt.Sprintf("- %s (confidence: %.1f%%)\n",
						suggestion.Description, suggestion.Confidence*100))
				}
			}
		}

		remainingCount := len(suggestions) - appliedCount
		if remainingCount > 0 {
			content.WriteString(fmt.Sprintf("\n🔍 %d suggestions need manual review:\n\n", remainingCount))
			for _, suggestion := range suggestions {
				if !suggestion.Applied {
					content.WriteString(fmt.Sprintf("- %s (confidence: %.1f%%)\n",
						suggestion.Description, suggestion.Confidence*100))
				}
			}
		}
	}

	return &CommentResponse{
		Type:      ResponseTypeImplementation,
		Content:   cr.truncateIfNeeded(content.String()),
		Author:    "ccagents-ai",
		CreatedAt: time.Now(),
		Status:    ResponseStatusDraft,
	}, nil
}

// GenerateRequestResponse generates a response to change requests
func (cr *CommentResponder) GenerateRequestResponse(ctx context.Context, comment *Comment, actions []ResponseAction) (*CommentResponse, error) {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("@%s, I'm working on your request.\n\n", comment.Author))

	if len(actions) == 0 {
		content.WriteString("I'll address your request and update this PR accordingly.")
	} else {
		completedActions := 0
		for _, action := range actions {
			if action.Status == "completed" {
				completedActions++
			}
		}

		if completedActions > 0 {
			content.WriteString(fmt.Sprintf("✅ **Completed Actions** (%d/%d):\n", completedActions, len(actions)))
			for _, action := range actions {
				if action.Status == "completed" {
					content.WriteString(fmt.Sprintf("- %s\n", action.Description))
				}
			}
		}

		failedActions := 0
		for _, action := range actions {
			if action.Status == statusFailed {
				failedActions++
			}
		}

		if failedActions > 0 {
			content.WriteString(fmt.Sprintf("\n❌ **Failed Actions** (%d):\n", failedActions))
			for _, action := range actions {
				if action.Status == statusFailed {
					content.WriteString(fmt.Sprintf("- %s: %s\n", action.Description, action.Result))
				}
			}
		}

		pendingActions := len(actions) - completedActions - failedActions
		if pendingActions > 0 {
			content.WriteString(fmt.Sprintf("\n⏳ **Pending Actions** (%d):\n", pendingActions))
			for _, action := range actions {
				if action.Status == "pending" {
					content.WriteString(fmt.Sprintf("- %s\n", action.Description))
				}
			}
		}
	}

	content.WriteString("\nI'll notify you when all changes are complete.")

	return &CommentResponse{
		Type:      ResponseTypeImplementation,
		Content:   cr.truncateIfNeeded(content.String()),
		Author:    "ccagents-ai",
		CreatedAt: time.Now(),
		Status:    ResponseStatusDraft,
	}, nil
}

// GenerateBlockingResponse generates a response to blocking comments
func (cr *CommentResponder) GenerateBlockingResponse(ctx context.Context, comment *Comment) (*CommentResponse, error) {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("🚨 **Urgent**: @%s has identified a blocking issue.\n\n", comment.Author))
	content.WriteString("I'm prioritizing this feedback and will address it immediately.\n\n")

	// Extract the specific concern
	concern := cr.extractMainConcern(comment.Body)
	if concern != "" {
		content.WriteString(fmt.Sprintf("**Issue**: %s\n\n", concern))
	}

	content.WriteString("**Next Steps**:\n")
	content.WriteString("1. Investigating the reported issue\n")
	content.WriteString("2. Implementing necessary fixes\n")
	content.WriteString("3. Running comprehensive tests\n")
	content.WriteString("4. Requesting re-review\n\n")

	content.WriteString("I'll update this comment with progress and notify you when resolved.")

	return &CommentResponse{
		Type:      ResponseTypeEscalation,
		Content:   cr.truncateIfNeeded(content.String()),
		Author:    "ccagents-ai",
		CreatedAt: time.Now(),
		Status:    ResponseStatusDraft,
	}, nil
}

// GenerateApprovalResponse generates a response to approval comments
func (cr *CommentResponder) GenerateApprovalResponse(ctx context.Context, comment *Comment) (*CommentResponse, error) {
	responses := []string{
		"Thank you for the approval! 🎉",
		"Appreciate the review and approval! ✅",
		"Thanks for the feedback and approval! 👍",
		"Great! Thank you for reviewing and approving! 🚀",
	}

	baseResponse := responses[0] // Could randomize in the future

	content := fmt.Sprintf("%s @%s\n\nI'll proceed with merging once all checks pass.",
		baseResponse, comment.Author)

	return &CommentResponse{
		Type:      ResponseTypeAcknowledgment,
		Content:   content,
		Author:    "ccagents-ai",
		CreatedAt: time.Now(),
		Status:    ResponseStatusDraft,
	}, nil
}

// GeneratePraiseResponse generates a response to positive feedback
func (cr *CommentResponder) GeneratePraiseResponse(ctx context.Context, comment *Comment) (*CommentResponse, error) {
	responses := []string{
		"Thank you for the kind words! 😊",
		"Appreciate the positive feedback! 🙏",
		"Thanks! Always happy to deliver quality code! 💯",
		"Thank you! Glad you found it helpful! ✨",
	}

	baseResponse := responses[0] // Could randomize in the future

	content := fmt.Sprintf("%s @%s", baseResponse, comment.Author)

	return &CommentResponse{
		Type:      ResponseTypeAcknowledgment,
		Content:   content,
		Author:    "ccagents-ai",
		CreatedAt: time.Now(),
		Status:    ResponseStatusDraft,
	}, nil
}

// GenerateClarificationResponse generates a response asking for clarification
func (cr *CommentResponder) GenerateClarificationResponse(ctx context.Context, comment *Comment) (*CommentResponse, error) {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("Hi @%s! ", comment.Author))

	// Analyze what might need clarification
	if cr.isVague(comment.Body) {
		content.WriteString("Could you provide more details about what you'd like me to address? ")
	} else if cr.hasMultiplePoints(comment.Body) {
		content.WriteString("I see several points in your comment. Could you help me prioritize which ones to address first? ")
	} else {
		content.WriteString("I want to make sure I understand your comment correctly. ")
	}

	content.WriteString("Any additional context would be helpful!")

	return &CommentResponse{
		Type:      ResponseTypeClarification,
		Content:   content.String(),
		Author:    "ccagents-ai",
		CreatedAt: time.Now(),
		Status:    ResponseStatusDraft,
	}, nil
}

// Helper methods

func (cr *CommentResponder) analyzeQuestionType(body string) string {
	body = strings.ToLower(body)

	if strings.Contains(body, "how") && (strings.Contains(body, "work") || strings.Contains(body, "implement")) {
		return "implementation"
	}
	if strings.Contains(body, "why") {
		return "reasoning"
	}
	if strings.Contains(body, "how") && (strings.Contains(body, "use") || strings.Contains(body, "call")) {
		return "usage"
	}
	if strings.Contains(body, "alternative") || strings.Contains(body, "option") {
		return "alternatives"
	}

	return "general"
}

func (cr *CommentResponder) generateImplementationExplanation(comment *Comment) string {
	var explanation strings.Builder

	explanation.WriteString("The implementation follows these key principles:\n\n")

	// Extract mentioned code or files
	if len(comment.Metadata.MentionedCode) > 0 {
		explanation.WriteString("**Code Context**:\n")
		for _, code := range comment.Metadata.MentionedCode {
			explanation.WriteString(fmt.Sprintf("- `%s`\n", code))
		}
		explanation.WriteString("\n")
	}

	explanation.WriteString("**Approach**:\n")
	explanation.WriteString("1. Follows established patterns in the codebase\n")
	explanation.WriteString("2. Maintains compatibility with existing functionality\n")
	explanation.WriteString("3. Includes appropriate error handling\n")
	explanation.WriteString("4. Adds comprehensive tests\n")

	return explanation.String()
}

func (cr *CommentResponder) generateReasoningExplanation(comment *Comment) string {
	return "This approach was chosen because:\n\n" +
		"1. **Performance**: Optimizes for common use cases\n" +
		"2. **Maintainability**: Follows existing code patterns\n" +
		"3. **Reliability**: Includes proper error handling\n" +
		"4. **Testability**: Designed with testing in mind\n"
}

func (cr *CommentResponder) generateUsageExplanation(comment *Comment) string {
	return "Here's how to use this functionality:\n\n" +
		"```go\n" +
		"// Example usage will be provided based on the specific implementation\n" +
		"// This would be customized based on the actual code changes\n" +
		"```\n\n" +
		"The API is designed to be intuitive and follows Go conventions."
}

func (cr *CommentResponder) generateAlternativesExplanation(comment *Comment) string {
	return "I considered several alternatives:\n\n" +
		"1. **Current approach**: Chosen for its balance of simplicity and performance\n" +
		"2. **Alternative 1**: Would have been more complex but slightly faster\n" +
		"3. **Alternative 2**: Simpler but with potential scalability concerns\n\n" +
		"The current implementation provides the best trade-off for this use case."
}

func (cr *CommentResponder) generateGenericExplanation(comment *Comment) string {
	if len(comment.Metadata.Keywords) > 0 {
		return fmt.Sprintf("Based on your question about %s, here's the relevant information:\n\n"+
			"This implementation addresses the specific requirements while maintaining "+
			"code quality and following best practices.",
			strings.Join(comment.Metadata.Keywords[:min(3, len(comment.Metadata.Keywords))], ", "))
	}

	return "I'll provide the relevant details about this implementation and its rationale."
}

func (cr *CommentResponder) extractMainConcern(body string) string {
	// Simple extraction of the main concern
	sentences := strings.Split(body, ".")
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 20 && (strings.Contains(strings.ToLower(sentence), "issue") ||
			strings.Contains(strings.ToLower(sentence), "problem") ||
			strings.Contains(strings.ToLower(sentence), "concern")) {
			return sentence
		}
	}

	// Return first substantial sentence if no specific concern found
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 20 {
			return sentence
		}
	}

	return ""
}

func (cr *CommentResponder) isVague(body string) bool {
	body = strings.ToLower(body)
	vagueIndicators := []string{
		"this", "that", "it", "here", "there", "something", "anything", "everything",
	}

	vagueCount := 0
	words := strings.Fields(body)

	for _, word := range words {
		for _, indicator := range vagueIndicators {
			if word == indicator {
				vagueCount++
				break
			}
		}
	}

	// If more than 20% of words are vague indicators, consider it vague
	return float64(vagueCount)/float64(len(words)) > 0.2
}

func (cr *CommentResponder) hasMultiplePoints(body string) bool {
	// Count sentence endings, bullet points, and numbered lists
	points := 0

	points += strings.Count(body, ".")
	points += strings.Count(body, "?")
	points += strings.Count(body, "!")
	points += strings.Count(body, "- ")
	points += strings.Count(body, "* ")

	// Look for numbered lists
	for i := 1; i <= 9; i++ {
		points += strings.Count(body, fmt.Sprintf("%d. ", i))
		points += strings.Count(body, fmt.Sprintf("%d) ", i))
	}

	return points > 3
}

func (cr *CommentResponder) truncateIfNeeded(content string) string {
	if len(content) <= cr.config.MaxLength {
		return content
	}

	// Find a good truncation point
	truncateAt := cr.config.MaxLength - 100
	for i := truncateAt; i > truncateAt-100 && i > 0; i-- {
		if content[i] == '\n' || content[i] == '.' {
			truncateAt = i
			break
		}
	}

	truncated := content[:truncateAt]
	truncated += "\n\n*[Response truncated due to length. Please let me know if you need more details.]*"

	return truncated
}

func getDefaultTemplates() map[string]string {
	return map[string]string{
		"acknowledgment": "Thank you for the feedback, @{{.Author}}!",
		"clarification":  "Could you provide more details about {{.Topic}}?",
		"implementation": "I'll implement the following changes: {{.Changes}}",
		"explanation":    "Here's how this works: {{.Explanation}}",
		"approval":       "Thank you for the approval! 🎉",
		"praise":         "Appreciate the positive feedback! 🙏",
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
