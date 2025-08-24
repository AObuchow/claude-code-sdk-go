package main

import (
	"context"
	"fmt"
	"log"

	claudecode "github.com/AObuchow/claude-code-sdk-go"
)

// basicExample demonstrates a simple question
func basicExample(ctx context.Context) {
	fmt.Println("=== Basic Example ===")

	request := claudecode.QueryRequest{
		Prompt: "What is 2 + 2?",
	}

	messages, err := claudecode.QueryWithRequest(ctx, request)
	if err != nil {
		log.Fatalf("Basic example failed: %v", err)
	}

	for _, message := range messages {
		if assistantMsg, ok := message.(*claudecode.AssistantMessage); ok {
			for _, block := range assistantMsg.Content() {
				if textBlock, ok := block.(*claudecode.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		}
	}
	fmt.Println()
}

// withOptionsExample demonstrates using custom options
func withOptionsExample(ctx context.Context) {
	fmt.Println("=== With Options Example ===")

	request := claudecode.QueryRequest{
		Prompt: "Explain what Go is in one sentence.",
		Options: &claudecode.Options{
			SystemPrompt: stringPtr("You are a helpful assistant that explains things simply."),
			MaxTurns:     intPtr(1),
		},
	}

	messages, err := claudecode.QueryWithRequest(ctx, request)
	if err != nil {
		log.Fatalf("With options example failed: %v", err)
	}

	for _, message := range messages {
		if assistantMsg, ok := message.(*claudecode.AssistantMessage); ok {
			for _, block := range assistantMsg.Content() {
				if textBlock, ok := block.(*claudecode.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		}
	}
	fmt.Println()
}

// withToolsExample demonstrates using tools
func withToolsExample(ctx context.Context) {
	fmt.Println("=== With Tools Example ===")

	request := claudecode.QueryRequest{
		Prompt: "Create a file called hello.txt with 'Hello, World!' in it",
		Options: &claudecode.Options{
			AllowedTools: []string{"Read", "Write"},
			SystemPrompt: stringPtr("You are a helpful file assistant."),
		},
	}

	messages, err := claudecode.QueryWithRequest(ctx, request)
	if err != nil {
		log.Fatalf("With tools example failed: %v", err)
	}

	for _, message := range messages {
		switch msg := message.(type) {
		case *claudecode.AssistantMessage:
			for _, block := range msg.Content() {
				if textBlock, ok := block.(*claudecode.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		case *claudecode.ResultMessage:
			if msg.TotalCostUSD != nil && *msg.TotalCostUSD > 0 {
				fmt.Printf("\nCost: $%.4f\n", *msg.TotalCostUSD)
			}
		}
	}
	fmt.Println()
}

func main() {
	// Authentication is handled by Claude CLI itself
	// Ensure 'claude' command is working before using this SDK

	ctx := context.Background()

	// Run all examples
	basicExample(ctx)
	withOptionsExample(ctx)
	withToolsExample(ctx)
}

// Helper functions for pointer types
func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}