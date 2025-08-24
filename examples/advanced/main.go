package main

import (
	"context"
	"fmt"
	"log"

	claudecode "github.com/AObuchow/claude-code-sdk-go"
)

func main() {
	// This example demonstrates advanced features of the Claude Code SDK
	ctx := context.Background()

	// Example 1: Using MCP and permission prompt tools
	fmt.Println("=== Example 1: MCP Configuration ===")
	fmt.Println("Note: This example demonstrates MCP configuration but is commented out")
	fmt.Println("To use MCP features, create mcp-servers.json and uncomment the code below")
	
	// Example MCP request (commented out to avoid errors when MCP is not configured)
	_ = claudecode.QueryRequest{
		Prompt: "Use filesystem tools to analyze the project structure",
		Options: &claudecode.Options{
			MaxTurns:            intPtr(5),
			AllowedTools:        []string{"mcp__filesystem__read_file", "mcp__filesystem__list_directory"},
			MCPConfig:           stringPtr("mcp-servers.json"),
			PermissionPromptTool: stringPtr("mcp__permissions__approve"),
			OutputFormat:        outputFormatPtr(claudecode.OutputFormatJSON),
			Verbose:             boolPtr(true),
		},
	}
	
	// messages, err := claudecode.QueryWithRequest(ctx, mcpRequest)
	// if err != nil {
	//     log.Printf("MCP query failed (expected if MCP not configured): %v", err)
	// } else {
	//     fmt.Printf("Received %d messages with MCP\n", len(messages))
	// }

	// Example 2: Session resumption and continuation
	fmt.Println("\n=== Example 2: Session Management ===")
	
	// Start a new session
	initialRequest := claudecode.QueryRequest{
		Prompt: "Create a simple Go function that calculates fibonacci numbers",
		Options: &claudecode.Options{
			MaxTurns:     intPtr(2),
			AllowedTools: []string{"Write"},
			OutputFormat: outputFormatPtr(claudecode.OutputFormatJSON),
		},
	}

	messages, err := claudecode.QueryWithRequest(ctx, initialRequest)
	if err != nil {
		log.Fatalf("Initial request failed: %v", err)
	}

	// Extract session ID from a result message (simplified for example)
	var sessionID string
	for _, message := range messages {
		if _, ok := message.(*claudecode.ResultMessage); ok {
			// In a real implementation, you'd extract the session ID from the result
			// For this example, we'll use a placeholder
			sessionID = "example-session-id"
			break
		}
	}

	if sessionID != "" {
		// Demonstrate resume functionality (commented out for this example)
		_ = claudecode.QueryRequest{
			Prompt: "Now add unit tests for the fibonacci function",
			Options: &claudecode.Options{
				Resume:       stringPtr(sessionID),
				MaxTurns:     intPtr(2),
				AllowedTools: []string{"Write"},
				OutputFormat: outputFormatPtr(claudecode.OutputFormatText),
			},
		}

		fmt.Println("Session management example prepared (commented out for demo)")
		// continuedMessages, err := claudecode.QueryWithRequest(ctx, continueRequest)
		// if err != nil {
		//     log.Printf("Continue request failed: %v", err)
		// } else {
		//     fmt.Printf("Received %d continued messages\n", len(continuedMessages))
		// }
	}

	// Example 3: Tool restriction and custom system prompts
	fmt.Println("\n=== Example 3: Tool Restrictions and Custom Prompts ===")
	restrictedRequest := claudecode.QueryRequest{
		Prompt: "Analyze this Go project and suggest improvements",
		Options: &claudecode.Options{
			SystemPrompt:       stringPtr("You are a senior Go architect. Focus on performance, maintainability, and best practices."),
			AppendSystemPrompt: stringPtr("Always provide specific, actionable recommendations."),
			MaxTurns:           intPtr(3),
			AllowedTools:       []string{"Read", "LS", "Grep"},
			DisallowedTools:    []string{"Write", "Bash"},
			OutputFormat:       outputFormatPtr(claudecode.OutputFormatStreamJSON),
			Verbose:            boolPtr(false),
		},
	}

	fmt.Println("Starting analysis with tool restrictions...")
	messageChan, errorChan := claudecode.QueryStreamWithRequest(ctx, restrictedRequest)
	
	messageCount := 0
	for {
		select {
		case message, ok := <-messageChan:
			if !ok {
				fmt.Printf("Analysis completed. Processed %d messages.\n", messageCount)
				goto nextExample
			}
			
			messageCount++
			fmt.Printf("📊 Analysis Message %d (%s)\n", messageCount, message.Type())
			
			for _, block := range message.Content() {
				switch b := block.(type) {
				case *claudecode.TextBlock:
					if len(b.Text) > 100 {
						fmt.Printf("   Text: %s...\n", b.Text[:100])
					} else {
						fmt.Printf("   Text: %s\n", b.Text)
					}
				case *claudecode.ToolUseBlock:
					fmt.Printf("   🔧 Using tool: %s\n", b.Name)
				case *claudecode.ToolResultBlock:
					fmt.Printf("   ✅ Tool completed: %s\n", b.ToolUseID)
				}
			}

		case err, ok := <-errorChan:
			if !ok {
				continue
			}
			if err != nil {
				log.Printf("Analysis error: %v", err)
				goto nextExample
			}

		case <-ctx.Done():
			fmt.Println("Analysis cancelled")
			goto nextExample
		}
	}

nextExample:
	// Example 4: Different output formats comparison
	fmt.Println("\n=== Example 4: Output Format Comparison ===")
	
	formats := []claudecode.OutputFormat{
		claudecode.OutputFormatText,
		claudecode.OutputFormatJSON,
		claudecode.OutputFormatStreamJSON,
	}

	for _, format := range formats {
		fmt.Printf("\n--- Testing %s format ---\n", format)
		
		formatRequest := claudecode.QueryRequest{
			Prompt: "List the Go files in this project",
			Options: &claudecode.Options{
				MaxTurns:     intPtr(1),
				AllowedTools: []string{"LS", "Glob"},
				OutputFormat: &format,
			},
		}

		messages, err := claudecode.QueryWithRequest(ctx, formatRequest)
		if err != nil {
			log.Printf("Format %s failed: %v", format, err)
			continue
		}

		fmt.Printf("Format %s: Received %d messages\n", format, len(messages))
		
		// Show first message content for comparison
		if len(messages) > 0 {
			content := messages[0].Content()
			if len(content) > 0 {
				if textBlock, ok := content[0].(*claudecode.TextBlock); ok {
					if len(textBlock.Text) > 50 {
						fmt.Printf("   First content: %s...\n", textBlock.Text[:50])
					} else {
						fmt.Printf("   First content: %s\n", textBlock.Text)
					}
				}
			}
		}
	}

	fmt.Println("\n🎉 Advanced examples completed!")
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