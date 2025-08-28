package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"simple-relay/backend/internal/services/provider"
	"simple-relay/shared/database"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/storage"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

const (
	oauthBetaFlag = "oauth-2025-04-20"
	// DefaultUserID is a temporary hardcoded user ID that will be replaced
	// with actual user identification from subscription service
	DefaultUserID = "hardcoded-user-123"
)

// getIdentityToken retrieves an identity token for service-to-service authentication
func getIdentityToken(audience string) (string, error) {
	// Use Google's official metadata library
	return metadata.Get("instance/service-accounts/default/identity?audience=" + audience)
}


type Config struct {
	APIKey                   string
	AllowedClientSecretKey   string
	OfficialTarget           *url.URL
	BillingServiceURL        string
	ProjectID                string
	DatabaseName             string
	APIResponsesBucket       string
}

func loadConfig() *Config {
	// Load .env file for local development
	godotenv.Load()
	

	// Get API key from environment variable
	apiKey := os.Getenv("API_SECRET_KEY")
	if apiKey == "" {
		log.Fatal("API_SECRET_KEY environment variable is required")
	}
	
	// Get allowed client secret key from environment variable
	allowedClientSecretKey := os.Getenv("ALLOWED_CLIENT_SECRET_KEY")
	if allowedClientSecretKey == "" {
		log.Fatal("ALLOWED_CLIENT_SECRET_KEY environment variable is required")
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

	apiResponsesBucket := os.Getenv("API_RESPONSES_BUCKET")
	if apiResponsesBucket == "" {
		log.Fatal("API_RESPONSES_BUCKET environment variable is required")
	}

	return &Config{
		APIKey:                   apiKey,
		AllowedClientSecretKey:   allowedClientSecretKey,
		OfficialTarget:           officialTarget,
		BillingServiceURL:        billingServiceURL,
		ProjectID:                projectID,
		DatabaseName:             databaseName,
		APIResponsesBucket:       apiResponsesBucket,
	}
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
	oauthStore := provider.NewOAuthStore(dbService)
	
	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(config.OfficialTarget)
	
	// Set target URL for all requests and add OAuth token
	proxy.Director = func(req *http.Request) {
		// TODO: Extract actual user ID from request context/headers/authentication
		// For now, using the default hardcoded user ID
		userID := DefaultUserID
		
		tokenBinding, err := oauthStore.GetValidTokenForUser(userID)
		if err != nil {
			// Fail the request if no valid OAuth token
			return
		}
		
		// Use official target URL and OAuth token
		req.URL.Scheme = config.OfficialTarget.Scheme
		req.URL.Host = config.OfficialTarget.Host
		req.Host = config.OfficialTarget.Host
		
		// Use the OAuth access token for this user
		req.Header.Set("Authorization", "Bearer "+tokenBinding.AccessToken)
		
		// Ensure host header matches target
		req.Header.Set("Host", config.OfficialTarget.Host)
		
		// Add OAuth beta feature to anthropic-beta header if not already present
		addOAuthBetaHeader(req)
		
		req.Header["X-Forwarded-For"] = nil
	}
	
	// Intercept response for billing and storage
	proxy.ModifyResponse = func(resp *http.Response) error {
		if strings.Contains(resp.Request.URL.Path, "/messages") {
			// Store original body before modification
			originalBody := resp.Body
			
			// Create pipes for streaming to both GCS and billing
			gcsPR, gcsPW := io.Pipe()
			billingPR, billingPW := io.Pipe()
			
			// Use MultiWriter to send to both pipes
			multiWriter := io.MultiWriter(gcsPW, billingPW)
			
			// Replace response body with teed version
			resp.Body = &struct {
				io.Reader
				io.Closer
			}{
				Reader: io.TeeReader(originalBody, multiWriter),
				Closer: &multiCloser{gcsPW, billingPW}, // Close both pipes
			}
			
			// Start streaming to both services
			go sendResponseToStorage(gcsPR, resp, config)
			go sendToBillingService(billingPR, resp, config)
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
	r.PathPrefix("/").HandlerFunc(clientApiKeyValidationMiddleware(config.AllowedClientSecretKey, proxy.ServeHTTP))
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("Server starting on port %s", port)
	log.Printf("Proxying to %s", config.OfficialTarget.String())
	log.Fatal(http.ListenAndServe(":"+port, r))
}


func sendToBillingService(reader io.Reader, resp *http.Response, config *Config) {
	// Get identity token for service-to-service authentication
	idToken, err := getIdentityToken(config.BillingServiceURL)
	if err != nil {
		log.Printf("Error getting identity token: %v", err)
		return
	}

	// Stream the response body directly from pipe reader
	req, err := http.NewRequest("POST", config.BillingServiceURL, reader)
	if err != nil {
		log.Printf("Error creating billing request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+idToken)
	// TODO: implement subscription system - this hardcoded user ID will be replaced
	// with actual user identification from subscription management
	req.Header.Set("X-User-ID", DefaultUserID)
	
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

func sendResponseToStorage(reader io.Reader, resp *http.Response, config *Config) {
	ctx := context.Background()
	
	// Create a storage client
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Printf("Error creating storage client: %v", err)
		return
	}
	defer client.Close()
	
	// Generate object name with timestamp and status code
	objectName := fmt.Sprintf("api-responses/%d/%d-%s.json", 
		resp.StatusCode, 
		time.Now().Unix(), 
		DefaultUserID)
	
	// Get bucket handle
	bucket := client.Bucket(config.APIResponsesBucket)
	obj := bucket.Object(objectName)
	
	// Create writer
	writer := obj.NewWriter(ctx)
	writer.ContentType = "application/json"
	
	// Add metadata with JSON-encoded headers
	metadata := map[string]string{
		"user-id":     DefaultUserID,
		"status-code": fmt.Sprintf("%d", resp.StatusCode),
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"url":         resp.Request.URL.String(),
	}
	
	// JSON encode all response headers as a single metadata value
	if headersJSON, err := json.Marshal(resp.Header); err == nil {
		metadata["headers"] = string(headersJSON)
	} else {
		log.Printf("Error marshaling headers to JSON: %v", err)
	}
	
	writer.Metadata = metadata
	
	// Stream directly from pipe reader - this blocks until EOF!
	if _, err := io.Copy(writer, reader); err != nil {
		log.Printf("Error writing to storage: %v", err)
		writer.Close()
		return
	}
	
	if err := writer.Close(); err != nil {
		log.Printf("Error closing storage writer: %v", err)
		return
	}
	
	log.Printf("API response saved to storage: %s", objectName)
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

func clientApiKeyValidationMiddleware(allowedClientSecretKey string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get API secret key from Authorization header
		var apiSecretKey string
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			apiSecretKey = strings.TrimPrefix(authHeader, "Bearer ")
		}
		
		// Check if API secret key matches
		if apiSecretKey != allowedClientSecretKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		
		next(w, r)
	}
}

// multiCloser closes multiple io.Closers
type multiCloser []io.Closer

func (mc multiCloser) Close() error {
	for _, closer := range mc {
		closer.Close() // Ignore individual errors for simplicity
	}
	return nil
}


