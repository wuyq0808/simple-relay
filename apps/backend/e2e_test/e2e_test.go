package e2e_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/suite"

	"simple-relay/backend/e2e_test/helpers"
	"simple-relay/backend/e2e_test/mocks"
)

type E2EIntegrationTestSuite struct {
	suite.Suite
	
	// Real backend process
	backendCmd *exec.Cmd
	backendURL string
	
	// Mock services
	mockClaudeAPI      *mocks.MockClaudeAPI
	mockBillingService *mocks.MockBillingService
	
	// Firestore client for test data
	firestoreClient *firestore.Client
	testData        *helpers.TestDataManager
	
	// Test configuration
	testAPIKey    string
	testUserEmail string
}

func TestE2EIntegrationSuite(t *testing.T) {
	suite.Run(t, new(E2EIntegrationTestSuite))
}

func (suite *E2EIntegrationTestSuite) SetupSuite() {
	ctx := context.Background()
	projectID := "test-project"
	
	// Set up Firestore client (assumes emulator is running via docker-compose)
	os.Setenv("FIRESTORE_EMULATOR_HOST", "localhost:8080")
	
	client, err := firestore.NewClient(ctx, projectID)
	suite.Require().NoError(err, "Failed to create Firestore client")
	suite.firestoreClient = client
	
	// Initialize test data manager
	suite.testData = helpers.NewTestDataManager(client, projectID)
	
	// Start mock services
	suite.mockClaudeAPI = mocks.NewMockClaudeAPI()
	suite.mockBillingService = mocks.NewMockBillingService()
	
	// Set environment variables for the backend process
	os.Setenv("API_SECRET_KEY", "test-secret-key")
	os.Setenv("OFFICIAL_BASE_URL", suite.mockClaudeAPI.Server.URL)
	os.Setenv("BILLING_SERVICE_URL", suite.mockBillingService.Server.URL)
	os.Setenv("GCP_PROJECT_ID", projectID)
	os.Setenv("FIRESTORE_DATABASE_NAME", "(default)")
	os.Setenv("PORT", "8888") // Use a different port for testing
	
	// Disable identity token fetching in test environment
	// This prevents the backend from trying to contact GCP metadata service
	os.Setenv("DISABLE_IDENTITY_TOKEN", "true")
	
	// Default test values
	suite.testAPIKey = "test-api-key-123"
	suite.testUserEmail = "test@example.com"
	suite.backendURL = "http://localhost:8888"
	
	// Build the backend binary
	suite.buildBackend()
	
	// Start the backend process
	suite.startBackendProcess()
	
	// Wait for backend to be ready
	suite.waitForBackend()
}

func (suite *E2EIntegrationTestSuite) TearDownSuite() {
	// Stop backend process
	if suite.backendCmd != nil && suite.backendCmd.Process != nil {
		// Send interrupt signal for graceful shutdown
		suite.backendCmd.Process.Signal(syscall.SIGINT)
		
		// Wait for process to exit (with timeout)
		done := make(chan error, 1)
		go func() {
			done <- suite.backendCmd.Wait()
		}()
		
		select {
		case <-done:
			// Process exited gracefully
		case <-time.After(5 * time.Second):
			// Force kill if it doesn't exit gracefully
			suite.backendCmd.Process.Kill()
		}
	}
	
	// Close mock services
	if suite.mockClaudeAPI != nil {
		suite.mockClaudeAPI.Close()
	}
	if suite.mockBillingService != nil {
		suite.mockBillingService.Close()
	}
	
	// Close Firestore client
	if suite.firestoreClient != nil {
		suite.firestoreClient.Close()
	}
}

func (suite *E2EIntegrationTestSuite) SetupTest() {
	ctx := context.Background()
	
	// Clean up all test data
	err := suite.testData.CleanupAll(ctx)
	suite.Require().NoError(err, "Failed to cleanup test data")
	
	// Reset mock services
	suite.mockClaudeAPI.Reset()
	suite.mockBillingService.Reset()
}

// Helper: Build the backend binary
func (suite *E2EIntegrationTestSuite) buildBackend() {
	cmd := exec.Command("go", "build", "-o", "../bin/backend-test", "../cmd")
	output, err := cmd.CombinedOutput()
	suite.Require().NoError(err, "Failed to build backend: %s", string(output))
}

// Helper: Start the backend process
func (suite *E2EIntegrationTestSuite) startBackendProcess() {
	suite.backendCmd = exec.Command("../bin/backend-test")
	
	// Capture stdout and stderr for debugging
	suite.backendCmd.Stdout = os.Stdout
	suite.backendCmd.Stderr = os.Stderr
	
	// Set environment variables
	suite.backendCmd.Env = os.Environ()
	
	// Start the process
	err := suite.backendCmd.Start()
	suite.Require().NoError(err, "Failed to start backend process")
	
	suite.T().Logf("Started backend process with PID: %d", suite.backendCmd.Process.Pid)
}

// Helper: Wait for backend to be ready
func (suite *E2EIntegrationTestSuite) waitForBackend() {
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(suite.backendURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			suite.T().Log("Backend is ready")
			return
		}
		time.Sleep(1 * time.Second)
	}
	suite.Fail("Backend did not start in time")
}

// Helper: Make authenticated request
func (suite *E2EIntegrationTestSuite) makeAuthenticatedRequest(method, path string, body io.Reader) (*http.Response, string) {
	req, err := http.NewRequest(method, suite.backendURL+path, body)
	suite.Require().NoError(err)
	
	req.Header.Set("Authorization", "Bearer "+suite.testAPIKey)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	suite.Require().NoError(err)
	
	bodyBytes, err := io.ReadAll(resp.Body)
	suite.Require().NoError(err)
	resp.Body.Close()
	
	return resp, string(bodyBytes)
}

// TEST: Happy path - successful request with valid API key and OAuth token
func (suite *E2EIntegrationTestSuite) TestE2E_SuccessfulProxyRequest() {
	ctx := context.Background()
	
	// Seed test user with API access
	err := suite.testData.SeedUser(ctx, helpers.TestUser{
		Email:            suite.testUserEmail,
		APIKey:           suite.testAPIKey,
		HasAPIAccess:     true,
		DailyPointsLimit: 1000,
		CreatedAt:        time.Now(),
	})
	suite.Require().NoError(err, "Failed to seed test user")
	
	// Seed valid OAuth token
	err = suite.testData.SeedOAuthToken(ctx, helpers.TestOAuthToken{
		UserID:       suite.testUserEmail,
		AccessToken:  "test-oauth-token-123",
		RefreshToken: "test-refresh-token-123",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		AccountUUID:  "test-account-uuid",
		OrgName:      "Test Organization",
	})
	suite.Require().NoError(err, "Failed to seed OAuth token")
	
	// Make a request to the messages endpoint
	requestBody := `{
		"model": "claude-3-opus-20240229",
		"messages": [{"role": "user", "content": "Hello, Claude!"}],
		"max_tokens": 100
	}`
	
	resp, body := suite.makeAuthenticatedRequest("POST", "/v1/messages", bytes.NewBufferString(requestBody))
	
	// Verify response
	suite.Equal(http.StatusOK, resp.StatusCode, "Expected 200 OK response")
	suite.Contains(body, "message_start", "Response should contain SSE events")
	
	// Wait a bit for async operations
	time.Sleep(100 * time.Millisecond)
	
	// Verify Claude API was called with OAuth token
	claudeRequests := suite.mockClaudeAPI.GetRequests()
	suite.Require().Len(claudeRequests, 1, "Claude API should be called once")
	suite.Equal("/v1/messages", claudeRequests[0].Path)
	suite.Equal("test-oauth-token-123", claudeRequests[0].AuthToken)
	suite.Contains(claudeRequests[0].Headers.Get("anthropic-beta"), "oauth-2025-04-20")
	
	// Verify billing service was called
	billingRequests := suite.mockBillingService.GetRequests()
	suite.Require().Len(billingRequests, 1, "Billing service should be called once")
	suite.Equal(suite.testUserEmail, billingRequests[0].UserID)
	suite.Equal("test-account-uuid", billingRequests[0].UpstreamAccountUUID)
	suite.Contains(billingRequests[0].Body, "message_start", "Billing should receive SSE stream")
}

// TEST: Unauthorized request without API key
func (suite *E2EIntegrationTestSuite) TestE2E_Unauthorized_NoAPIKey() {
	req, err := http.NewRequest("POST", suite.backendURL+"/v1/messages", nil)
	suite.Require().NoError(err)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().NoError(err)
	defer resp.Body.Close()
	
	suite.Equal(http.StatusUnauthorized, resp.StatusCode, "Expected 401 Unauthorized")
}

// TEST: Rate limiting - daily points exceeded
func (suite *E2EIntegrationTestSuite) TestE2E_RateLimiting_DailyPointsExceeded() {
	ctx := context.Background()
	
	// Use a different user for this test to avoid cache issues
	rateLimitedUser := "ratelimited@example.com"
	rateLimitedAPIKey := "rate-limited-api-key"
	
	// Seed test user with zero points limit
	err := suite.testData.SeedUser(ctx, helpers.TestUser{
		Email:            rateLimitedUser,
		APIKey:           rateLimitedAPIKey,
		HasAPIAccess:     true,
		DailyPointsLimit: 0, // No points available
		CreatedAt:        time.Now(),
	})
	suite.Require().NoError(err, "Failed to seed test user")
	
	// Make a request with this user's API key
	req, err := http.NewRequest("POST", suite.backendURL+"/v1/messages", nil)
	suite.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+rateLimitedAPIKey)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	suite.Require().NoError(err)
	defer resp.Body.Close()
	
	// This user has 0 points, so should get rate limited
	suite.Equal(http.StatusTooManyRequests, resp.StatusCode, 
		"Expected 429 Too Many Requests for user with no daily points")
}

// TEST: Health check endpoint
func (suite *E2EIntegrationTestSuite) TestE2E_HealthCheck() {
	resp, err := http.Get(suite.backendURL + "/health")
	suite.Require().NoError(err)
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	suite.Require().NoError(err)
	
	suite.Equal(http.StatusOK, resp.StatusCode)
	suite.Equal("OK", string(body))
}