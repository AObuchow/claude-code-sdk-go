package claudecode

import (
	"context"
	"strings"
	"testing"
)

// TestStderrCaptureFailure tests the scenario where the CLI process fails
// and we should capture stderr properly, but currently get an unhelpful error message
func TestStderrCaptureFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	
	// Force an invalid MCP config that will cause the real claude CLI to fail
	// This should reliably trigger a ProcessError with stderr output
	options := &Options{
		MCPConfig: stringPtr("/nonexistent/invalid/mcp/config.json"),
	}

	_, err := Query(ctx, "test prompt", options)
	
	// We expect a ProcessError
	if err == nil {
		t.Fatal("Expected an error but got none")
	}

	processErr, ok := err.(*ProcessError)
	if !ok {
		// If we get a CLINotFoundError, that means claude CLI is not installed
		if _, isNotFound := err.(*CLINotFoundError); isNotFound {
			t.Skip("Claude CLI not found, skipping stderr capture test")
		}
		t.Fatalf("Expected ProcessError, got %T: %v", err, err)
	}

	// This test demonstrates the bug: we should get meaningful stderr content,
	// but instead we get "failed to read stderr"
	t.Logf("Error message: %v", processErr)
	t.Logf("Stderr content: %q", processErr.Stderr)
	
	// After the fix, we should get more helpful error information
	// Either the actual stderr content, or at least context about what went wrong
	if processErr.Stderr == "failed to read stderr" {
		t.Errorf("Still getting the old unhelpful error message: %q", processErr.Stderr)
		t.Error("The fix didn't work - we should get better error information")
	} else {
		// The fix worked - we got either actual stderr or better context
		t.Logf("SUCCESS: Got more helpful error information: %q", processErr.Stderr)
	}
}

// TestStreamingStderrCaptureFailure tests the streaming scenario where CLI process fails
// and we should capture stderr properly from QueryStreamWithRequest  
func TestStreamingStderrCaptureFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	
	// Force an invalid MCP config that will cause the streaming claude CLI to fail
	request := QueryRequest{
		Prompt: "test prompt",
		Options: &Options{
			MCPConfig: stringPtr("/nonexistent/invalid/streaming/mcp/config.json"),
		},
	}

	messageChan, errorChan := QueryStreamWithRequest(ctx, request)
	
	// Read from channels until we get an error
	var finalErr error
	for {
		select {
		case _, ok := <-messageChan:
			if !ok {
				// Channel closed, check if we got an error
				if finalErr == nil {
					t.Fatal("Expected an error but message channel closed without error")
				}
				goto checkError
			}
			// Keep reading messages
			
		case err, ok := <-errorChan:
			if !ok {
				// Error channel closed
				goto checkError
			}
			if err != nil {
				finalErr = err
				goto checkError
			}
		}
	}

checkError:
	if finalErr == nil {
		// If we get a CLINotFoundError, that means claude CLI is not installed
		t.Skip("No error received - this might indicate CLI not found")
	}

	processErr, ok := finalErr.(*ProcessError)
	if !ok {
		// If we get a CLINotFoundError, that means claude CLI is not installed
		if _, isNotFound := finalErr.(*CLINotFoundError); isNotFound {
			t.Skip("Claude CLI not found, skipping streaming stderr capture test")
		}
		t.Fatalf("Expected ProcessError for streaming, got %T: %v", finalErr, finalErr)
	}

	t.Logf("Streaming error message: %v", processErr)
	t.Logf("Streaming stderr content: %q", processErr.Stderr)
	
	// Check if we're getting the problematic "file already closed" error
	if strings.Contains(processErr.Stderr, "file already closed") {
		t.Errorf("Got 'file already closed' error in streaming: %q", processErr.Stderr)
		t.Error("This indicates the stderr pipe timing issue still exists in streaming")
	} else if processErr.Stderr == "failed to read stderr" {
		t.Errorf("Still getting the old unhelpful error message in streaming: %q", processErr.Stderr)
		t.Error("The streaming fix didn't work - we should get better error information")
	} else {
		// The fix worked - we got either actual stderr or better context
		t.Logf("SUCCESS: Streaming got helpful error information: %q", processErr.Stderr)
	}
}


