package help

import (
	"strings"
	"testing"
)

// Test constants
const (
	testCommandProcess = "process"
)

func TestNewHelpSystem(t *testing.T) {
	hs := NewHelpSystem()
	if hs == nil {
		t.Fatal("NewHelpSystem returned nil")
	}

	if hs.topics == nil {
		t.Error("Expected topics map to be initialized")
	}

	if hs.commands == nil {
		t.Error("Expected commands map to be initialized")
	}

	// Verify built-in topics are loaded
	expectedTopics := []string{"configuration", "authentication", "workflows", "troubleshooting"}
	for _, topic := range expectedTopics {
		if _, exists := hs.topics[topic]; !exists {
			t.Errorf("Expected topic '%s' to be initialized", topic)
		}
	}

	// Verify built-in commands are loaded
	expectedCommands := []string{testCommandProcess, "status", "config"}
	for _, cmd := range expectedCommands {
		if _, exists := hs.commands[cmd]; !exists {
			t.Errorf("Expected command '%s' to be initialized", cmd)
		}
	}
}

func TestGetTopic(t *testing.T) {
	hs := NewHelpSystem()

	// Test existing topic
	topic, exists := hs.GetTopic("configuration")
	if !exists {
		t.Error("Expected 'configuration' topic to exist")
	}
	if topic == nil {
		t.Error("Expected topic to not be nil")
		return
	}
	if topic.Title != "Configuration" {
		t.Errorf("Expected title 'Configuration', got '%s'", topic.Title)
	}

	// Test case insensitive
	_, exists = hs.GetTopic("CONFIGURATION")
	if !exists {
		t.Error("Expected case insensitive topic lookup to work")
	}

	// Test non-existing topic
	_, exists = hs.GetTopic("nonexistent")
	if exists {
		t.Error("Expected non-existent topic to not exist")
	}
}

func TestGetCommand(t *testing.T) {
	hs := NewHelpSystem()

	// Test existing command
	cmd, exists := hs.GetCommand(testCommandProcess)
	if !exists {
		t.Error("Expected 'process' command to exist")
	}
	if cmd == nil {
		t.Error("Expected command to not be nil")
		return
	}
	if cmd.Name != testCommandProcess {
		t.Errorf("Expected name 'process', got '%s'", cmd.Name)
	}

	// Test case insensitive
	_, exists = hs.GetCommand("PROCESS")
	if !exists {
		t.Error("Expected case insensitive command lookup to work")
	}

	// Test non-existing command
	_, exists = hs.GetCommand("nonexistent")
	if exists {
		t.Error("Expected non-existent command to not exist")
	}
}

func TestListTopics(t *testing.T) {
	hs := NewHelpSystem()
	topics := hs.ListTopics()

	if len(topics) == 0 {
		t.Error("Expected at least some topics to be listed")
	}

	// Should be sorted
	for i := 1; i < len(topics); i++ {
		if topics[i] < topics[i-1] {
			t.Error("Expected topics to be sorted alphabetically")
			break
		}
	}

	// Check for expected topics
	expectedTopics := []string{"authentication", "configuration", "troubleshooting", "workflows"}
	for _, expected := range expectedTopics {
		found := false
		for _, topic := range topics {
			if topic == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected topic '%s' to be in list", expected)
		}
	}
}

func TestListCommands(t *testing.T) {
	hs := NewHelpSystem()
	commands := hs.ListCommands()

	if len(commands) == 0 {
		t.Error("Expected at least some commands to be listed")
	}

	// Should be sorted
	for i := 1; i < len(commands); i++ {
		if commands[i] < commands[i-1] {
			t.Error("Expected commands to be sorted alphabetically")
			break
		}
	}

	// Check for expected commands
	expectedCommands := []string{"config", testCommandProcess, "status"}
	for _, expected := range expectedCommands {
		found := false
		for _, cmd := range commands {
			if cmd == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected command '%s' to be in list", expected)
		}
	}
}

func TestSearchTopics(t *testing.T) {
	hs := NewHelpSystem()

	// Test search by title
	results := hs.SearchTopics("configuration")
	if len(results) == 0 {
		t.Error("Expected to find topics matching 'configuration'")
	}
	found := false
	for _, topic := range results {
		if strings.ToLower(topic.Title) == "configuration" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find configuration topic by title")
	}

	// Test search by tag
	results = hs.SearchTopics("auth")
	if len(results) == 0 {
		t.Error("Expected to find topics matching 'auth' tag")
	}

	// Test search case insensitive
	results = hs.SearchTopics("WORKFLOW")
	if len(results) == 0 {
		t.Error("Expected case insensitive search to work")
	}

	// Test search no matches
	results = hs.SearchTopics("nonexistentterm")
	if len(results) != 0 {
		t.Error("Expected no results for non-existent search term")
	}
}

func TestTopicMatches(t *testing.T) {
	hs := NewHelpSystem()
	topic := &HelpTopic{
		Title:       "Test Topic",
		Description: "A description about configuration",
		Content:     "Content with workflow information",
		Tags:        []string{"testing", "example"},
	}

	// Test title match
	if !hs.topicMatches(topic, "test") {
		t.Error("Expected match on title")
	}

	// Test description match
	if !hs.topicMatches(topic, "configuration") {
		t.Error("Expected match on description")
	}

	// Test content match
	if !hs.topicMatches(topic, "workflow") {
		t.Error("Expected match on content")
	}

	// Test tag match
	if !hs.topicMatches(topic, "testing") {
		t.Error("Expected match on tag")
	}

	// Test no match
	if hs.topicMatches(topic, "nomatch") {
		t.Error("Expected no match for unrelated term")
	}

	// Test case insensitive
	if !hs.topicMatches(topic, "TEST") {
		t.Error("Expected case insensitive match")
	}
}

func TestGetQuickStart(t *testing.T) {
	hs := NewHelpSystem()
	quickStart := hs.GetQuickStart()

	if quickStart == "" {
		t.Error("Expected quick start guide to not be empty")
	}

	// Check for expected sections
	expectedSections := []string{"Initial Setup", "Authentication", "Process an Issue", "Monitor Progress", "Get Help"}
	for _, section := range expectedSections {
		if !strings.Contains(quickStart, section) {
			t.Errorf("Expected quick start to contain section '%s'", section)
		}
	}

	// Check for expected commands
	expectedCommands := []string{"ccagents init", "ccagents validate", "gh auth login", "claude auth"}
	for _, cmd := range expectedCommands {
		if !strings.Contains(quickStart, cmd) {
			t.Errorf("Expected quick start to contain command '%s'", cmd)
		}
	}
}

func TestGetContextualHelp(t *testing.T) {
	hs := NewHelpSystem()

	testCases := []struct {
		context  string
		expected string
	}{
		{"authentication_failed", "Authentication Help"},
		{"configuration_error", "Configuration Help"},
		{"github_error", "GitHub Help"},
		{"claude_error", "Claude Help"},
		{"workflow_failed", "Workflow Help"},
	}

	for _, tc := range testCases {
		help := hs.GetContextualHelp(tc.context, "", nil)
		if !strings.Contains(help, tc.expected) {
			t.Errorf("Expected contextual help for '%s' to contain '%s'", tc.context, tc.expected)
		}
	}

	// Test with command context
	help := hs.GetContextualHelp("", testCommandProcess, nil)
	if !strings.Contains(help, testCommandProcess) {
		t.Error("Expected contextual help to include command help for 'process'")
	}

	// Test default context
	help = hs.GetContextualHelp("unknown", "", nil)
	if !strings.Contains(help, "Quick Start") {
		t.Error("Expected default contextual help to be quick start guide")
	}
}

func TestFormatCommandHelp(t *testing.T) {
	hs := NewHelpSystem()
	cmd := &CommandHelp{
		Name:        "test",
		Usage:       "test [options]",
		Description: "A test command",
		Options: []CommandOption{
			{
				Name:        "verbose",
				Short:       "v",
				Description: "Enable verbose output",
				Type:        "bool",
			},
			{
				Name:        "config",
				Description: "Configuration file",
				Type:        "string",
				Default:     "config.yaml",
			},
		},
		Examples: []HelpExample{
			{
				Title:       "Basic usage",
				Description: "Run with default settings",
				Command:     "test",
			},
		},
		SeeAlso: []string{"help", "init"},
	}

	formatted := hs.formatCommandHelp(cmd)

	// Check structure
	if !strings.Contains(formatted, "# test") {
		t.Error("Expected formatted help to contain command name as header")
	}
	if !strings.Contains(formatted, "**Usage:** test [options]") {
		t.Error("Expected formatted help to contain usage")
	}
	if !strings.Contains(formatted, "A test command") {
		t.Error("Expected formatted help to contain description")
	}
	if !strings.Contains(formatted, "## Options") {
		t.Error("Expected formatted help to contain options section")
	}
	if !strings.Contains(formatted, "--verbose, -v") {
		t.Error("Expected formatted help to contain option with short form")
	}
	if !strings.Contains(formatted, "(default: config.yaml)") {
		t.Error("Expected formatted help to contain default value")
	}
	if !strings.Contains(formatted, "## Examples") {
		t.Error("Expected formatted help to contain examples section")
	}
	if !strings.Contains(formatted, "## See Also") {
		t.Error("Expected formatted help to contain see also section")
	}
}

func TestFormatHelpTopic(t *testing.T) {
	hs := NewHelpSystem()
	topic := &HelpTopic{
		Title:       "Test Topic",
		Description: "A test topic",
		Content:     "Detailed content here",
		Category:    "testing",
		Difficulty:  "beginner",
		Examples: []HelpExample{
			{
				Title:       "Example 1",
				Description: "First example",
				Command:     "command1",
				Output:      "output1",
			},
		},
		SeeAlso: []string{"related1", "related2"},
		Tags:    []string{"tag1", "tag2"},
	}

	formatted := hs.FormatHelpTopic(topic)

	// Check structure
	if !strings.Contains(formatted, "# Test Topic") {
		t.Error("Expected formatted topic to contain title as header")
	}
	if !strings.Contains(formatted, "A test topic") {
		t.Error("Expected formatted topic to contain description")
	}
	if !strings.Contains(formatted, "Detailed content here") {
		t.Error("Expected formatted topic to contain content")
	}
	if !strings.Contains(formatted, "## Examples") {
		t.Error("Expected formatted topic to contain examples section")
	}
	if !strings.Contains(formatted, "### Example 1") {
		t.Error("Expected formatted topic to contain example title")
	}
	if !strings.Contains(formatted, "```bash\ncommand1\n```") {
		t.Error("Expected formatted topic to contain command in code block")
	}
	if !strings.Contains(formatted, "Expected output:\n```\noutput1\n```") {
		t.Error("Expected formatted topic to contain output")
	}
	if !strings.Contains(formatted, "## See Also") {
		t.Error("Expected formatted topic to contain see also section")
	}
	if !strings.Contains(formatted, "**Category:** testing") {
		t.Error("Expected formatted topic to contain category")
	}
	if !strings.Contains(formatted, "**Difficulty:** beginner") {
		t.Error("Expected formatted topic to contain difficulty")
	}
	if !strings.Contains(formatted, "**Tags:** tag1, tag2") {
		t.Error("Expected formatted topic to contain tags")
	}
}

func TestGetTroubleshootingHelp(t *testing.T) {
	hs := NewHelpSystem()

	testCases := []struct {
		errorType string
		expected  string
	}{
		{"network_timeout", "Network Timeout Help"},
		{"authentication_error", "Authentication Error Help"},
		{"configuration_error", "Configuration Error Help"},
		{"workflow_failed", "Workflow Failure Help"},
		{"unknown_error", "General Troubleshooting"},
	}

	for _, tc := range testCases {
		help := hs.GetTroubleshootingHelp(tc.errorType, "test error message")
		if !strings.Contains(help, tc.expected) {
			t.Errorf("Expected troubleshooting help for '%s' to contain '%s'", tc.errorType, tc.expected)
		}
	}

	// Test with custom error message
	help := hs.GetTroubleshootingHelp("unknown_error", "custom error")
	if !strings.Contains(help, "custom error") {
		t.Error("Expected custom error message to be included in help")
	}
}

func TestGetSuggestedCommands(t *testing.T) {
	hs := NewHelpSystem()

	testCases := []struct {
		context  string
		expected []string
	}{
		{"new_user", []string{"ccagents init", "ccagents validate", "gh auth login"}},
		{"workflow_failed", []string{"ccagents workflow status", "ccagents logs"}},
		{"authentication_failed", []string{"gh auth status", "claude auth status"}},
		{"configuration_error", []string{"ccagents validate", "ccagents config dump"}},
		{"unknown", []string{"ccagents help", "ccagents status"}},
	}

	for _, tc := range testCases {
		suggestions := hs.GetSuggestedCommands(tc.context)
		if len(suggestions) == 0 {
			t.Errorf("Expected suggestions for context '%s'", tc.context)
		}

		for _, expected := range tc.expected {
			found := false
			for _, suggestion := range suggestions {
				if strings.Contains(suggestion, expected) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected suggestions for '%s' to contain '%s'", tc.context, expected)
			}
		}
	}
}

func TestSearchCommands(t *testing.T) {
	hs := NewHelpSystem()

	// Test search by name
	results := hs.SearchCommands(testCommandProcess)
	if len(results) == 0 {
		t.Error("Expected to find commands matching 'process'")
	}
	found := false
	for _, cmd := range results {
		if cmd.Name == testCommandProcess {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find process command by name")
	}

	// Test search by description
	results = hs.SearchCommands("status")
	if len(results) == 0 {
		t.Error("Expected to find commands matching 'status'")
	}

	// Test case insensitive
	results = hs.SearchCommands("CONFIG")
	if len(results) == 0 {
		t.Error("Expected case insensitive command search to work")
	}

	// Test no matches
	results = hs.SearchCommands("nonexistentcommand")
	if len(results) != 0 {
		t.Error("Expected no results for non-existent command search")
	}
}

func TestCommandMatches(t *testing.T) {
	hs := NewHelpSystem()
	cmd := &CommandHelp{
		Name:        "test",
		Description: "A test command for configuration",
		Usage:       "test [options]",
		Options: []CommandOption{
			{
				Name:        "verbose",
				Description: "Enable debugging output",
			},
		},
	}

	// Test name match
	if !hs.commandMatches(cmd, "test") {
		t.Error("Expected match on command name")
	}

	// Test description match
	if !hs.commandMatches(cmd, "configuration") {
		t.Error("Expected match on command description")
	}

	// Test usage match
	if !hs.commandMatches(cmd, "options") {
		t.Error("Expected match on command usage")
	}

	// Test option match
	if !hs.commandMatches(cmd, "verbose") {
		t.Error("Expected match on option name")
	}
	if !hs.commandMatches(cmd, "debugging") {
		t.Error("Expected match on option description")
	}

	// Test no match
	if hs.commandMatches(cmd, "nomatch") {
		t.Error("Expected no match for unrelated term")
	}

	// Test case insensitive
	if !hs.commandMatches(cmd, "TEST") {
		t.Error("Expected case insensitive match")
	}
}

func TestGetCommandByName(t *testing.T) {
	hs := NewHelpSystem()

	// Test existing command
	cmd := hs.GetCommandByName(testCommandProcess)
	if cmd == nil {
		t.Error("Expected to find process command")
	}
	if cmd != nil && cmd.Name != testCommandProcess {
		t.Errorf("Expected command name 'process', got '%s'", cmd.Name)
	}

	// Test case insensitive
	cmd = hs.GetCommandByName("PROCESS")
	if cmd == nil {
		t.Error("Expected case insensitive command lookup to work")
	}

	// Test non-existing command
	cmd = hs.GetCommandByName("nonexistent")
	if cmd != nil {
		t.Error("Expected nil for non-existent command")
	}
}

func TestAddCustomTopic(t *testing.T) {
	hs := NewHelpSystem()
	customTopic := &HelpTopic{
		Title:       "Custom Topic",
		Description: "A custom help topic",
		Category:    "custom",
	}

	hs.AddCustomTopic("custom", customTopic)

	// Test retrieval
	topic, exists := hs.GetTopic("custom")
	if !exists {
		t.Error("Expected custom topic to exist after adding")
	}
	if topic.Title != "Custom Topic" {
		t.Error("Expected custom topic to have correct title")
	}

	// Test case insensitive key
	hs.AddCustomTopic("ANOTHER", customTopic)
	_, exists = hs.GetTopic("another")
	if !exists {
		t.Error("Expected custom topic key to be case insensitive")
	}
}

func TestAddCustomCommand(t *testing.T) {
	hs := NewHelpSystem()
	customCmd := &CommandHelp{
		Name:        "custom",
		Description: "A custom command",
		Usage:       "custom [args]",
	}

	hs.AddCustomCommand("custom", customCmd)

	// Test retrieval
	cmd, exists := hs.GetCommand("custom")
	if !exists {
		t.Error("Expected custom command to exist after adding")
	}
	if cmd.Name != "custom" {
		t.Error("Expected custom command to have correct name")
	}

	// Test case insensitive key
	hs.AddCustomCommand("ANOTHER", customCmd)
	_, exists = hs.GetCommand("another")
	if !exists {
		t.Error("Expected custom command key to be case insensitive")
	}
}

func TestHelpSystemCoverage(t *testing.T) {
	hs := NewHelpSystem()

	// Test that all major public methods work without panicking
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Help system method panicked: %v", r)
		}
	}()

	// Exercise all public methods
	_ = hs.ListTopics()
	_ = hs.ListCommands()
	_ = hs.SearchTopics("test")
	_ = hs.SearchCommands("test")
	_ = hs.GetQuickStart()
	_ = hs.GetContextualHelp("test", "test", []string{})
	_ = hs.GetTroubleshootingHelp("test", "test")
	_ = hs.GetSuggestedCommands("test")
	_ = hs.GetCommandByName("test")

	// Test topic formatting
	if topic, exists := hs.GetTopic("configuration"); exists {
		_ = hs.FormatHelpTopic(topic)
	}

	// Test command formatting
	if cmd, exists := hs.GetCommand(testCommandProcess); exists {
		_ = hs.formatCommandHelp(cmd)
	}
}
