// Package comments provides comment analysis and management functionality for ccAgents
package comments

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Keyword constants
const (
	keywordBlocking = "blocking"
)

// CommentAnalyzer analyzes comment content and intent
type CommentAnalyzer struct {
	config CommentAnalyzerConfig
}

// CommentAnalyzerConfig configures the comment analyzer
type CommentAnalyzerConfig struct {
	EnableSentimentAnalysis bool
	EnableIntentDetection   bool
	EnableKeywordExtraction bool
	ConfidenceThreshold     float64
	CustomPatterns          []IntentPattern
	LanguageModel           string
}

// IntentPattern defines patterns for intent detection
type IntentPattern struct {
	Intent   CommentIntent `json:"intent"`
	Patterns []string      `json:"patterns"`
	Keywords []string      `json:"keywords"`
	Weight   float64       `json:"weight"`
	Context  string        `json:"context"`
}

// NewCommentAnalyzer creates a new comment analyzer
func NewCommentAnalyzer(config CommentAnalyzerConfig) *CommentAnalyzer {
	if config.ConfidenceThreshold == 0 {
		config.ConfidenceThreshold = 0.7
	}

	return &CommentAnalyzer{
		config: config,
	}
}

// AnalyzeComment analyzes a comment for intent, sentiment, and metadata
func (ca *CommentAnalyzer) AnalyzeComment(ctx context.Context, comment *Comment) error {
	// Detect intent
	intent, confidence := ca.detectIntent(comment.Body)
	comment.Intent = intent
	comment.Metadata.Confidence = confidence

	// Analyze sentiment
	if ca.config.EnableSentimentAnalysis {
		sentiment := ca.analyzeSentiment(comment.Body)
		comment.Metadata.Sentiment = sentiment
	}

	// Extract keywords
	if ca.config.EnableKeywordExtraction {
		keywords := ca.extractKeywords(comment.Body)
		comment.Metadata.Keywords = keywords
	}

	// Determine priority
	priority := ca.determinePriority(comment)
	comment.Priority = priority

	// Assess complexity
	complexity := ca.assessComplexity(comment.Body)
	comment.Metadata.Complexity = complexity

	// Assess urgency
	urgency := ca.assessUrgency(comment.Body)
	comment.Metadata.Urgency = urgency

	// Check if action is required
	comment.Metadata.ActionRequired = ca.requiresAction(comment)

	// Extract mentioned code
	mentionedCode := ca.extractMentionedCode(comment.Body)
	comment.Metadata.MentionedCode = mentionedCode

	return nil
}

// detectIntent determines the intent of a comment
func (ca *CommentAnalyzer) detectIntent(body string) (CommentIntent, float64) {
	body = strings.ToLower(strings.TrimSpace(body))

	// Define intent patterns
	patterns := []struct {
		intent   CommentIntent
		keywords []string
		patterns []string
		weight   float64
	}{
		{
			intent:   CommentIntentQuestion,
			keywords: []string{"what", "why", "how", "when", "where", "which", "?"},
			patterns: []string{`\?`, `what\s+is`, `how\s+do`, `why\s+did`, `can\s+you\s+explain`},
			weight:   1.0,
		},
		{
			intent:   CommentIntentSuggestion,
			keywords: []string{"suggest", "recommend", "consider", "maybe", "perhaps", "could"},
			patterns: []string{`i\s+suggest`, `you\s+could`, `consider\s+using`, `maybe\s+try`},
			weight:   0.9,
		},
		{
			intent:   CommentIntentRequest,
			keywords: []string{"please", "can you", "could you", "need", "should", "must"},
			patterns: []string{`please\s+\w+`, `can\s+you\s+\w+`, `you\s+need\s+to`, `you\s+should`},
			weight:   1.0,
		},
		{
			intent:   CommentIntentBlocking,
			keywords: []string{keywordBlocking, "blocks", "prevents", "stops", "critical", "urgent"},
			patterns: []string{`this\s+blocks`, `blocking\s+merge`, `critical\s+issue`, `urgent\s+fix`},
			weight:   1.2,
		},
		{
			intent:   CommentIntentApproval,
			keywords: []string{"lgtm", "looks good", "approved", "approve", "great", "excellent"},
			patterns: []string{`looks?\s+good`, `lgtm`, `approved?`, `ship\s+it`},
			weight:   1.0,
		},
		{
			intent:   CommentIntentPraise,
			keywords: []string{"great", "awesome", "nice", "good job", "well done", "excellent"},
			patterns: []string{`great\s+work`, `nice\s+job`, `well\s+done`, `awesome`},
			weight:   0.8,
		},
		{
			intent:   CommentIntentConcern,
			keywords: []string{"concern", "worried", "problem", "issue", "trouble", "wrong"},
			patterns: []string{`i'm\s+concerned`, `this\s+could\s+cause`, `potential\s+issue`},
			weight:   0.9,
		},
	}

	bestIntent := CommentIntentClarification
	bestScore := 0.0

	for _, pattern := range patterns {
		score := ca.calculateIntentScore(body, pattern.keywords, pattern.patterns, pattern.weight)
		if score > bestScore && score >= ca.config.ConfidenceThreshold {
			bestScore = score
			bestIntent = pattern.intent
		}
	}

	return bestIntent, bestScore
}

// calculateIntentScore calculates confidence score for an intent
func (ca *CommentAnalyzer) calculateIntentScore(body string, keywords []string, patterns []string, weight float64) float64 {
	score := 0.0
	wordCount := len(strings.Fields(body))
	if wordCount == 0 {
		return 0.0
	}

	// Check keyword matches
	keywordMatches := 0
	for _, keyword := range keywords {
		if strings.Contains(body, keyword) {
			keywordMatches++
		}
	}
	keywordScore := float64(keywordMatches) / float64(len(keywords))

	// Check pattern matches
	patternMatches := 0
	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, body) //nolint:errcheck // Error indicates invalid regex which shouldn't happen with predefined patterns
		if matched {
			patternMatches++
		}
	}
	patternScore := float64(patternMatches) / float64(len(patterns))

	// Combine scores
	score = (keywordScore*0.4 + patternScore*0.6) * weight

	return score
}

// analyzeSentiment analyzes the sentiment of comment text
func (ca *CommentAnalyzer) analyzeSentiment(body string) float64 {
	body = strings.ToLower(body)

	positiveWords := []string{
		"good", "great", "excellent", "awesome", "nice", "perfect", "love",
		"amazing", "wonderful", "fantastic", "brilliant", "outstanding",
		"impressive", "helpful", "useful", "clean", "elegant", "smart",
	}

	negativeWords := []string{
		"bad", "terrible", "awful", "horrible", "wrong", "broken", "ugly",
		"messy", "confusing", "complicated", "difficult", "problematic",
		"concerning", "worrying", "dangerous", "risky", "inefficient",
	}

	positiveCount := 0
	negativeCount := 0

	words := strings.Fields(body)
	for _, word := range words {
		for _, pos := range positiveWords {
			if strings.Contains(word, pos) {
				positiveCount++
				break
			}
		}
		for _, neg := range negativeWords {
			if strings.Contains(word, neg) {
				negativeCount++
				break
			}
		}
	}

	totalSentimentWords := positiveCount + negativeCount
	if totalSentimentWords == 0 {
		return 0.0 // Neutral
	}

	// Return sentiment score between -1 (negative) and 1 (positive)
	return (float64(positiveCount) - float64(negativeCount)) / float64(totalSentimentWords)
}

// extractKeywords extracts important keywords from comment text
func (ca *CommentAnalyzer) extractKeywords(body string) []string {
	body = strings.ToLower(body)

	// Technical keywords to look for
	technicalKeywords := []string{
		"function", "method", "class", "variable", "constant", "interface",
		"struct", "type", "error", "exception", "bug", "test", "testing",
		"performance", "security", "memory", "database", "api", "endpoint",
		"authentication", "authorization", "validation", "configuration",
		"deployment", "build", "compile", "lint", "format", "style",
		"refactor", "optimize", "fix", "update", "add", "remove", "delete",
		"implement", "feature", "enhancement", "improvement", "documentation",
	}

	var found []string
	for _, keyword := range technicalKeywords {
		if strings.Contains(body, keyword) {
			found = append(found, keyword)
		}
	}

	return found
}

// determinePriority determines comment priority based on content and intent
func (ca *CommentAnalyzer) determinePriority(comment *Comment) CommentPriority {
	body := strings.ToLower(comment.Body)

	// Check for blocking keywords
	blockingKeywords := []string{keywordBlocking, "blocks", "critical", "security"}
	for _, keyword := range blockingKeywords {
		if strings.Contains(body, keyword) {
			if keyword == keywordBlocking || keyword == "blocks" {
				return CommentPriorityBlocking
			}
			return CommentPriorityCritical
		}
	}

	// Check intent-based priority
	switch comment.Intent {
	case CommentIntentBlocking:
		return CommentPriorityBlocking
	case CommentIntentRequest:
		if ca.hasUrgentKeywords(body) {
			return CommentPriorityHigh
		}
		return CommentPriorityMedium
	case CommentIntentConcern:
		return CommentPriorityHigh
	case CommentIntentQuestion, CommentIntentSuggestion:
		return CommentPriorityMedium
	case CommentIntentApproval, CommentIntentPraise:
		return CommentPriorityLow
	default:
		return CommentPriorityMedium
	}
}

// hasUrgentKeywords checks for urgent keywords
func (ca *CommentAnalyzer) hasUrgentKeywords(body string) bool {
	body = strings.ToLower(body)
	urgentKeywords := []string{"urgent", "asap", "immediately", "quickly", "soon"}
	for _, keyword := range urgentKeywords {
		if strings.Contains(body, keyword) {
			return true
		}
	}
	return false
}

// assessComplexity assesses the complexity of the comment
func (ca *CommentAnalyzer) assessComplexity(body string) ComplexityLevel {
	// Simple heuristics based on length and content
	wordCount := len(strings.Fields(body))

	if wordCount < 10 {
		return ComplexityLevelSimple
	} else if wordCount < 30 {
		return ComplexityLevelModerate
	} else if wordCount < 80 {
		return ComplexityLevelComplex
	}

	return ComplexityLevelVeryComplex
}

// assessUrgency assesses the urgency of the comment
func (ca *CommentAnalyzer) assessUrgency(body string) UrgencyLevel {
	body = strings.ToLower(body)

	if strings.Contains(body, "urgent") || strings.Contains(body, "asap") {
		return UrgencyLevelUrgent
	}
	if strings.Contains(body, "quickly") || strings.Contains(body, "soon") {
		return UrgencyLevelHigh
	}
	if strings.Contains(body, "when you can") || strings.Contains(body, "no rush") {
		return UrgencyLevelLow
	}

	return UrgencyLevelMedium
}

// requiresAction determines if the comment requires action
func (ca *CommentAnalyzer) requiresAction(comment *Comment) bool {
	switch comment.Intent {
	case CommentIntentRequest, CommentIntentBlocking, CommentIntentSuggestion:
		return true
	case CommentIntentQuestion:
		return true // Questions need responses
	case CommentIntentConcern:
		return true // Concerns need addressing
	case CommentIntentApproval, CommentIntentPraise:
		return false // These are just acknowledgments
	default:
		return comment.Priority >= CommentPriorityMedium
	}
}

// extractMentionedCode extracts code snippets or file references from comments
func (ca *CommentAnalyzer) extractMentionedCode(body string) []string {
	var mentioned []string

	// Look for code blocks first and remove them to avoid conflicts
	codeBlockRegex := regexp.MustCompile("```[\\s\\S]*?```")
	codeBlocks := codeBlockRegex.FindAllString(body, -1)
	mentioned = append(mentioned, codeBlocks...)

	// Remove code blocks from body to avoid inline code conflicts
	bodyWithoutCodeBlocks := codeBlockRegex.ReplaceAllString(body, "")

	// Look for inline code (single backticks, not spanning newlines)
	inlineCodeRegex := regexp.MustCompile("`[^`\n]+`")
	inlineCodes := inlineCodeRegex.FindAllString(bodyWithoutCodeBlocks, -1)
	mentioned = append(mentioned, inlineCodes...)

	// Look for file paths
	filePathRegex := regexp.MustCompile(`[\w/.-]+\.(go|js|ts|py|java|rs|cpp|c|h|json|yaml|yml|md|txt)`)
	filePaths := filePathRegex.FindAllString(body, -1)
	mentioned = append(mentioned, filePaths...)

	// Look for function/method names
	functionRegex := regexp.MustCompile(`\b[a-zA-Z_][a-zA-Z0-9_]*\(\)`)
	functions := functionRegex.FindAllString(body, -1)
	mentioned = append(mentioned, functions...)

	return mentioned
}

// ExtractCodeSuggestions extracts code suggestions from comments
func (ca *CommentAnalyzer) ExtractCodeSuggestions(comment *Comment) []CodeSuggestion {
	var suggestions []CodeSuggestion

	// Look for suggested changes
	body := comment.Body

	// Pattern: "change X to Y"
	changePattern := regexp.MustCompile(`(?i)change\s+(\w+)\s+to\s+(\w+)`)
	changeMatches := changePattern.FindAllStringSubmatch(body, -1)

	for _, match := range changeMatches {
		if len(match) >= 3 {
			suggestion := CodeSuggestion{
				File:        comment.File,
				StartLine:   comment.Line,
				EndLine:     comment.Line,
				OldCode:     strings.TrimSpace(match[1]),
				NewCode:     strings.TrimSpace(match[2]),
				Description: fmt.Sprintf("Change '%s' to '%s'", match[1], match[2]),
				Confidence:  0.7,
				Applied:     false,
			}
			suggestions = append(suggestions, suggestion)
		}
	}

	// Pattern: "replace X with Y"
	replacePattern := regexp.MustCompile(`(?i)replace\s+(\w+)\s+with\s+(\w+)`)
	replaceMatches := replacePattern.FindAllStringSubmatch(body, -1)

	for _, match := range replaceMatches {
		if len(match) >= 3 {
			suggestion := CodeSuggestion{
				File:        comment.File,
				StartLine:   comment.Line,
				EndLine:     comment.Line,
				OldCode:     strings.TrimSpace(match[1]),
				NewCode:     strings.TrimSpace(match[2]),
				Description: fmt.Sprintf("Replace '%s' with '%s'", match[1], match[2]),
				Confidence:  0.8,
				Applied:     false,
			}
			suggestions = append(suggestions, suggestion)
		}
	}

	// Look for code blocks with suggested changes
	codeBlockPattern := regexp.MustCompile("```\\w*\\n([\\s\\S]*?)```")
	codeBlocks := codeBlockPattern.FindAllStringSubmatch(body, -1)

	for _, match := range codeBlocks {
		if len(match) >= 2 {
			suggestion := CodeSuggestion{
				File:        comment.File,
				StartLine:   comment.Line,
				EndLine:     comment.Line + strings.Count(match[1], "\n"),
				NewCode:     strings.TrimSpace(match[1]),
				Description: "Code suggestion from comment",
				Confidence:  0.6,
				Applied:     false,
			}
			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions
}

// ExtractActionItems extracts action items from comments
func (ca *CommentAnalyzer) ExtractActionItems(comment *Comment) []ResponseAction {
	var actions []ResponseAction
	body := strings.ToLower(comment.Body)

	// Common action patterns
	actionPatterns := []struct {
		pattern     string
		actionType  ActionType
		description string
	}{
		{`add\s+test`, ActionTypeTestAdd, "Add test"},
		{`write\s+test`, ActionTypeTestAdd, "Write test"},
		{`create\s+test`, ActionTypeTestAdd, "Create test"},
		{`fix\s+`, ActionTypeCodeChange, "Fix issue"},
		{`update.*documentation`, ActionTypeDocUpdate, "Update documentation"},
		{`update\s+(?!.*documentation)`, ActionTypeFileModify, "Update file"},
		{`change\s+`, ActionTypeCodeChange, "Change code"},
		{`modify\s+`, ActionTypeFileModify, "Modify file"},
		{`refactor\s+`, ActionTypeRefactor, "Refactor code"},
		{`document\s+`, ActionTypeDocUpdate, "Update documentation"},
		{`add\s+comment`, ActionTypeDocUpdate, "Add comment"},
		{`explain\s+`, ActionTypeDocUpdate, "Add explanation"},
	}

	seenPatterns := make(map[string]bool)
	for _, pattern := range actionPatterns {
		matched, _ := regexp.MatchString(pattern.pattern, body) //nolint:errcheck // Error indicates invalid regex which shouldn't happen with predefined patterns
		if matched {
			// Avoid duplicate patterns, but allow same action type from different patterns
			if !seenPatterns[pattern.pattern] {
				action := ResponseAction{
					Type:        pattern.actionType,
					Description: pattern.description,
					File:        comment.File,
					Status:      "pending",
					ExecutedAt:  time.Now(),
				}
				actions = append(actions, action)
				seenPatterns[pattern.pattern] = true
			}
		}
	}

	// If no specific actions found but action is required, add generic action
	if len(actions) == 0 && comment.Metadata.ActionRequired {
		action := ResponseAction{
			Type:        ActionTypeInvestigate,
			Description: "Investigate and address comment",
			File:        comment.File,
			Status:      "pending",
			ExecutedAt:  time.Now(),
		}
		actions = append(actions, action)
	}

	return actions
}
