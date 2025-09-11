package mocks

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

type BillingRequest struct {
	UserID              string
	UpstreamAccountUUID string
	Headers             http.Header
	Body                string
	Timestamp           time.Time
}

type MockBillingService struct {
	Server   *httptest.Server
	Requests []BillingRequest
	mu       sync.Mutex
	
	// Control behavior
	ShouldFail bool
	StatusCode int
}

func NewMockBillingService() *MockBillingService {
	mock := &MockBillingService{
		Requests:   []BillingRequest{},
		StatusCode: http.StatusOK,
	}
	
	mock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST requests
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		
		// Validate required headers
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("X-User-ID header is required"))
			return
		}
		
		upstreamAccountUUID := r.Header.Get("X-Upstream-Account-UUID")
		if upstreamAccountUUID == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("X-Upstream-Account-UUID header is required"))
			return
		}
		
		// Read body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		
		// Capture request
		mock.mu.Lock()
		mock.Requests = append(mock.Requests, BillingRequest{
			UserID:              userID,
			UpstreamAccountUUID: upstreamAccountUUID,
			Headers:             r.Header.Clone(),
			Body:                string(body),
			Timestamp:           time.Now(),
		})
		
		// Check if we should fail
		if mock.ShouldFail {
			w.WriteHeader(http.StatusInternalServerError)
			mock.mu.Unlock()
			return
		}
		
		statusCode := mock.StatusCode
		mock.mu.Unlock()
		
		// Simulate processing delay
		time.Sleep(10 * time.Millisecond)
		
		w.WriteHeader(statusCode)
		w.Write([]byte("OK"))
	}))
	
	return mock
}

func (m *MockBillingService) GetRequests() []BillingRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]BillingRequest{}, m.Requests...)
}

func (m *MockBillingService) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Requests = []BillingRequest{}
	m.ShouldFail = false
	m.StatusCode = http.StatusOK
}

func (m *MockBillingService) SetShouldFail(shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ShouldFail = shouldFail
}

func (m *MockBillingService) Close() {
	m.Server.Close()
}