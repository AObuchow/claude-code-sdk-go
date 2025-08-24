package claudecode

import (
	"context"
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


