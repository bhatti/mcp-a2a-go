package tools

import (
	"context"
	"fmt"

	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/protocol"
)

// Tool represents an MCP tool that can be executed
type Tool interface {
	// Definition returns the MCP tool definition
	Definition() protocol.Tool
	// Execute runs the tool with the given arguments
	Execute(ctx context.Context, args map[string]interface{}) (protocol.ToolCallResult, error)
}

// Registry manages available tools
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register registers a new tool
func (r *Registry) Register(tool Tool) {
	def := tool.Definition()
	r.tools[def.Name] = tool
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// List returns all registered tools
func (r *Registry) List() []protocol.Tool {
	tools := make([]protocol.Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool.Definition())
	}
	return tools
}

// Execute executes a tool by name
func (r *Registry) Execute(ctx context.Context, name string, args map[string]interface{}) (protocol.ToolCallResult, error) {
	tool, ok := r.Get(name)
	if !ok {
		return protocol.ToolCallResult{
			IsError: true,
		}, fmt.Errorf("tool not found: %s", name)
	}

	return tool.Execute(ctx, args)
}
