package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"simple-relay/backend/internal/messages"
	"simple-relay/backend/internal/services"
	"simple-relay/backend/internal/services/upstream"
	"simple-relay/shared/database"

	"cloud.google.com/go/compute/metadata"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

const (
	oauthBetaFlag = "oauth-2025-04-20"
)

// writeError writes an HTTP error response without adding extra newlines
// We use this custom function instead of http.Error() because http.Error()
// automatically appends a newline (\n) to the response body, which causes
// formatting issues in API clients that display the error messages
func writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Should-Retry", "false")
	w.WriteHeader(statusCode)
	w.Write([]byte(message))
}

// getIdentityToken retrieves an identity token for service-to-service authentication
func getIdentityToken(audience string) (string, error) {
	// Use Google's official metadata library
	return metadata.Get("instance/service-accounts/default/identity?audience=" + audience)
}

type Config struct {
	APIKey            string
	OfficialTarget    *url.URL
	BillingServiceURL string
	ProjectID         string
	DatabaseName      string
}

func loadConfig() *Config {
	// Load .env file for local development
	godotenv.Load()

	// Get API key from environment variable
	apiKey := os.Getenv("API_SECRET_KEY")
	if apiKey == "" {
		log.Fatal("API_SECRET_KEY environment variable is required")
	}

	// Get official base URL from environment variable
	var officialTarget *url.URL
	officialBaseURL := os.Getenv("OFFICIAL_BASE_URL")
	if officialBaseURL != "" {
		var err error
		officialTarget, err = url.Parse(officialBaseURL)
		if err != nil {
			log.Fatal("Failed to parse official target URL:", err)
		}
	}

	// Get billing service URL (required)
	billingServiceURL := os.Getenv("BILLING_SERVICE_URL")
	if billingServiceURL == "" {
		log.Fatal("BILLING_SERVICE_URL environment variable is required")
	}

	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		log.Fatal("GCP_PROJECT_ID environment variable is required")
	}

	databaseName := os.Getenv("FIRESTORE_DATABASE_NAME")
	if databaseName == "" {
		log.Fatal("FIRESTORE_DATABASE_NAME environment variable is required")
	}

	return &Config{
		APIKey:            apiKey,
		OfficialTarget:    officialTarget,
		BillingServiceURL: billingServiceURL,
		ProjectID:         projectID,
		DatabaseName:      databaseName,
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	config := loadConfig()

	// Initialize database service for OAuth
	dbService, err := database.NewService(config.ProjectID, config.DatabaseName)
	if err != nil {
		log.Fatalf("Failed to initialize database service: %v", err)
	}
	defer dbService.Close()

	// Initialize OAuth store
	oauthStore := upstream.NewOAuthStore(dbService)

	// Initialize API key service
	apiKeyService := services.NewApiKeyService(dbService.Client())

	// Initialize usage checker
	usageChecker := services.NewUsageChecker(dbService.Client())

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(config.OfficialTarget)

	// Create a custom handler that checks authentication before proxying
	proxyHandler := func(w http.ResponseWriter, req *http.Request) {
		log.Printf("[OAUTH] Request received: %s %s", req.Method, req.URL.Path)
		// Extract user ID from API key
		userId := extractUserIdFromAPIKey(req, apiKeyService)

		// Reject request if no valid API key provided
		if userId == "" {
			log.Printf("[OAUTH] No valid user ID found from API key")
			writeError(w, messages.ClientErrorMessages.Unauthorized, http.StatusUnauthorized)
			return
		}
		log.Printf("[OAUTH] Found user ID: %s", userId)

		// Check daily points limit before processing request
		remainingPoints, err := usageChecker.CheckDailyPointsLimit(req.Context(), userId)
		if err != nil {
			log.Printf("Error checking points limit for user %s: %v", userId, err)
			writeError(w, messages.ClientErrorMessages.InternalServerError, http.StatusInternalServerError)
			return
		}
		if remainingPoints <= 0 {
			w.Header().Set("X-Should-Retry", "false")
			writeError(w, messages.ClientErrorMessages.DailyLimitExceeded, http.StatusTooManyRequests)
			return
		}

		// Get OAuth token for user
		log.Printf("[OAUTH] Getting OAuth token for user %s", userId)
		tokenBinding, err := oauthStore.GetValidTokenForUser(userId)
		if err != nil {
			log.Printf("[OAUTH] ERROR: Failed to get valid token for user %s: %v", userId, err)
			writeError(w, messages.ClientErrorMessages.InternalServerError, http.StatusInternalServerError)
			return
		}
		log.Printf("[OAUTH] Successfully got token for user %s: expires=%s", 
			userId, tokenBinding.ExpiresAt.Format(time.RFC3339))

		// Store user ID, access token, and account UUID in request context for proxy director
		ctx := context.WithValue(req.Context(), "userId", userId)
		ctx = context.WithValue(ctx, "accessToken", tokenBinding.AccessToken)
		ctx = context.WithValue(ctx, "upstreamAccountUUID", tokenBinding.AccountUUID)
		req = req.WithContext(ctx)
		proxy.ServeHTTP(w, req)
	}

	// Set target URL for all requests and add OAuth token
	proxy.Director = func(req *http.Request) {
		accessToken := req.Context().Value("accessToken").(string)
		log.Printf("[OAUTH] Proxying request with token: %s...", accessToken[:min(20, len(accessToken))])

		// Use official target URL and OAuth token
		req.URL.Scheme = config.OfficialTarget.Scheme
		req.URL.Host = config.OfficialTarget.Host
		req.Host = config.OfficialTarget.Host

		// Use the OAuth access token for this user
		req.Header.Set("Authorization", "Bearer "+accessToken)

		// Ensure host header matches target
		req.Header.Set("Host", config.OfficialTarget.Host)

		// Add OAuth beta feature to anthropic-beta header if not already present
		addOAuthBetaHeader(req)

		req.Header["X-Forwarded-For"] = nil
	}

	// Intercept response for billing and 429 handling
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Log all non-200 responses with body
		if resp.StatusCode != http.StatusOK {
			logNon200Response(resp)
		}

		// Handle rate limit responses
		if resp.StatusCode == http.StatusTooManyRequests {
			handleRateLimitResponse(resp, oauthStore)
		}

		if strings.Contains(resp.Request.URL.Path, "/messages") {
			// Store original body before modification
			originalBody := resp.Body

			// Create pipe for streaming to billing
			billingPR, billingPW := io.Pipe()

			// Replace response body with teed version
			resp.Body = &struct {
				io.Reader
				io.Closer
			}{
				Reader: io.TeeReader(originalBody, billingPW),
				Closer: billingPW,
			}

			// Get user ID and account UUID from request context
			userId := resp.Request.Context().Value("userId").(string)
			accountUUID := resp.Request.Context().Value("upstreamAccountUUID").(string)

			// Start streaming to billing service
			go sendToBillingService(billingPR, resp, config, userId, accountUUID)
		}

		return nil
	}

	r := mux.NewRouter()

	// Health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// Proxy all requests with API key validation
	r.PathPrefix("/").HandlerFunc(proxyHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Printf("Proxying to %s", config.OfficialTarget.String())
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func sendToBillingService(reader io.Reader, resp *http.Response, config *Config, userId string, accountUUID string) {
	// Stream the response body directly from pipe reader
	req, err := http.NewRequest("POST", config.BillingServiceURL, reader)
	if err != nil {
		log.Printf("Error creating billing request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	
	// Only get identity token if not disabled (for testing)
	if os.Getenv("DISABLE_IDENTITY_TOKEN") != "true" {
		idToken, err := getIdentityToken(config.BillingServiceURL)
		if err != nil {
			log.Printf("Error getting identity token: %v", err)
			return
		}
		req.Header.Set("Authorization", "Bearer "+idToken)
	}
	req.Header.Set("X-User-ID", userId)
	req.Header.Set("X-Upstream-Account-UUID", accountUUID)

	// Forward all response headers to billing service
	for key, values := range resp.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	client := &http.Client{
		// No timeouts at all - let's see what happens
	}
	billingResp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending billing request: %v", err)
		return
	}
	defer billingResp.Body.Close()

	if billingResp.StatusCode != http.StatusOK {
		log.Printf("Billing service returned non-200 status: %d", billingResp.StatusCode)
	}
}

func addOAuthBetaHeader(req *http.Request) {
	existingBeta := req.Header.Get("anthropic-beta")
	if existingBeta != "" {
		if !strings.Contains(existingBeta, oauthBetaFlag) {
			req.Header.Set("anthropic-beta", oauthBetaFlag+","+existingBeta)
		}
	} else {
		req.Header.Set("anthropic-beta", oauthBetaFlag)
	}
}

// handleRateLimitResponse handles 429 rate limit responses by logging, converting to 529, and cleaning up tokens
func handleRateLimitResponse(resp *http.Response, oauthStore *upstream.OAuthStore) {
	accessToken := resp.Request.Context().Value("accessToken").(string)
	userId := resp.Request.Context().Value("userId").(string)
	log.Printf("[429] Rate limit for user %s, clearing token and returning 529", userId)

	// Capture all headers from the 429 response
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Return 529 (overloaded) to client instead of 429
	resp.StatusCode = 529
	resp.Status = messages.ClientErrorMessages.TokenOverloaded

	// Clear all headers from the response
	for key := range resp.Header {
		resp.Header.Del(key)
	}

	go func() {
		// Save headers to the OAuth token
		if err := oauthStore.SaveRateLimitHeadersByToken(accessToken, headers); err != nil {
			log.Printf("[429] Failed to save rate limit headers: %v", err)
		}

		// Clear the user token binding so they get a fresh token next time
		if err := oauthStore.ClearUserTokenBinding(userId); err != nil {
			log.Printf("[429] Failed to clear user token binding for %s: %v", userId, err)
		}
	}()
}

// logNon200Response logs non-200 responses with their body content
func logNon200Response(resp *http.Response) {
	// Read the response body for logging
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[RESPONSE] Non-200 response: %d %s for %s %s (failed to read body: %v)", resp.StatusCode, resp.Status, resp.Request.Method, resp.Request.URL.Path, err)
		return
	}
	
	// Log with truncated body (first 500 chars to avoid huge logs)
	bodyStr := string(bodyBytes)
	if len(bodyStr) > 500 {
		bodyStr = bodyStr[:500] + "..."
	}
	log.Printf("[RESPONSE] Non-200 response: %d %s for %s %s - Body: %s", resp.StatusCode, resp.Status, resp.Request.Method, resp.Request.URL.Path, bodyStr)
	
	// Restore the body for downstream consumption
	resp.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
}

// extractUserIdFromAPIKey extracts user ID from API key in Authorization header
func extractUserIdFromAPIKey(req *http.Request, apiKeyService *services.ApiKeyService) string {
	authHeader := req.Header.Get("Authorization")

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}

	apiKey := strings.TrimPrefix(authHeader, "Bearer ")

	// Look up user ID by API key with caching
	// Note: For convenience, we use email address as userId in our system
	userId, err := apiKeyService.FindUserEmailByApiKey(req.Context(), apiKey)
	if err != nil {
		return ""
	}
	if userId == "" {
		return ""
	}

	return userId
}
