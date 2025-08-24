# Claude Code SDK for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/AObuchow/claude-code-sdk-go.svg)](https://pkg.go.dev/github.com/AObuchow/claude-code-sdk-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/AObuchow/claude-code-sdk-go)](https://goreportcard.com/report/github.com/AObuchow/claude-code-sdk-go)

A Go SDK for Claude Code that provides programmatic access to Claude's agentic coding capabilities. This SDK wraps the Claude Code CLI and provides a Go-native interface compatible with the TypeScript and Python SDKs.

## Features

- **Full CLI Option Support**: All Claude Code CLI options are supported
- **TypeScript/Python SDK Compatible API**: `QueryRequest` pattern for consistency across languages  
- **True Streaming**: Real-time message processing using Go channels
- **Session Management**: Resume and continue conversations
- **MCP Support**: Model Context Protocol integration
- **Multiple Output Formats**: text, json, and stream-json
- **Robust Error Handling**: Detailed error types with context
- **Backward Compatibility**: Existing APIs continue to work

## Prerequisites

- Go 1.21 or later
- Claude CLI: `npm install -g @anthropic-ai/claude-code`
- Authenticated Claude CLI (run `claude` to verify)

## Installation

```bash
go get github.com/AObuchow/claude-code-sdk-go
```

## Quick Start

### Basic Usage (TypeScript/Python Compatible API)

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    claudecode "github.com/AObuchow/claude-code-sdk-go"
)

func main() {
    ctx := context.Background()
    
    // Create a request using the TypeScript/Python SDK compatible format
    request := claudecode.QueryRequest{
        Prompt: "Create a simple hello.go file that prints 'Hello, World!'",
        Options: &claudecode.Options{
            MaxTurns:     intPtr(3),
            AllowedTools: []string{"Read", "Write"},
            SystemPrompt: stringPtr("You are a helpful Go programming assistant"),
        },
    }
    
    messages, err := claudecode.QueryWithRequest(ctx, request)
    if err != nil {
        log.Fatalf("Query failed: %v", err)
    }
    
    fmt.Printf("Received %d messages\n", len(messages))
    for _, message := range messages {
        fmt.Printf("Message type: %s\n", message.Type())
        for _, block := range message.Content() {
            if textBlock, ok := block.(*claudecode.TextBlock); ok {
                fmt.Printf("Content: %s\n", textBlock.Text)
            }
        }
    }
}

func intPtr(i int) *int { return &i }
func stringPtr(s string) *string { return &s }
```

### Streaming Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    claudecode "github.com/AObuchow/claude-code-sdk-go"
)

func main() {
    ctx := context.Background()
    
    request := claudecode.QueryRequest{
        Prompt: "Analyze this Go project and suggest improvements",
        Options: &claudecode.Options{
            AllowedTools: []string{"Read", "LS", "Grep"},
            OutputFormat: outputFormatPtr(claudecode.OutputFormatStreamJSON),
            Verbose:      boolPtr(true),
        },
    }
    
    messageChan, errorChan := claudecode.QueryStreamWithRequest(ctx, request)
    
    for {
        select {
        case message, ok := <-messageChan:
            if !ok {
                fmt.Println("Streaming completed")
                return
            }
            
            fmt.Printf("Received %s message\n", message.Type())
            // Process message...
            
        case err := <-errorChan:
            if err != nil {
                log.Fatalf("Streaming error: %v", err)
            }
            
        case <-ctx.Done():
            fmt.Println("Context cancelled")
            return
        }
    }
}

func boolPtr(b bool) *bool { return &b }
func outputFormatPtr(f claudecode.OutputFormat) *claudecode.OutputFormat { return &f }
```

## Configuration Options

The `Options` struct supports all Claude Code CLI options:

```go
type Options struct {
    // System prompts
    SystemPrompt       *string           // Custom system prompt
    AppendSystemPrompt *string           // Append to default system prompt
    
    // Conversation control
    MaxTurns           *int              // Limit conversation turns
    
    // Tool configuration
    AllowedTools       []string          // Tools Claude can use
    DisallowedTools    []string          // Tools Claude cannot use
    
    // Session management
    Resume             *string           // Resume session by ID
    Continue           *bool             // Continue latest session
    
    // Output and logging
    OutputFormat       *OutputFormat     // text, json, stream-json
    Verbose            *bool             // Enable verbose logging
    
    // MCP (Model Context Protocol)
    MCPConfig          *string           // Path to MCP config JSON
    PermissionPromptTool *string         // MCP tool for permissions
    
    // System
    WorkingDirectory   *string           // Working directory
    Executable         *string           // Custom CLI path
}
```

### Output Formats

```go
// Available output formats
claudecode.OutputFormatText       // Plain text (default for Query)
claudecode.OutputFormatJSON       // Structured JSON
claudecode.OutputFormatStreamJSON // Streaming JSON (default for QueryStream)
```

## Advanced Features

### Session Management

```go
// Start a session and get session ID
messages, err := claudecode.QueryWithRequest(ctx, claudecode.QueryRequest{
    Prompt: "Create a function",
    Options: &claudecode.Options{
        OutputFormat: outputFormatPtr(claudecode.OutputFormatJSON),
    },
})

// Extract session ID from result message
var sessionID string
for _, msg := range messages {
    if result, ok := msg.(*claudecode.ResultMessage); ok {
        // Parse session ID from result
        // sessionID = ... 
    }
}

// Resume the session
claudecode.QueryWithRequest(ctx, claudecode.QueryRequest{
    Prompt: "Add tests for that function",
    Options: &claudecode.Options{
        Resume: &sessionID,
    },
})

// Or continue the latest session
claudecode.QueryWithRequest(ctx, claudecode.QueryRequest{
    Prompt: "Optimize the code",
    Options: &claudecode.Options{
        Continue: boolPtr(true),
    },
})
```

### MCP Integration

```go
// mcp-servers.json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path"]
    }
  }
}

// Use MCP tools
request := claudecode.QueryRequest{
    Prompt: "Analyze project files",
    Options: &claudecode.Options{
        MCPConfig:    stringPtr("mcp-servers.json"),
        AllowedTools: []string{"mcp__filesystem__read_file", "mcp__filesystem__list_directory"},
        PermissionPromptTool: stringPtr("mcp__permissions__approve"),
    },
}
```

### Tool Restrictions

```go
options := &claudecode.Options{
    AllowedTools:    []string{"Read", "LS", "Grep"},     // Only these tools
    DisallowedTools: []string{"Bash", "Write"},          // Block these tools
}
```

## API Compatibility

This SDK provides two API styles:

### 1. TypeScript/Python Compatible API (Recommended)

```go
// Single object parameter like TypeScript/Python SDKs
claudecode.QueryWithRequest(ctx, claudecode.QueryRequest{
    Prompt: "...",
    Options: &claudecode.Options{...},
})

claudecode.QueryStreamWithRequest(ctx, claudecode.QueryRequest{
    Prompt: "...",
    Options: &claudecode.Options{...},
})
```

### 2. Traditional Go API (Backward Compatible)

```go
// Multiple parameters (original API)
claudecode.Query(ctx, prompt, options)
claudecode.QueryStream(ctx, prompt, options)
```

## Error Handling

The SDK provides detailed error types:

```go
if err != nil {
    switch e := err.(type) {
    case *claudecode.CLINotFoundError:
        fmt.Printf("Claude CLI not found: %v\n", e)
    case *claudecode.ProcessError:
        fmt.Printf("CLI process error (exit %d): %s\n", e.ExitCode, e.Stderr)
    case *claudecode.CLIConnectionError:
        fmt.Printf("Connection error: %v\n", e)
    case *claudecode.CLIJSONDecodeError:
        fmt.Printf("JSON decode error: %v\n", e)
    default:
        fmt.Printf("Unknown error: %v\n", e)
    }
}
```

## Message Types

The SDK handles all Claude Code message types:

```go
for _, message := range messages {
    switch msg := message.(type) {
    case *claudecode.AssistantMessage:
        fmt.Println("Assistant response")
    case *claudecode.UserMessage:
        fmt.Println("User message")
    case *claudecode.SystemMessage:
        fmt.Println("System message")
    case *claudecode.ResultMessage:
        fmt.Println("Final result")
    }
    
    // Process content blocks
    for _, block := range message.Content() {
        switch b := block.(type) {
        case *claudecode.TextBlock:
            fmt.Printf("Text: %s\n", b.Text)
        case *claudecode.ToolUseBlock:
            fmt.Printf("Tool: %s\n", b.Name)
        case *claudecode.ToolResultBlock:
            fmt.Printf("Result: %v\n", b.Content)
        }
    }
}
```

## Examples

See the [examples](./examples/) directory for complete working examples:

- [`basic/`](./examples/basic/) - Basic usage with both API styles
- [`streaming/`](./examples/streaming/) - Real-time streaming examples  
- [`advanced/`](./examples/advanced/) - MCP, sessions, and advanced features

## Development

### Running Tests

```bash
# Run all tests
go test -v

# Run tests with coverage
go test -v -cover

# Run only unit tests (skip integration tests)
go test -v -short

# Run specific test
go test -v -run TestOptions
```

### Building

```bash
go build
```

### Integration Tests

Integration tests require an authenticated Claude CLI:

```bash
# Ensure Claude CLI is working
claude --help

# Run integration tests
go test -v
```

## Implementation Notes

### CLI Integration

This SDK communicates with Claude Code by:

1. Finding the Claude CLI executable (`claude` command)
2. Executing it with `--print` and appropriate flags
3. Sending prompts via stdin and reading JSON responses from stdout
4. Parsing streaming JSON messages in real-time

### Streaming vs Non-Streaming

- **Query**: Reads all messages at once, suitable for simple requests
- **QueryStream**: Processes messages in real-time as they arrive

### CLI Option Limitations

Some options are only available in `--print` mode (which this SDK uses):
- `SystemPrompt` and `AppendSystemPrompt` ✅ Available
- `MaxTurns` ✅ Available  
- `PermissionPromptTool` ✅ Available

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Related Resources

- [Claude Code Documentation](https://docs.anthropic.com/en/docs/claude-code)
- [Claude Code CLI](https://www.npmjs.com/package/@anthropic-ai/claude-code)
- [Claude Code TypeScript SDK](https://www.npmjs.com/package/@anthropic-ai/claude-code)
- [Claude Code Python SDK](https://pypi.org/project/claude-code-sdk/)