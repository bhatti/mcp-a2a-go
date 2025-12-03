package protocol

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeRequestMarshaling(t *testing.T) {
	initReq := InitializeRequest{
		ProtocolVersion: "2024-11-05",
		Capabilities: ClientCapabilities{
			Tools: &ToolCapabilities{
				SupportsProgress: true,
			},
			Resources: &ResourceCapabilities{
				SupportsSubscribe: true,
			},
		},
		ClientInfo: ClientInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
		Metadata: map[string]interface{}{
			"platform": "linux",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(initReq)
	require.NoError(t, err)

	// Unmarshal back
	var decoded InitializeRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, initReq.ProtocolVersion, decoded.ProtocolVersion)
	assert.Equal(t, initReq.ClientInfo.Name, decoded.ClientInfo.Name)
	assert.NotNil(t, decoded.Capabilities.Tools)
	assert.True(t, decoded.Capabilities.Tools.SupportsProgress)
}

func TestInitializeResultMarshaling(t *testing.T) {
	initResult := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{
				ListChanged: false,
			},
			Resources: &ResourcesCapability{
				ListChanged: true,
				Subscribe:   true,
			},
		},
		ServerInfo: ServerInfo{
			Name:    "mcp-rag-server",
			Version: "1.0.0",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(initResult)
	require.NoError(t, err)

	// Unmarshal back
	var decoded InitializeResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, initResult.ProtocolVersion, decoded.ProtocolVersion)
	assert.Equal(t, initResult.ServerInfo.Name, decoded.ServerInfo.Name)
	assert.NotNil(t, decoded.Capabilities.Tools)
	assert.False(t, decoded.Capabilities.Tools.ListChanged)
}

func TestToolMarshaling(t *testing.T) {
	tool := Tool{
		Name:        "search_documents",
		Description: "Search documents by text query",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The search query",
				},
			},
			"required": []string{"query"},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(tool)
	require.NoError(t, err)

	// Unmarshal back
	var decoded Tool
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, tool.Name, decoded.Name)
	assert.Equal(t, tool.Description, decoded.Description)
	assert.NotNil(t, decoded.InputSchema)
	assert.Equal(t, "object", decoded.InputSchema["type"])
}

func TestToolsListResult(t *testing.T) {
	result := ToolsListResult{
		Tools: []Tool{
			{
				Name:        "tool1",
				Description: "First tool",
				InputSchema: map[string]interface{}{
					"type": "object",
				},
			},
			{
				Name:        "tool2",
				Description: "Second tool",
				InputSchema: map[string]interface{}{
					"type": "object",
				},
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(result)
	require.NoError(t, err)

	// Unmarshal back
	var decoded ToolsListResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.Tools, 2)
	assert.Equal(t, "tool1", decoded.Tools[0].Name)
	assert.Equal(t, "tool2", decoded.Tools[1].Name)
}

func TestToolCallRequest(t *testing.T) {
	tests := []struct {
		name     string
		request  ToolCallRequest
		validate func(t *testing.T, decoded ToolCallRequest)
	}{
		{
			name: "with arguments",
			request: ToolCallRequest{
				Name: "search_documents",
				Arguments: map[string]interface{}{
					"query": "test",
					"limit": 10,
				},
			},
			validate: func(t *testing.T, decoded ToolCallRequest) {
				assert.Equal(t, "search_documents", decoded.Name)
				assert.NotNil(t, decoded.Arguments)
				assert.Equal(t, "test", decoded.Arguments["query"])
				assert.Equal(t, float64(10), decoded.Arguments["limit"])
			},
		},
		{
			name: "without arguments",
			request: ToolCallRequest{
				Name:      "list_documents",
				Arguments: nil,
			},
			validate: func(t *testing.T, decoded ToolCallRequest) {
				assert.Equal(t, "list_documents", decoded.Name)
				assert.Nil(t, decoded.Arguments)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.request)
			require.NoError(t, err)

			// Unmarshal back
			var decoded ToolCallRequest
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			tt.validate(t, decoded)
		})
	}
}

func TestToolCallResult(t *testing.T) {
	tests := []struct {
		name   string
		result ToolCallResult
	}{
		{
			name: "successful result",
			result: ToolCallResult{
				Content: []ContentBlock{
					{
						Type: "text",
						Text: "Found 3 documents",
					},
				},
				IsError: false,
			},
		},
		{
			name: "error result",
			result: ToolCallResult{
				Content: []ContentBlock{
					{
						Type: "text",
						Text: "Document not found",
					},
				},
				IsError: true,
			},
		},
		{
			name: "multiple content blocks",
			result: ToolCallResult{
				Content: []ContentBlock{
					{
						Type: "text",
						Text: "Result summary",
					},
					{
						Type:     "image",
						Data:     "base64data",
						MimeType: "image/png",
					},
				},
				IsError: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.result)
			require.NoError(t, err)

			// Unmarshal back
			var decoded ToolCallResult
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, tt.result.IsError, decoded.IsError)
			assert.Len(t, decoded.Content, len(tt.result.Content))
			for i, block := range tt.result.Content {
				assert.Equal(t, block.Type, decoded.Content[i].Type)
				assert.Equal(t, block.Text, decoded.Content[i].Text)
			}
		})
	}
}

func TestContentBlock(t *testing.T) {
	tests := []struct {
		name  string
		block ContentBlock
	}{
		{
			name: "text content",
			block: ContentBlock{
				Type: "text",
				Text: "Sample text content",
			},
		},
		{
			name: "image content",
			block: ContentBlock{
				Type:     "image",
				Data:     "base64encodeddata",
				MimeType: "image/jpeg",
			},
		},
		{
			name: "resource content",
			block: ContentBlock{
				Type:     "resource",
				Data:     "resource data",
				MimeType: "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.block)
			require.NoError(t, err)

			// Unmarshal back
			var decoded ContentBlock
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, tt.block.Type, decoded.Type)
			assert.Equal(t, tt.block.Text, decoded.Text)
			assert.Equal(t, tt.block.Data, decoded.Data)
			assert.Equal(t, tt.block.MimeType, decoded.MimeType)
		})
	}
}

func TestResource(t *testing.T) {
	resource := Resource{
		URI:         "documents://acme-corp/policy-001",
		Name:        "Security Policy",
		Description: "Company security policy document",
		MimeType:    "text/markdown",
		Metadata: map[string]interface{}{
			"version":    "2024.1",
			"department": "security",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(resource)
	require.NoError(t, err)

	// Unmarshal back
	var decoded Resource
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resource.URI, decoded.URI)
	assert.Equal(t, resource.Name, decoded.Name)
	assert.Equal(t, resource.Description, decoded.Description)
	assert.Equal(t, resource.MimeType, decoded.MimeType)
	assert.NotNil(t, decoded.Metadata)
}

func TestResourcesListResult(t *testing.T) {
	result := ResourcesListResult{
		Resources: []Resource{
			{
				URI:  "doc://1",
				Name: "Document 1",
			},
			{
				URI:  "doc://2",
				Name: "Document 2",
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(result)
	require.NoError(t, err)

	// Unmarshal back
	var decoded ResourcesListResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.Resources, 2)
	assert.Equal(t, "doc://1", decoded.Resources[0].URI)
}

func TestResourceReadRequest(t *testing.T) {
	request := ResourceReadRequest{
		URI: "documents://acme-corp/doc-123",
	}

	// Marshal to JSON
	data, err := json.Marshal(request)
	require.NoError(t, err)

	// Unmarshal back
	var decoded ResourceReadRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, request.URI, decoded.URI)
}

func TestResourceReadResult(t *testing.T) {
	result := ResourceReadResult{
		Contents: []ResourceContents{
			{
				URI:      "doc://1",
				MimeType: "text/plain",
				Text:     "Document content",
			},
			{
				URI:      "doc://2",
				MimeType: "application/octet-stream",
				Blob:     "YmFzZTY0ZGF0YQ==", // base64 for "base64data"
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(result)
	require.NoError(t, err)

	// Unmarshal back
	var decoded ResourceReadResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.Contents, 2)
	assert.Equal(t, "Document content", decoded.Contents[0].Text)
	assert.Equal(t, "YmFzZTY0ZGF0YQ==", decoded.Contents[1].Blob)
}

func TestPrompt(t *testing.T) {
	prompt := Prompt{
		Name:        "code_review",
		Description: "Generate a code review",
		Arguments: []PromptArgument{
			{
				Name:        "code",
				Description: "The code to review",
				Required:    true,
			},
			{
				Name:        "language",
				Description: "Programming language",
				Required:    false,
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(prompt)
	require.NoError(t, err)

	// Unmarshal back
	var decoded Prompt
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, prompt.Name, decoded.Name)
	assert.Len(t, decoded.Arguments, 2)
	assert.True(t, decoded.Arguments[0].Required)
	assert.False(t, decoded.Arguments[1].Required)
}

func TestPromptsListResult(t *testing.T) {
	result := PromptsListResult{
		Prompts: []Prompt{
			{Name: "prompt1", Description: "First prompt"},
			{Name: "prompt2", Description: "Second prompt"},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(result)
	require.NoError(t, err)

	// Unmarshal back
	var decoded PromptsListResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.Prompts, 2)
}

func TestPromptGetRequest(t *testing.T) {
	request := PromptGetRequest{
		Name: "code_review",
		Arguments: map[string]interface{}{
			"code":     "func main() {}",
			"language": "go",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(request)
	require.NoError(t, err)

	// Unmarshal back
	var decoded PromptGetRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, request.Name, decoded.Name)
	assert.NotNil(t, decoded.Arguments)
}

func TestPromptGetResult(t *testing.T) {
	result := PromptGetResult{
		Messages: []PromptMessage{
			{
				Role: "system",
				Content: ContentBlock{
					Type: "text",
					Text: "You are a code reviewer",
				},
			},
			{
				Role: "user",
				Content: ContentBlock{
					Type: "text",
					Text: "Review this code",
				},
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(result)
	require.NoError(t, err)

	// Unmarshal back
	var decoded PromptGetResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.Messages, 2)
	assert.Equal(t, "system", decoded.Messages[0].Role)
	assert.Equal(t, "user", decoded.Messages[1].Role)
}

func TestProgressNotification(t *testing.T) {
	notification := ProgressNotification{
		ProgressToken: "task-123",
		Progress:      0.75,
		Total:         100,
	}

	// Marshal to JSON
	data, err := json.Marshal(notification)
	require.NoError(t, err)

	// Unmarshal back
	var decoded ProgressNotification
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, notification.ProgressToken, decoded.ProgressToken)
	assert.Equal(t, notification.Progress, decoded.Progress)
	assert.Equal(t, notification.Total, decoded.Total)
}

func TestMCPMethodNames(t *testing.T) {
	// Test that all method constants are defined
	assert.Equal(t, "initialize", MethodInitialize)
	assert.Equal(t, "notifications/initialized", MethodInitialized)
	assert.Equal(t, "tools/list", MethodToolsList)
	assert.Equal(t, "tools/call", MethodToolsCall)
	assert.Equal(t, "resources/list", MethodResourcesList)
	assert.Equal(t, "resources/read", MethodResourcesRead)
	assert.Equal(t, "prompts/list", MethodPromptsList)
	assert.Equal(t, "prompts/get", MethodPromptsGet)
	assert.Equal(t, "notifications/progress", MethodProgress)
}

// Benchmark tests
func BenchmarkToolMarshaling(b *testing.B) {
	tool := Tool{
		Name:        "search_documents",
		Description: "Search documents",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(tool)
	}
}

func BenchmarkToolCallRequestMarshaling(b *testing.B) {
	request := ToolCallRequest{
		Name: "search_documents",
		Arguments: map[string]interface{}{
			"query": "test",
			"limit": 10,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(request)
	}
}
