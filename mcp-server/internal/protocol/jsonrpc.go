package protocol

import (
	"encoding/json"
	"fmt"
)

// JSON-RPC 2.0 Specification Implementation
// https://www.jsonrpc.org/specification

const (
	JSONRPCVersion = "2.0"
)

// Request represents a JSON-RPC 2.0 request
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"` // Can be string, number, or null
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Error represents a JSON-RPC 2.0 error object
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC error codes
const (
	ParseError     = -32700 // Invalid JSON was received
	InvalidRequest = -32600 // The JSON sent is not a valid Request object
	MethodNotFound = -32601 // The method does not exist / is not available
	InvalidParams  = -32602 // Invalid method parameter(s)
	InternalError  = -32603 // Internal JSON-RPC error
	ServerError    = -32000 // Generic server error
)

// MCP-specific error codes (extending JSON-RPC)
const (
	AuthenticationRequired = -32001 // Authentication is required
	AuthorizationFailed    = -32002 // Insufficient permissions
	RateLimitExceeded      = -32003 // Rate limit exceeded
	ResourceNotFound       = -32004 // Requested resource not found
	ValidationError        = -32005 // Input validation failed
)

// NewRequest creates a new JSON-RPC request
func NewRequest(id interface{}, method string, params interface{}) (*Request, error) {
	var paramsBytes json.RawMessage
	if params != nil {
		bytes, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		paramsBytes = bytes
	}

	return &Request{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  method,
		Params:  paramsBytes,
	}, nil
}

// NewResponse creates a new JSON-RPC success response
func NewResponse(id interface{}, result interface{}) *Response {
	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Result:  result,
	}
}

// NewErrorResponse creates a new JSON-RPC error response
func NewErrorResponse(id interface{}, code int, message string, data interface{}) *Response {
	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

// IsNotification returns true if the request is a notification (no ID)
func (r *Request) IsNotification() bool {
	return r.ID == nil
}

// Validate checks if the request is valid according to JSON-RPC 2.0 spec
func (r *Request) Validate() error {
	if r.JSONRPC != JSONRPCVersion {
		return fmt.Errorf("invalid jsonrpc version: expected %s, got %s", JSONRPCVersion, r.JSONRPC)
	}
	if r.Method == "" {
		return fmt.Errorf("method is required")
	}
	// ID can be string, number, or null - we accept all in interface{}
	return nil
}

// ParseParams unmarshals the params into the provided struct
func (r *Request) ParseParams(v interface{}) error {
	if len(r.Params) == 0 {
		return nil
	}
	if err := json.Unmarshal(r.Params, v); err != nil {
		return fmt.Errorf("failed to parse params: %w", err)
	}
	return nil
}

// ErrorFromCode creates a standard error message for a given code
func ErrorFromCode(code int) string {
	switch code {
	case ParseError:
		return "Parse error"
	case InvalidRequest:
		return "Invalid Request"
	case MethodNotFound:
		return "Method not found"
	case InvalidParams:
		return "Invalid params"
	case InternalError:
		return "Internal error"
	case ServerError:
		return "Server error"
	case AuthenticationRequired:
		return "Authentication required"
	case AuthorizationFailed:
		return "Authorization failed"
	case RateLimitExceeded:
		return "Rate limit exceeded"
	case ResourceNotFound:
		return "Resource not found"
	case ValidationError:
		return "Validation error"
	default:
		return "Unknown error"
	}
}
