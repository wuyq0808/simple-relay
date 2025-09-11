package mocks

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

type ClaudeRequest struct {
	Path        string
	Method      string
	Headers     http.Header
	Body        string
	AuthToken   string
}

type MockClaudeAPI struct {
	Server   *httptest.Server
	Requests []ClaudeRequest
	mu       sync.Mutex
}

func NewMockClaudeAPI() *MockClaudeAPI {
	mock := &MockClaudeAPI{
		Requests: []ClaudeRequest{},
	}
	
	mock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture request
		mock.mu.Lock()
		defer mock.mu.Unlock()
		
		authHeader := r.Header.Get("Authorization")
		body := ""
		if r.Body != nil {
			bodyBytes := make([]byte, 1024*10) // Read up to 10KB
			n, _ := r.Body.Read(bodyBytes)
			body = string(bodyBytes[:n])
		}
		
		mock.Requests = append(mock.Requests, ClaudeRequest{
			Path:      r.URL.Path,
			Method:    r.Method,
			Headers:   r.Header.Clone(),
			Body:      body,
			AuthToken: strings.TrimPrefix(authHeader, "Bearer "),
		})
		
		// Check OAuth token
		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "Unauthorized"}`))
			return
		}
		
		// Check for beta header
		betaHeader := r.Header.Get("anthropic-beta")
		if !strings.Contains(betaHeader, "oauth-2025-04-20") {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Missing required beta header"}`))
			return
		}
		
		// Simulate different responses based on path
		switch r.URL.Path {
		case "/v1/messages":
			// Return a simple SSE stream response
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			
			// Simulate SSE stream with message_start and message_delta events
			fmt.Fprintf(w, "event: message_start\n")
			fmt.Fprintf(w, "data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_test123\",\"model\":\"claude-3-opus-20240229\",\"usage\":{\"input_tokens\":10,\"output_tokens\":0}}}\n\n")
			
			fmt.Fprintf(w, "event: content_block_start\n")
			fmt.Fprintf(w, "data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n")
			
			fmt.Fprintf(w, "event: content_block_delta\n")
			fmt.Fprintf(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello, this is a test response.\"}}\n\n")
			
			fmt.Fprintf(w, "event: content_block_stop\n")
			fmt.Fprintf(w, "data: {\"type\":\"content_block_stop\",\"index\":0}\n\n")
			
			fmt.Fprintf(w, "event: message_delta\n")
			fmt.Fprintf(w, "data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\",\"stop_sequence\":null,\"usage\":{\"output_tokens\":8}}}\n\n")
			
			fmt.Fprintf(w, "event: message_stop\n")
			fmt.Fprintf(w, "data: {\"type\":\"message_stop\"}\n\n")
			
			w.(http.Flusher).Flush()
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "Not found"}`))
		}
	}))
	
	return mock
}

func (m *MockClaudeAPI) GetRequests() []ClaudeRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]ClaudeRequest{}, m.Requests...)
}

func (m *MockClaudeAPI) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Requests = []ClaudeRequest{}
}

func (m *MockClaudeAPI) Close() {
	m.Server.Close()
}