package claudecode

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// QueryWithRequest executes a query using the TypeScript/Python SDK compatible request format
func QueryWithRequest(ctx context.Context, request QueryRequest) ([]Message, error) {
	return Query(ctx, request.Prompt, request.Options)
}

// Query executes a query against Claude Code and returns the messages
func Query(ctx context.Context, prompt string, options *Options) ([]Message, error) {
	if options == nil {
		options = &Options{}
	}

	// Set environment variable to identify SDK
	os.Setenv("CLAUDE_CODE_ENTRYPOINT", "sdk-go")

	cliPath, err := findCLIExecutable(options.Executable)
	if err != nil {
		return nil, err
	}

	args := buildCommandArgs(options)
	cmd := exec.CommandContext(ctx, cliPath, args...)

	if options.Cwd != nil {
		cmd.Dir = *options.Cwd
	}

	// Set up stdin for the prompt
	cmd.Stdin = strings.NewReader(prompt)

	// Use CombinedOutput for simpler error handling - captures both stdout and stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		// CombinedOutput includes stderr, so we have the actual error message
		if exitError, ok := err.(*exec.ExitError); ok {
			return nil, &ProcessError{
				ExitCode: exitError.ExitCode(),
				Stderr:   string(output), // Contains the actual error output
				Stdout:   "",
			}
		}
		return nil, &CLIConnectionError{
			Message: "CLI process failed",
			Cause:   err,
		}
	}

	// Parse the successful output
	return parseSuccessfulOutput(output, options)
}

func parseSuccessfulOutput(output []byte, options *Options) ([]Message, error) {
	reader := bytes.NewReader(output)
	outputFormat := OutputFormatStreamJSON
	if options.OutputFormat != nil {
		outputFormat = *options.OutputFormat
	}

	if outputFormat == OutputFormatText {
		return readTextOutput(reader)
	}
	return readMessages(reader)
}

func setupCommand(ctx context.Context, options *Options) (*exec.Cmd, error) {
	cliPath, err := findCLIExecutable(options.Executable)
	if err != nil {
		return nil, err
	}

	args := buildCommandArgs(options)
	cmd := exec.CommandContext(ctx, cliPath, args...)

	if options.Cwd != nil {
		cmd.Dir = *options.Cwd
	}

	return cmd, nil
}

func createPipes(cmd *exec.Cmd) (io.WriteCloser, io.ReadCloser, io.ReadCloser, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, &CLIConnectionError{
			Message: "failed to create stdin pipe",
			Cause:   err,
		}
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, &CLIConnectionError{
			Message: "failed to create stdout pipe",
			Cause:   err,
		}
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, &CLIConnectionError{
			Message: "failed to create stderr pipe",
			Cause:   err,
		}
	}

	return stdin, stdout, stderr, nil
}

func readOutput(stdout io.ReadCloser, options *Options) ([]Message, error) {
	outputFormat := OutputFormatStreamJSON
	if options.OutputFormat != nil {
		outputFormat = *options.OutputFormat
	}

	if outputFormat == OutputFormatText {
		return readTextOutput(stdout)
	}
	return readMessages(stdout)
}


// QueryStreamWithRequest executes a streaming query using the TypeScript/Python SDK compatible request format
func QueryStreamWithRequest(ctx context.Context, request QueryRequest) (<-chan Message, <-chan error) {
	return QueryStream(ctx, request.Prompt, request.Options)
}

// QueryStream executes a query against Claude Code and returns a channel of messages
// This provides true streaming by reading messages in real-time
func QueryStream(ctx context.Context, prompt string, options *Options) (<-chan Message, <-chan error) {
	messageChan := make(chan Message, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(messageChan)
		defer close(errorChan)

		if options == nil {
			options = &Options{}
		}

		os.Setenv("CLAUDE_CODE_ENTRYPOINT", "sdk-go")

		streamOptions := prepareStreamOptions(options)
		cmd, err := setupStreamCommand(ctx, &streamOptions)
		if err != nil {
			errorChan <- err
			return
		}

		stdin, stdout, stderr, err := createStreamPipes(cmd, errorChan)
		if err != nil {
			return
		}

		if err := cmd.Start(); err != nil {
			errorChan <- &CLIConnectionError{
				Message: "failed to start Claude CLI",
				Cause:   err,
			}
			return
		}

		go sendPrompt(stdin, prompt)

		if !streamMessages(ctx, stdout, messageChan, errorChan) {
			return
		}

		waitForStreamCommand(cmd, stderr, errorChan)
	}()

	return messageChan, errorChan
}

func prepareStreamOptions(options *Options) Options {
	streamOptions := *options
	streamFormat := OutputFormatStreamJSON
	streamOptions.OutputFormat = &streamFormat
	return streamOptions
}

func setupStreamCommand(ctx context.Context, options *Options) (*exec.Cmd, error) {
	cliPath, err := findCLIExecutable(options.Executable)
	if err != nil {
		return nil, err
	}

	args := buildCommandArgs(options)
	cmd := exec.CommandContext(ctx, cliPath, args...)

	if options.Cwd != nil {
		cmd.Dir = *options.Cwd
	}

	return cmd, nil
}

func createStreamPipes(cmd *exec.Cmd, errorChan chan<- error) (io.WriteCloser, io.ReadCloser, io.ReadCloser, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		errorChan <- &CLIConnectionError{
			Message: "failed to create stdin pipe",
			Cause:   err,
		}
		return nil, nil, nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		errorChan <- &CLIConnectionError{
			Message: "failed to create stdout pipe",
			Cause:   err,
		}
		return nil, nil, nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		errorChan <- &CLIConnectionError{
			Message: "failed to create stderr pipe",
			Cause:   err,
		}
		return nil, nil, nil, err
	}

	return stdin, stdout, stderr, nil
}

func sendPrompt(stdin io.WriteCloser, prompt string) {
	defer stdin.Close()
	_, _ = stdin.Write([]byte(prompt))
}

func streamMessages(ctx context.Context, stdout io.ReadCloser, messageChan chan<- Message, errorChan chan<- error) bool {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var rawMessage map[string]interface{}
		if err := json.Unmarshal([]byte(line), &rawMessage); err != nil {
			errorChan <- &CLIJSONDecodeError{
				Data:  line,
				Cause: err,
			}
			return false
		}

		message, err := parseMessage(rawMessage)
		if err != nil {
			errorChan <- err
			return false
		}

		select {
		case messageChan <- message:
		case <-ctx.Done():
			errorChan <- ctx.Err()
			return false
		}
	}

	if err := scanner.Err(); err != nil {
		errorChan <- &CLIConnectionError{
			Message: "error reading CLI output",
			Cause:   err,
		}
		return false
	}

	return true
}

func waitForStreamCommand(cmd *exec.Cmd, stderr io.ReadCloser, errorChan chan<- error) {
	if err := cmd.Wait(); err != nil {
		stderrBytes, readErr := io.ReadAll(stderr)
		var stderrContent string
		if readErr != nil {
			stderrContent = fmt.Sprintf("failed to read stderr: %v (command error: %v)", readErr, err)
		} else {
			stderrContent = string(stderrBytes)
			if stderrContent == "" {
				stderrContent = fmt.Sprintf("no stderr output (command error: %v)", err)
			}
		}
		if exitError, ok := err.(*exec.ExitError); ok {
			errorChan <- &ProcessError{
				ExitCode: exitError.ExitCode(),
				Stderr:   stderrContent,
				Stdout:   "",
			}
		} else {
			errorChan <- &CLIConnectionError{
				Message: "CLI process failed",
				Cause:   err,
			}
		}
	}
}

// findCLIExecutable finds the Claude Code CLI executable
func findCLIExecutable(customPath *string) (string, error) {
	if customPath != nil && *customPath != "" {
		if _, err := os.Stat(*customPath); err != nil {
			return "", &CLINotFoundError{Path: *customPath}
		}
		return *customPath, nil
	}

	// Try common locations
	candidates := []string{
		"claude",
		"npx @anthropic-ai/claude-code",
	}

	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}

	// Try npm global installation path
	if npmPath, err := exec.LookPath("npm"); err == nil {
		cmd := exec.Command(npmPath, "root", "-g")
		output, err := cmd.Output()
		if err == nil {
			globalPath := strings.TrimSpace(string(output))
			claudePath := filepath.Join(globalPath, "@anthropic-ai", "claude-code", "bin", "claude")
			if _, err := os.Stat(claudePath); err == nil {
				return claudePath, nil
			}
		}
	}

	return "", &CLINotFoundError{}
}

// buildCommandArgs builds CLI arguments from options
func buildCommandArgs(options *Options) []string {
	args := []string{"--print"}

	args = addPromptArgs(args, options)
	args = addModelAndToolArgs(args, options)
	args = addSessionArgs(args, options)
	args = addOutputArgs(args, options)
	args = addMCPArgs(args, options)
	args = addPermissionArgs(args, options)
	args = addMiscArgs(args, options)

	return args
}

func addPromptArgs(args []string, options *Options) []string {
	if options.SystemPrompt != nil && *options.SystemPrompt != "" {
		args = append(args, "--system-prompt", *options.SystemPrompt)
	}
	if options.AppendSystemPrompt != nil && *options.AppendSystemPrompt != "" {
		args = append(args, "--append-system-prompt", *options.AppendSystemPrompt)
	}
	if options.MaxTurns != nil {
		args = append(args, "--max-turns", fmt.Sprintf("%d", *options.MaxTurns))
	}
	return args
}

func addModelAndToolArgs(args []string, options *Options) []string {
	if options.Model != nil && *options.Model != "" {
		args = append(args, "--model", *options.Model)
	}
	if len(options.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(options.AllowedTools, ","))
	}
	if len(options.DisallowedTools) > 0 {
		args = append(args, "--disallowedTools", strings.Join(options.DisallowedTools, ","))
	}
	return args
}

func addSessionArgs(args []string, options *Options) []string {
	if options.Resume != nil && *options.Resume != "" {
		args = append(args, "--resume", *options.Resume)
	}
	if options.Continue != nil && *options.Continue {
		args = append(args, "--continue")
	}
	return args
}

func addOutputArgs(args []string, options *Options) []string {
	outputFormat := OutputFormatStreamJSON
	if options.OutputFormat != nil {
		outputFormat = *options.OutputFormat
	}
	args = append(args, "--output-format", string(outputFormat))

	if options.Verbose != nil && *options.Verbose {
		args = append(args, "--verbose")
	} else if outputFormat == OutputFormatStreamJSON {
		args = append(args, "--verbose")
	}
	return args
}

func addMCPArgs(args []string, options *Options) []string {
	if options.MCPConfig != nil && *options.MCPConfig != "" {
		args = append(args, "--mcp-config", *options.MCPConfig)
	}
	if len(options.MCPServers) > 0 {
		mcpConfig := map[string]interface{}{
			"mcpServers": options.MCPServers,
		}
		configJSON, err := json.Marshal(mcpConfig)
		if err == nil {
			args = append(args, "--mcp-config", string(configJSON))
		}
	}
	return args
}

func addPermissionArgs(args []string, options *Options) []string {
	if options.PermissionMode != nil && *options.PermissionMode != "" {
		args = append(args, "--permission-mode", *options.PermissionMode)
	}
	if options.PermissionPromptTool != nil && *options.PermissionPromptTool != "" {
		args = append(args, "--permission-prompt-tool", *options.PermissionPromptTool)
	}
	if options.DangerouslySkipPermissions != nil && *options.DangerouslySkipPermissions {
		args = append(args, "--dangerously-skip-permissions")
	}
	return args
}

func addMiscArgs(args []string, options *Options) []string {
	if options.Debug != nil && *options.Debug {
		args = append(args, "--debug")
	}
	if options.InputFormat != nil && *options.InputFormat != "" {
		args = append(args, "--input-format", *options.InputFormat)
	}
	if len(options.AddDir) > 0 {
		for _, dir := range options.AddDir {
			args = append(args, "--add-dir", dir)
		}
	}
	return args
}

// readTextOutput reads plain text output and creates a single result message
func readTextOutput(reader io.Reader) ([]Message, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, &CLIConnectionError{
			Message: "error reading text output",
			Cause:   err,
		}
	}

	// Create a single result message with the text content
	resultText := string(content)
	message := &ResultMessage{
		Subtype:   "text_output",
		Result:    &resultText,
		SessionID: "text_output_session",
		CreatedAt: time.Now(),
	}

	return []Message{message}, nil
}

// readMessages reads and parses messages from the CLI output
func readMessages(reader io.Reader) ([]Message, error) {
	var messages []Message
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse JSON message
		var rawMessage map[string]interface{}
		if err := json.Unmarshal([]byte(line), &rawMessage); err != nil {
			return nil, &CLIJSONDecodeError{
				Data:  line,
				Cause: err,
			}
		}

		message, err := parseMessage(rawMessage)
		if err != nil {
			return nil, err
		}

		messages = append(messages, message)
	}

	if err := scanner.Err(); err != nil {
		return nil, &CLIConnectionError{
			Message: "error reading CLI output",
			Cause:   err,
		}
	}

	return messages, nil
}

// parseMessage parses a raw message map into a Message interface
func parseMessage(rawMessage map[string]interface{}) (Message, error) {
	messageType, ok := rawMessage["type"].(string)
	if !ok {
		return nil, &CLIJSONDecodeError{
			Data:  fmt.Sprintf("%v", rawMessage),
			Cause: fmt.Errorf("missing or invalid message type"),
		}
	}

	timestamp := parseTimestamp(rawMessage)
	sessionID, parentToolUseIDPtr := parseCommonFields(rawMessage)

	switch MessageType(messageType) {
	case "system":
		return parseSystemMessage(rawMessage, sessionID, timestamp)
	case MessageTypeAssistant:
		return parseAssistantMessage(rawMessage, sessionID, parentToolUseIDPtr, timestamp)
	case MessageTypeUser:
		return parseUserMessage(rawMessage, sessionID, parentToolUseIDPtr, timestamp)
	case "result":
		return parseResultMessage(rawMessage, sessionID, timestamp)
	default:
		return &SystemMessage{
			Subtype:   messageType,
			SessionID: sessionID,
			CreatedAt: timestamp,
		}, nil
	}
}

func parseTimestamp(rawMessage map[string]interface{}) time.Time {
	timestamp := time.Now()
	if ts, ok := rawMessage["timestamp"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
			timestamp = parsed
		}
	}
	return timestamp
}

func parseCommonFields(rawMessage map[string]interface{}) (string, *string) {
	sessionID, _ := rawMessage["session_id"].(string)
	parentToolUseID, _ := rawMessage["parent_tool_use_id"].(string)
	var parentToolUseIDPtr *string
	if parentToolUseID != "" {
		parentToolUseIDPtr = &parentToolUseID
	}
	return sessionID, parentToolUseIDPtr
}

func parseSystemMessage(rawMessage map[string]interface{}, sessionID string, timestamp time.Time) (Message, error) {
	subtype, _ := rawMessage["subtype"].(string)

	return &SystemMessage{
		Subtype:        subtype,
		APIKeySource:   parseStringPtr(rawMessage, "apiKeySource"),
		Cwd:            parseStringPtr(rawMessage, "cwd"),
		SessionID:      sessionID,
		Tools:          parseToolsArray(rawMessage),
		MCPServers:     parseMCPServers(rawMessage),
		Model:          parseStringPtr(rawMessage, "model"),
		PermissionMode: parseStringPtr(rawMessage, "permissionMode"),
		CreatedAt:      timestamp,
	}, nil
}

func parseAssistantMessage(rawMessage map[string]interface{}, sessionID string, parentToolUseIDPtr *string, timestamp time.Time) (Message, error) {
	contentBlocks, err := parseMessageContent(rawMessage)
	if err != nil {
		return nil, err
	}
	return &AssistantMessage{
		ContentBlocks:   contentBlocks,
		ParentToolUseID: parentToolUseIDPtr,
		SessionID:       sessionID,
		CreatedAt:       timestamp,
	}, nil
}

func parseUserMessage(rawMessage map[string]interface{}, sessionID string, parentToolUseIDPtr *string, timestamp time.Time) (Message, error) {
	contentBlocks, err := parseMessageContent(rawMessage)
	if err != nil {
		return nil, err
	}
	return &UserMessage{
		ContentBlocks:   contentBlocks,
		ParentToolUseID: parentToolUseIDPtr,
		SessionID:       sessionID,
		CreatedAt:       timestamp,
	}, nil
}

func parseResultMessage(rawMessage map[string]interface{}, sessionID string, timestamp time.Time) (Message, error) {
	subtype, _ := rawMessage["subtype"].(string)
	durationMs, _ := rawMessage["duration_ms"].(float64)
	durationAPIMs, _ := rawMessage["duration_api_ms"].(float64)
	isError, _ := rawMessage["is_error"].(bool)
	numTurns, _ := rawMessage["num_turns"].(float64)
	totalCostUSD, _ := rawMessage["total_cost_usd"].(float64)

	var totalCostUSDPtr *float64
	if totalCostUSD > 0 {
		totalCostUSDPtr = &totalCostUSD
	}

	var resultPtr *string
	if result, ok := rawMessage["result"]; ok {
		resultStr := fmt.Sprintf("%v", result)
		resultPtr = &resultStr
	}

	return &ResultMessage{
		Subtype:       subtype,
		DurationMs:    int(durationMs),
		DurationAPIMs: int(durationAPIMs),
		IsError:       isError,
		NumTurns:      int(numTurns),
		SessionID:     sessionID,
		TotalCostUSD:  totalCostUSDPtr,
		Usage:         parseUsage(rawMessage),
		Result:        resultPtr,
		CreatedAt:     timestamp,
	}, nil
}

func parseStringPtr(rawMessage map[string]interface{}, key string) *string {
	if value, ok := rawMessage[key].(string); ok && value != "" {
		return &value
	}
	return nil
}

func parseToolsArray(rawMessage map[string]interface{}) []string {
	var tools []string
	if toolsData, ok := rawMessage["tools"]; ok {
		if toolsArray, ok := toolsData.([]interface{}); ok {
			for _, tool := range toolsArray {
				if toolStr, ok := tool.(string); ok {
					tools = append(tools, toolStr)
				}
			}
		}
	}
	return tools
}

func parseMCPServers(rawMessage map[string]interface{}) []MCPServer {
	var mcpServers []MCPServer
	if mcpData, ok := rawMessage["mcp_servers"]; ok {
		if mcpArray, ok := mcpData.([]interface{}); ok {
			for _, server := range mcpArray {
				if serverMap, ok := server.(map[string]interface{}); ok {
					name, _ := serverMap["name"].(string)
					status, _ := serverMap["status"].(string)
					mcpServers = append(mcpServers, MCPServer{Name: name, Status: status})
				}
			}
		}
	}
	return mcpServers
}

func parseMessageContent(rawMessage map[string]interface{}) ([]ContentBlock, error) {
	var contentBlocks []ContentBlock
	if msgData, ok := rawMessage["message"]; ok {
		if msgMap, ok := msgData.(map[string]interface{}); ok {
			if content, ok := msgMap["content"]; ok {
				return parseContentBlocks(content)
			}
		}
	}
	return contentBlocks, nil
}

func parseUsage(rawMessage map[string]interface{}) *Usage {
	if usageData, ok := rawMessage["usage"]; ok {
		if usageMap, ok := usageData.(map[string]interface{}); ok {
			inputTokens, _ := usageMap["input_tokens"].(float64)
			outputTokens, _ := usageMap["output_tokens"].(float64)
			return &Usage{
				InputTokens:  int(inputTokens),
				OutputTokens: int(outputTokens),
			}
		}
	}
	return nil
}

// parseContentBlocks parses content blocks from raw JSON
func parseContentBlocks(rawContent interface{}) ([]ContentBlock, error) {
	var blocks []ContentBlock

	switch content := rawContent.(type) {
	case string:
		// Simple text content
		blocks = append(blocks, &TextBlock{Text: content})
	case []interface{}:
		// Array of content blocks
		for _, rawBlock := range content {
			block, err := parseContentBlock(rawBlock)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, block)
		}
	case map[string]interface{}:
		// Single content block
		block, err := parseContentBlock(content)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	default:
		return nil, &CLIJSONDecodeError{
			Data:  fmt.Sprintf("%v", rawContent),
			Cause: fmt.Errorf("invalid content format"),
		}
	}

	return blocks, nil
}

// parseContentBlock parses a single content block
func parseContentBlock(rawBlock interface{}) (ContentBlock, error) {
	blockMap, ok := rawBlock.(map[string]interface{})
	if !ok {
		// If it's not a map, treat it as text
		if str, isString := rawBlock.(string); isString {
			return &TextBlock{Text: str}, nil
		}
		return nil, &CLIJSONDecodeError{
			Data:  fmt.Sprintf("%v", rawBlock),
			Cause: fmt.Errorf("invalid content block format"),
		}
	}

	blockType, ok := blockMap["type"].(string)
	if !ok {
		// If no type specified, check for common fields
		if text, ok := blockMap["text"].(string); ok {
			return &TextBlock{Text: text}, nil
		}
		return nil, &CLIJSONDecodeError{
			Data:  fmt.Sprintf("%v", rawBlock),
			Cause: fmt.Errorf("missing content block type"),
		}
	}

	switch ContentBlockType(blockType) {
	case ContentBlockTypeText:
		text, ok := blockMap["text"].(string)
		if !ok {
			return nil, &CLIJSONDecodeError{
				Data:  fmt.Sprintf("%v", rawBlock),
				Cause: fmt.Errorf("missing text in text block"),
			}
		}
		return &TextBlock{Text: text}, nil

	case ContentBlockTypeToolUse:
		id, _ := blockMap["id"].(string)
		name, _ := blockMap["name"].(string)
		input, _ := blockMap["input"].(map[string]interface{})
		return &ToolUseBlock{
			ID:    id,
			Name:  name,
			Input: input,
		}, nil

	case ContentBlockTypeToolResult:
		toolUseID, _ := blockMap["tool_use_id"].(string)
		content := blockMap["content"]
		isError, _ := blockMap["is_error"].(bool)
		return &ToolResultBlock{
			ToolUseID: toolUseID,
			Content:   content,
			IsError:   isError,
		}, nil

	default:
		return nil, &CLIJSONDecodeError{
			Data:  fmt.Sprintf("%v", rawBlock),
			Cause: fmt.Errorf("unknown content block type: %s", blockType),
		}
	}
}
