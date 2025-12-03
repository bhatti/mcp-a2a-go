package protocol

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRequest(t *testing.T) {
	tests := []struct {
		name        string
		id          interface{}
		method      string
		params      interface{}
		wantErr     bool
		expectedID  interface{}
		expectedMethod string
	}{
		{
			name:        "valid request with string ID",
			id:          "test-123",
			method:      "test_method",
			params:      map[string]string{"key": "value"},
			wantErr:     false,
			expectedID:  "test-123",
			expectedMethod: "test_method",
		},
		{
			name:        "valid request with number ID",
			id:          42,
			method:      "another_method",
			params:      nil,
			wantErr:     false,
			expectedID:  42,
			expectedMethod: "another_method",
		},
		{
			name:        "valid notification (nil ID)",
			id:          nil,
			method:      "notification",
			params:      map[string]int{"count": 5},
			wantErr:     false,
			expectedID:  nil,
			expectedMethod: "notification",
		},
		{
			name:        "invalid params",
			id:          "test",
			method:      "method",
			params:      make(chan int), // channels can't be marshaled
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := NewRequest(tt.id, tt.method, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, JSONRPCVersion, req.JSONRPC)
			assert.Equal(t, tt.expectedID, req.ID)
			assert.Equal(t, tt.expectedMethod, req.Method)

			if tt.params != nil {
				assert.NotNil(t, req.Params)
			}
		})
	}
}

func TestRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		request Request
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			request: Request{
				JSONRPC: JSONRPCVersion,
				ID:      "123",
				Method:  "test",
			},
			wantErr: false,
		},
		{
			name: "invalid JSON-RPC version",
			request: Request{
				JSONRPC: "1.0",
				ID:      "123",
				Method:  "test",
			},
			wantErr: true,
			errMsg:  "invalid jsonrpc version",
		},
		{
			name: "missing method",
			request: Request{
				JSONRPC: JSONRPCVersion,
				ID:      "123",
				Method:  "",
			},
			wantErr: true,
			errMsg:  "method is required",
		},
		{
			name: "valid notification",
			request: Request{
				JSONRPC: JSONRPCVersion,
				ID:      nil,
				Method:  "notification",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRequestIsNotification(t *testing.T) {
	tests := []struct {
		name     string
		request  Request
		expected bool
	}{
		{
			name:     "notification with nil ID",
			request:  Request{ID: nil},
			expected: true,
		},
		{
			name:     "request with string ID",
			request:  Request{ID: "123"},
			expected: false,
		},
		{
			name:     "request with number ID",
			request:  Request{ID: 42},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.request.IsNotification()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRequestParseParams(t *testing.T) {
	type TestParams struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	tests := []struct {
		name      string
		params    interface{}
		target    interface{}
		wantErr   bool
		validator func(t *testing.T, target interface{})
	}{
		{
			name:   "valid params",
			params: map[string]interface{}{"name": "test", "count": 42},
			target: &TestParams{},
			wantErr: false,
			validator: func(t *testing.T, target interface{}) {
				p := target.(*TestParams)
				assert.Equal(t, "test", p.Name)
				assert.Equal(t, 42, p.Count)
			},
		},
		{
			name:    "nil params",
			params:  nil,
			target:  &TestParams{},
			wantErr: false,
			validator: func(t *testing.T, target interface{}) {
				p := target.(*TestParams)
				assert.Equal(t, "", p.Name)
				assert.Equal(t, 0, p.Count)
			},
		},
		{
			name:    "invalid params type",
			params:  map[string]interface{}{"name": 123, "count": "not a number"},
			target:  &TestParams{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := NewRequest("test", "method", tt.params)
			require.NoError(t, err)

			err = req.ParseParams(tt.target)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validator != nil {
					tt.validator(t, tt.target)
				}
			}
		})
	}
}

func TestNewResponse(t *testing.T) {
	tests := []struct {
		name           string
		id             interface{}
		result         interface{}
		expectedResult interface{}
	}{
		{
			name:           "string result",
			id:             "123",
			result:         "success",
			expectedResult: "success",
		},
		{
			name:           "object result",
			id:             42,
			result:         map[string]string{"status": "ok"},
			expectedResult: map[string]string{"status": "ok"},
		},
		{
			name:           "nil result",
			id:             "test",
			result:         nil,
			expectedResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewResponse(tt.id, tt.result)

			assert.Equal(t, JSONRPCVersion, resp.JSONRPC)
			assert.Equal(t, tt.id, resp.ID)
			assert.Equal(t, tt.expectedResult, resp.Result)
			assert.Nil(t, resp.Error)
		})
	}
}

func TestNewErrorResponse(t *testing.T) {
	tests := []struct {
		name         string
		id           interface{}
		code         int
		message      string
		data         interface{}
		expectedCode int
		expectedMsg  string
	}{
		{
			name:         "parse error",
			id:           nil,
			code:         ParseError,
			message:      "Invalid JSON",
			data:         nil,
			expectedCode: ParseError,
			expectedMsg:  "Invalid JSON",
		},
		{
			name:         "method not found with data",
			id:           "123",
			code:         MethodNotFound,
			message:      "Method not found",
			data:         map[string]string{"method": "unknown"},
			expectedCode: MethodNotFound,
			expectedMsg:  "Method not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewErrorResponse(tt.id, tt.code, tt.message, tt.data)

			assert.Equal(t, JSONRPCVersion, resp.JSONRPC)
			assert.Equal(t, tt.id, resp.ID)
			assert.Nil(t, resp.Result)
			require.NotNil(t, resp.Error)
			assert.Equal(t, tt.expectedCode, resp.Error.Code)
			assert.Equal(t, tt.expectedMsg, resp.Error.Message)
			assert.Equal(t, tt.data, resp.Error.Data)
		})
	}
}

func TestErrorFromCode(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{ParseError, "Parse error"},
		{InvalidRequest, "Invalid Request"},
		{MethodNotFound, "Method not found"},
		{InvalidParams, "Invalid params"},
		{InternalError, "Internal error"},
		{ServerError, "Server error"},
		{AuthenticationRequired, "Authentication required"},
		{AuthorizationFailed, "Authorization failed"},
		{RateLimitExceeded, "Rate limit exceeded"},
		{ResourceNotFound, "Resource not found"},
		{ValidationError, "Validation error"},
		{99999, "Unknown error"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := ErrorFromCode(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRequestJSONMarshaling(t *testing.T) {
	req := &Request{
		JSONRPC: JSONRPCVersion,
		ID:      "test-123",
		Method:  "test_method",
		Params:  json.RawMessage(`{"key":"value"}`),
	}

	// Marshal to JSON
	data, err := json.Marshal(req)
	require.NoError(t, err)

	// Unmarshal back
	var decoded Request
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, req.JSONRPC, decoded.JSONRPC)
	assert.Equal(t, req.ID, decoded.ID)
	assert.Equal(t, req.Method, decoded.Method)
	assert.JSONEq(t, string(req.Params), string(decoded.Params))
}

func TestResponseJSONMarshaling(t *testing.T) {
	resp := &Response{
		JSONRPC: JSONRPCVersion,
		ID:      42,
		Result:  map[string]string{"status": "ok"},
	}

	// Marshal to JSON
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	// Unmarshal back
	var decoded Response
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resp.JSONRPC, decoded.JSONRPC)
	assert.Equal(t, float64(42), decoded.ID) // JSON numbers unmarshal as float64
	assert.NotNil(t, decoded.Result)
}

func TestErrorResponseJSONMarshaling(t *testing.T) {
	resp := NewErrorResponse("test", InvalidParams, "Invalid params", map[string]string{"field": "name"})

	// Marshal to JSON
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	// Unmarshal back
	var decoded Response
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resp.JSONRPC, decoded.JSONRPC)
	assert.Equal(t, resp.ID, decoded.ID)
	assert.Nil(t, decoded.Result)
	require.NotNil(t, decoded.Error)
	assert.Equal(t, InvalidParams, decoded.Error.Code)
	assert.Equal(t, "Invalid params", decoded.Error.Message)
}

// Benchmark tests
func BenchmarkNewRequest(b *testing.B) {
	params := map[string]interface{}{
		"query": "test query",
		"limit": 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewRequest(i, "test_method", params)
	}
}

func BenchmarkRequestValidate(b *testing.B) {
	req := &Request{
		JSONRPC: JSONRPCVersion,
		ID:      "test",
		Method:  "test_method",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = req.Validate()
	}
}

func BenchmarkRequestParseParams(b *testing.B) {
	type Params struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}

	req, _ := NewRequest("test", "method", map[string]interface{}{"query": "test", "limit": 10})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var params Params
		_ = req.ParseParams(&params)
	}
}
