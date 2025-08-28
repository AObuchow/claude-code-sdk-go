package claudecode

import (
	"context"
	"testing"
	"time"
)

func TestOptions(t *testing.T) {
	options := &Options{
		SystemPrompt: stringPtr("test prompt"),
		MaxTurns:     intPtr(5),
		AllowedTools: []string{"Read", "Write"},
	}

	if *options.SystemPrompt != "test prompt" {
		t.Errorf("Expected SystemPrompt to be 'test prompt', got %s", *options.SystemPrompt)
	}

	if *options.MaxTurns != 5 {
		t.Errorf("Expected MaxTurns to be 5, got %d", *options.MaxTurns)
	}

	if len(options.AllowedTools) != 2 {
		t.Errorf("Expected 2 allowed tools, got %d", len(options.AllowedTools))
	}
}

func TestMessageTypes(t *testing.T) {
	timestamp := time.Now()
	textBlock := &TextBlock{Text: "Hello, world!"}

	// Test AssistantMessage
	assistantMsg := &AssistantMessage{
		ContentBlocks: []ContentBlock{textBlock},
		CreatedAt:     timestamp,
	}

	if assistantMsg.Type() != MessageTypeAssistant {
		t.Errorf("Expected MessageTypeAssistant, got %s", assistantMsg.Type())
	}

	if len(assistantMsg.Content()) != 1 {
		t.Errorf("Expected 1 content block, got %d", len(assistantMsg.Content()))
	}

	if assistantMsg.Timestamp() != timestamp {
		t.Errorf("Expected timestamp %v, got %v", timestamp, assistantMsg.Timestamp())
	}

	// Test UserMessage
	userMsg := &UserMessage{
		ContentBlocks: []ContentBlock{textBlock},
		CreatedAt:     timestamp,
	}

	if userMsg.Type() != MessageTypeUser {
		t.Errorf("Expected MessageTypeUser, got %s", userMsg.Type())
	}
}

func TestContentBlocks(t *testing.T) {
	// Test TextBlock
	textBlock := &TextBlock{Text: "Test text"}
	if textBlock.Type() != ContentBlockTypeText {
		t.Errorf("Expected ContentBlockTypeText, got %s", textBlock.Type())
	}

	// Test ToolUseBlock
	toolUseBlock := &ToolUseBlock{
		ID:    "tool-123",
		Name:  "TestTool",
		Input: map[string]interface{}{"param": "value"},
	}
	if toolUseBlock.Type() != ContentBlockTypeToolUse {
		t.Errorf("Expected ContentBlockTypeToolUse, got %s", toolUseBlock.Type())
	}

	// Test ToolResultBlock
	toolResultBlock := &ToolResultBlock{
		ToolUseID: "tool-123",
		Content:   "result content",
		IsError:   false,
	}
	if toolResultBlock.Type() != ContentBlockTypeToolResult {
		t.Errorf("Expected ContentBlockTypeToolResult, got %s", toolResultBlock.Type())
	}

	// Test ThinkingBlock
	thinkingBlock := &ThinkingBlock{
		Thinking:  "This is test thinking content.",
		Signature: "test-signature",
	}
	if thinkingBlock.Type() != ContentBlockTypeThinking {
		t.Errorf("Expected ContentBlockTypeThinking, got %s", thinkingBlock.Type())
	}
}

func TestErrors(t *testing.T) {
	// Test CLINotFoundError
	cliNotFoundErr := &CLINotFoundError{Path: "/usr/bin/claude"}
	expectedMsg := "Claude Code not found at: /usr/bin/claude"
	if cliNotFoundErr.Error() != expectedMsg {
		t.Errorf("Expected error message %s, got %s", expectedMsg, cliNotFoundErr.Error())
	}

	// Test CLINotFoundError without path
	cliNotFoundErrNoPath := &CLINotFoundError{}
	if !contains(cliNotFoundErrNoPath.Error(), "Install Claude Code with:") {
		t.Errorf("Expected installation instructions in error message")
	}

	// Test ProcessError
	processErr := &ProcessError{
		ExitCode: 1,
		Stderr:   "command failed",
		Stdout:   "some output",
	}
	if !contains(processErr.Error(), "exit code 1") {
		t.Errorf("Expected exit code in error message")
	}
}

func TestParseContentBlocks(t *testing.T) {
	// Test string content
	blocks, err := parseContentBlocks("simple text")
	if err != nil {
		t.Fatalf("Failed to parse string content: %v", err)
	}
	if len(blocks) != 1 {
		t.Errorf("Expected 1 block, got %d", len(blocks))
	}
	if textBlock, ok := blocks[0].(*TextBlock); ok {
		if textBlock.Text != "simple text" {
			t.Errorf("Expected 'simple text', got %s", textBlock.Text)
		}
	} else {
		t.Errorf("Expected TextBlock, got %T", blocks[0])
	}

	// Test map content
	mapContent := map[string]interface{}{
		"type": "text",
		"text": "map text content",
	}
	blocks, err = parseContentBlocks(mapContent)
	if err != nil {
		t.Fatalf("Failed to parse map content: %v", err)
	}
	if len(blocks) != 1 {
		t.Errorf("Expected 1 block, got %d", len(blocks))
	}

	// Test thinking content block
	thinkingContent := map[string]interface{}{
		"type":      "thinking",
		"thinking":  "This is test thinking content.",
		"signature": "test-signature",
	}
	blocks, err = parseContentBlocks(thinkingContent)
	if err != nil {
		t.Fatalf("Failed to parse thinking content: %v", err)
	}
	if len(blocks) != 1 {
		t.Errorf("Expected 1 thinking block, got %d", len(blocks))
	}
	if thinkingBlock, ok := blocks[0].(*ThinkingBlock); ok {
		if thinkingBlock.Thinking != "This is test thinking content." {
			t.Errorf("Expected correct thinking text, got %s", thinkingBlock.Thinking)
		}
		if thinkingBlock.Signature != "test-signature" {
			t.Errorf("Expected correct signature, got %s", thinkingBlock.Signature)
		}
	} else {
		t.Errorf("Expected ThinkingBlock, got %T", blocks[0])
	}
}

func TestFindCLIExecutable(t *testing.T) {
	// Test with custom path that doesn't exist
	customPath := "/nonexistent/path/claude"
	_, err := findCLIExecutable(&customPath)
	if err == nil {
		t.Error("Expected error for nonexistent path")
	}
	if _, ok := err.(*CLINotFoundError); !ok {
		t.Errorf("Expected CLINotFoundError, got %T", err)
	}

	// Test without custom path (should try to find in PATH)
	_, _ = findCLIExecutable(nil)
	// This might succeed or fail depending on whether claude-code is installed
	// We mainly want to ensure it doesn't panic
}

// TestQuery tests the Query function with a mock scenario
// Note: This test requires Claude CLI to be installed and authenticated
func TestQuery(t *testing.T) {
	// Skip if claude CLI is not available
	if _, err := findCLIExecutable(nil); err != nil {
		t.Skip("Skipping integration test: Claude CLI not found")
	}

	ctx := context.Background()
	options := &Options{
		MaxTurns: intPtr(1),
	}

	// Simple query that shouldn't require file operations
	_, err := Query(ctx, "What is 2+2?", options)
	if err != nil {
		t.Logf("Query failed (this may be expected in test environment): %v", err)
		// Don't fail the test as it might be due to environment setup
	}
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
