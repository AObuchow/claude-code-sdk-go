package claudecode

import "time"

// MessageType represents the type of message
type MessageType string

const (
	MessageTypeAssistant MessageType = "assistant"
	MessageTypeUser      MessageType = "user"
	MessageTypeSystem    MessageType = "system"
	MessageTypeResult    MessageType = "result"
)

// Message represents a message in the conversation
type Message interface {
	Type() MessageType
	Content() []ContentBlock
	Timestamp() time.Time
}

// ContentBlockType represents the type of content block
type ContentBlockType string

const (
	ContentBlockTypeText       ContentBlockType = "text"
	ContentBlockTypeToolUse    ContentBlockType = "tool_use"
	ContentBlockTypeToolResult ContentBlockType = "tool_result"
	ContentBlockTypeThinking   ContentBlockType = "thinking"
)

// ContentBlock represents a block of content within a message
type ContentBlock interface {
	Type() ContentBlockType
}

// TextBlock represents a text content block
type TextBlock struct {
	Text string `json:"text"`
}

func (t *TextBlock) Type() ContentBlockType {
	return ContentBlockTypeText
}

// ToolUseBlock represents a tool use content block
type ToolUseBlock struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

func (t *ToolUseBlock) Type() ContentBlockType {
	return ContentBlockTypeToolUse
}

// ToolResultBlock represents a tool result content block
type ToolResultBlock struct {
	ToolUseID string      `json:"tool_use_id"`
	Content   interface{} `json:"content"`
	IsError   bool        `json:"is_error,omitempty"`
}

func (t *ToolResultBlock) Type() ContentBlockType {
	return ContentBlockTypeToolResult
}

// ThinkingBlock represents a thinking content block
type ThinkingBlock struct {
	Thinking  string `json:"thinking"`
	Signature string `json:"signature,omitempty"`
}

func (t *ThinkingBlock) Type() ContentBlockType {
	return ContentBlockTypeThinking
}

// AssistantMessage represents a message from the assistant
type AssistantMessage struct {
	ContentBlocks   []ContentBlock `json:"content"`
	ParentToolUseID *string        `json:"parent_tool_use_id,omitempty"`
	SessionID       string         `json:"session_id"`
	CreatedAt       time.Time      `json:"created_at"`
}

func (m *AssistantMessage) Type() MessageType {
	return MessageTypeAssistant
}

func (m *AssistantMessage) Content() []ContentBlock {
	return m.ContentBlocks
}

func (m *AssistantMessage) Timestamp() time.Time {
	return m.CreatedAt
}

// UserMessage represents a message from the user
type UserMessage struct {
	ContentBlocks   []ContentBlock `json:"content"`
	ParentToolUseID *string        `json:"parent_tool_use_id,omitempty"`
	SessionID       string         `json:"session_id"`
	CreatedAt       time.Time      `json:"created_at"`
}

func (m *UserMessage) Type() MessageType {
	return MessageTypeUser
}

func (m *UserMessage) Content() []ContentBlock {
	return m.ContentBlocks
}

func (m *UserMessage) Timestamp() time.Time {
	return m.CreatedAt
}

// SystemMessage represents a system message
type SystemMessage struct {
	Subtype        string      `json:"subtype"`
	APIKeySource   *string     `json:"apiKeySource,omitempty"`
	Cwd            *string     `json:"cwd,omitempty"`
	SessionID      string      `json:"session_id"`
	Tools          []string    `json:"tools,omitempty"`
	MCPServers     []MCPServer `json:"mcp_servers,omitempty"`
	Model          *string     `json:"model,omitempty"`
	PermissionMode *string     `json:"permissionMode,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
}

func (m *SystemMessage) Type() MessageType {
	return MessageTypeSystem
}

func (m *SystemMessage) Content() []ContentBlock {
	// SystemMessage doesn't have content blocks in the official SDK format
	return []ContentBlock{}
}

func (m *SystemMessage) Timestamp() time.Time {
	return m.CreatedAt
}

// ResultMessage represents a result message
type ResultMessage struct {
	Subtype       string    `json:"subtype"`
	DurationMs    int       `json:"duration_ms"`
	DurationAPIMs int       `json:"duration_api_ms"`
	IsError       bool      `json:"is_error"`
	NumTurns      int       `json:"num_turns"`
	SessionID     string    `json:"session_id"`
	TotalCostUSD  *float64  `json:"total_cost_usd,omitempty"`
	Usage         *Usage    `json:"usage,omitempty"`
	Result        *string   `json:"result,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func (m *ResultMessage) Type() MessageType {
	return MessageTypeResult
}

func (m *ResultMessage) Content() []ContentBlock {
	// ResultMessage doesn't have content blocks in the official SDK format
	// Result content is in the Result field
	if m.Result != nil {
		return []ContentBlock{&TextBlock{Text: *m.Result}}
	}
	return []ContentBlock{}
}

func (m *ResultMessage) Timestamp() time.Time {
	return m.CreatedAt
}

// OutputFormat represents the output format for Claude Code queries
type OutputFormat string

const (
	OutputFormatText       OutputFormat = "text"
	OutputFormatJSON       OutputFormat = "json"
	OutputFormatStreamJSON OutputFormat = "stream-json"
)

// McpServerConfig represents MCP server configuration
type McpServerConfig struct {
	Transport []string               `json:"transport"`
	Env       map[string]interface{} `json:"env,omitempty"`
}

// MCPServer represents an MCP server status
type MCPServer struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// Usage represents API usage information
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// QueryRequest represents a query request compatible with TypeScript/Python SDKs
type QueryRequest struct {
	// Prompt is the query to send to Claude Code
	Prompt string `json:"prompt"`

	// Options contains configuration options for the query
	Options *Options `json:"options,omitempty"`
}

// Options represents configuration options for Claude Code queries
type Options struct {
	// Core behavior options
	// Model specifies the model to use (e.g., 'sonnet', 'opus', or full model name)
	Model *string `json:"model,omitempty"`

	// SystemPrompt sets a custom system prompt to guide Claude's behavior
	SystemPrompt *string `json:"system_prompt,omitempty"`

	// AppendSystemPrompt appends to the default system prompt
	AppendSystemPrompt *string `json:"append_system_prompt,omitempty"`

	// MaxTurns limits the number of conversation turns
	MaxTurns *int `json:"max_turns,omitempty"`

	// Session management
	// Continue indicates whether to continue the latest session
	Continue *bool `json:"continue,omitempty"`

	// Resume specifies a session ID to resume
	Resume *string `json:"resume,omitempty"`

	// Tool configuration
	// AllowedTools specifies which tools Claude can use (comma or space-separated)
	AllowedTools []string `json:"allowed_tools,omitempty"`

	// DisallowedTools specifies which tools Claude cannot use (comma or space-separated)
	DisallowedTools []string `json:"disallowed_tools,omitempty"`

	// MCP (Model Context Protocol) configuration
	// MCPTools specifies MCP tools to use
	MCPTools []string `json:"mcp_tools,omitempty"`

	// MCPServers specifies MCP server configurations
	MCPServers map[string]McpServerConfig `json:"mcp_servers,omitempty"`

	// MCPConfig specifies the path to MCP server configuration JSON file or JSON string
	MCPConfig *string `json:"mcp_config,omitempty"`

	// Permission and security
	// PermissionMode defines the interaction permission level
	// Options: "default", "acceptEdits", "bypassPermissions", "plan"
	PermissionMode *string `json:"permission_mode,omitempty"`

	// PermissionPromptTool specifies the MCP tool to use for permission prompts
	PermissionPromptTool *string `json:"permission_prompt_tool,omitempty"`

	// DangerouslySkipPermissions bypasses all permission checks
	// Recommended only for sandboxes with no internet access
	DangerouslySkipPermissions *bool `json:"dangerously_skip_permissions,omitempty"`

	// Directory and environment
	// Cwd sets the working directory for Claude Code
	Cwd *string `json:"cwd,omitempty"`

	// AddDir specifies additional directories to allow tool access to
	AddDir []string `json:"add_dir,omitempty"`

	// I/O format options
	// InputFormat specifies the input format: "text" (default) or "stream-json"
	InputFormat *string `json:"input_format,omitempty"`

	// OutputFormat specifies the output format: "text", "json", or "stream-json"
	OutputFormat *OutputFormat `json:"output_format,omitempty"`

	// Debug and logging
	// Debug enables debug mode (shows MCP server errors)
	Debug *bool `json:"debug,omitempty"`

	// Verbose enables verbose logging (automatically enabled for stream-json output)
	Verbose *bool `json:"verbose,omitempty"`

	// SDK-specific options
	// AbortController allows cancellation of the query (Go context handles this)
	// This field is not used directly but kept for API compatibility
	AbortController interface{} `json:"abort_controller,omitempty"`

	// Executable specifies a custom path to the Claude Code CLI
	Executable *string `json:"executable,omitempty"`
}
