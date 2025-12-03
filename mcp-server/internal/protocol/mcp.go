package protocol

// MCP Protocol Types
// Based on Model Context Protocol specification

// InitializeRequest is sent by the client to initialize the MCP session
type InitializeRequest struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    ClientCapabilities     `json:"capabilities"`
	ClientInfo      ClientInfo             `json:"clientInfo"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ClientInfo contains information about the client
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientCapabilities describes what the client can do
type ClientCapabilities struct {
	Tools     *ToolCapabilities     `json:"tools,omitempty"`
	Resources *ResourceCapabilities `json:"resources,omitempty"`
	Prompts   *PromptCapabilities   `json:"prompts,omitempty"`
}

// ToolCapabilities describes tool-related capabilities
type ToolCapabilities struct {
	SupportsProgress bool `json:"supportsProgress,omitempty"`
}

// ResourceCapabilities describes resource-related capabilities
type ResourceCapabilities struct {
	SupportsSubscribe bool `json:"supportsSubscribe,omitempty"`
}

// PromptCapabilities describes prompt-related capabilities
type PromptCapabilities struct {
	SupportsTemplates bool `json:"supportsTemplates,omitempty"`
}

// InitializeResult is the response to an initialize request
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

// ServerInfo contains information about the server
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerCapabilities describes what the server can do
type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
}

// ToolsCapability indicates the server supports tools
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"` // Server will notify when tool list changes
}

// ResourcesCapability indicates the server supports resources
type ResourcesCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
	Subscribe   bool `json:"subscribe,omitempty"`
}

// PromptsCapability indicates the server supports prompts
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// Tool represents an MCP tool that can be called
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolsListResult is the response to tools/list
type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

// ToolCallRequest is the request to call a tool
type ToolCallRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ToolCallResult is the response from a tool call
type ToolCallResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock represents a piece of content in a response
type ContentBlock struct {
	Type string `json:"type"` // "text", "image", "resource"
	Text string `json:"text,omitempty"`
	Data string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

// Resource represents an MCP resource
type Resource struct {
	URI         string                 `json:"uri"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	MimeType    string                 `json:"mimeType,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ResourcesListResult is the response to resources/list
type ResourcesListResult struct {
	Resources []Resource `json:"resources"`
}

// ResourceReadRequest is the request to read a resource
type ResourceReadRequest struct {
	URI string `json:"uri"`
}

// ResourceReadResult is the response from reading a resource
type ResourceReadResult struct {
	Contents []ResourceContents `json:"contents"`
}

// ResourceContents represents the contents of a resource
type ResourceContents struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"` // base64 encoded
}

// Prompt represents an MCP prompt template
type Prompt struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	Arguments   []PromptArgument         `json:"arguments,omitempty"`
}

// PromptArgument describes an argument to a prompt
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// PromptsListResult is the response to prompts/list
type PromptsListResult struct {
	Prompts []Prompt `json:"prompts"`
}

// PromptGetRequest is the request to get a prompt
type PromptGetRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// PromptGetResult is the response from getting a prompt
type PromptGetResult struct {
	Messages []PromptMessage `json:"messages"`
}

// PromptMessage represents a message in a prompt
type PromptMessage struct {
	Role    string         `json:"role"` // "user", "assistant", "system"
	Content ContentBlock   `json:"content"`
}

// Progress notification
type ProgressNotification struct {
	ProgressToken string  `json:"progressToken"`
	Progress      float64 `json:"progress"` // 0.0 to 1.0
	Total         float64 `json:"total,omitempty"`
}

// MCP Method Names
const (
	MethodInitialize    = "initialize"
	MethodInitialized   = "notifications/initialized"
	MethodToolsList     = "tools/list"
	MethodToolsCall     = "tools/call"
	MethodResourcesList = "resources/list"
	MethodResourcesRead = "resources/read"
	MethodPromptsList   = "prompts/list"
	MethodPromptsGet    = "prompts/get"
	MethodProgress      = "notifications/progress"
)
