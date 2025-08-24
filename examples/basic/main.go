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

	// Basic usage example
	fmt.Println("=== Basic Usage ===")
	request := claudecode.QueryRequest{
		Prompt: "Create a simple hello.go file that prints 'Hello, World!'",
		Options: &claudecode.Options{
			AllowedTools: []string{"Read", "Write"},
			SystemPrompt: stringPtr("You are a helpful Go programming assistant"),
		},
	}

	messages, err := claudecode.QueryWithRequest(ctx, request)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	processMessages(messages)
}

// processMessages handles the display of messages
func processMessages(messages []claudecode.Message) {
	for i, message := range messages {
		fmt.Printf("--- Message %d (%s) ---\n", i+1, message.Type())
		
		for j, block := range message.Content() {
			fmt.Printf("Content Block %d (%s):\n", j+1, block.Type())
			
			switch b := block.(type) {
			case *claudecode.TextBlock:
				fmt.Printf("Text: %s\n", b.Text)
			case *claudecode.ToolUseBlock:
				fmt.Printf("Tool: %s (ID: %s)\n", b.Name, b.ID)
				fmt.Printf("Input: %+v\n", b.Input)
			case *claudecode.ToolResultBlock:
				fmt.Printf("Tool Result (ID: %s):\n", b.ToolUseID)
				fmt.Printf("Content: %+v\n", b.Content)
				if b.IsError {
					fmt.Println("(This was an error)")
				}
			}
			fmt.Println()
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

func outputFormatPtr(format claudecode.OutputFormat) *claudecode.OutputFormat {
	return &format
}