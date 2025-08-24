package main

import (
	"context"
	"fmt"
	"log"

	claudecode "github.com/AObuchow/claude-code-sdk-go"
)

func main() {
	// Authentication is handled by Claude CLI itself
	// Ensure 'claude' command is working before using this SDK

	ctx := context.Background()

	// Example 1: Using new QueryStreamWithRequest API
	fmt.Println("=== Example 1: QueryStreamWithRequest ===")
	request := claudecode.QueryRequest{
		Prompt: "List the files in the current directory and create a simple README.md",
		Options: &claudecode.Options{
			MaxTurns:     intPtr(3),
			AllowedTools: []string{"Read", "Write"},
			SystemPrompt: stringPtr("You are a helpful coding assistant. Be concise and direct."),
			OutputFormat: outputFormatPtr(claudecode.OutputFormatStreamJSON),
			Verbose:      boolPtr(true),
		},
	}

	messageChan, errorChan := claudecode.QueryStreamWithRequest(ctx, request)
	messageCount1 := processStreamingMessages(ctx, messageChan, errorChan, "QueryStreamWithRequest")

	// Example 2: Using traditional streaming API with new options
	fmt.Println("\n=== Example 2: Traditional QueryStream with new features ===")
	options := &claudecode.Options{
		MaxTurns:           intPtr(2),
		AllowedTools:       []string{"Read", "LS"},
		AppendSystemPrompt: stringPtr("Focus on showing file structure clearly."),
		OutputFormat:       outputFormatPtr(claudecode.OutputFormatStreamJSON),
		DisallowedTools:    []string{"Bash"},
	}

	// Execute streaming query
	fmt.Println("Starting traditional streaming query to Claude Code...")
	messageChan, errorChan = claudecode.QueryStream(ctx, "Show me the project structure", options)
	messageCount2 := processStreamingMessages(ctx, messageChan, errorChan, "QueryStream")

	fmt.Printf("\nCompleted both streaming examples! Total messages: %d + %d = %d\n", messageCount1, messageCount2, messageCount1+messageCount2)
}

// processStreamingMessages handles streaming message processing
func processStreamingMessages(ctx context.Context, messageChan <-chan claudecode.Message, errorChan <-chan error, apiName string) int {

	messageCount := 0

	// Process messages as they arrive
	for {
		select {
		case message, ok := <-messageChan:
			if !ok {
				fmt.Printf("\n%s streaming completed. Received %d messages total.\n", apiName, messageCount)
				return messageCount
			}

			messageCount++
			fmt.Printf("\n--- %s Message %d (%s) ---\n", apiName, messageCount, message.Type())

			for _, block := range message.Content() {
				switch b := block.(type) {
				case *claudecode.TextBlock:
					fmt.Printf("📝 Text: %s\n", b.Text)
				case *claudecode.ToolUseBlock:
					fmt.Printf("🔧 Tool Use: %s (ID: %s)\n", b.Name, b.ID)
					if len(b.Input) > 0 {
						fmt.Printf("   Input: %+v\n", b.Input)
					}
				case *claudecode.ToolResultBlock:
					fmt.Printf("✅ Tool Result (ID: %s)\n", b.ToolUseID)
					if b.IsError {
						fmt.Printf("❌ Error: %+v\n", b.Content)
					} else {
						fmt.Printf("   Result: %+v\n", b.Content)
					}
				}
			}

		case err, ok := <-errorChan:
			if !ok {
				continue
			}
			if err != nil {
				log.Fatalf("%s streaming error: %v", apiName, err)
			}

		case <-ctx.Done():
			fmt.Printf("\n%s context cancelled\n", apiName)
			return messageCount
		}
	}
}

// Helper functions for pointer types
func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func outputFormatPtr(format claudecode.OutputFormat) *claudecode.OutputFormat {
	return &format
}