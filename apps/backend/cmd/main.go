package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"simple-relay/backend/internal/services"
	"simple-relay/backend/internal/services/provider"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

const oauthBetaFlag = "oauth-2025-04-20"

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

	// Get official base URL from environment variable (optional)
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

	return &Config{
		APIKey:                   apiKey,
		AllowedClientSecretKey:   allowedClientSecretKey,
		OfficialTarget:           officialTarget,
		BillingServiceURL:        billingServiceURL,
		ProjectID:                projectID,
	}
}

func main() {
	config := loadConfig()
	
	// Initialize database service for OAuth
	dbService, err := services.NewDatabaseService(config.ProjectID)
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
		
		// Get valid OAuth access token for each request
		// TODO: add memory cache for the get access token method
		credentials, err := oauthStore.GetLatestAccessToken()
		if err != nil {
			log.Printf("Failed to get OAuth access token: %v", err)
			// Fail the request if no valid OAuth token
			return
		}
		
		// Check if token is expired and refresh if needed
		now := time.Now()
		if credentials.ExpiresAt.Before(now) {
			log.Printf("OAuth token expired at %v, refreshing...", credentials.ExpiresAt)
			// Token is expired, refresh it
			refresher := provider.NewOAuthRefresher(oauthStore)
			err = refresher.RefreshSingleCredentials(credentials)
			if err != nil {
				log.Printf("Failed to refresh OAuth credentials: %v", err)
				// Fail the request if refresh fails
				return
			}
			log.Printf("OAuth token refreshed successfully")
			
			// Get the refreshed token
			credentials, err = oauthStore.GetLatestAccessToken()
			if err != nil {
				log.Printf("Failed to get refreshed OAuth access token: %v", err)
				// Fail the request if can't get refreshed token
				return
			}
			log.Printf("Retrieved refreshed OAuth token, expires at: %v", credentials.ExpiresAt)
		}
		
		// Use official target URL and OAuth token
		req.URL.Scheme = config.OfficialTarget.Scheme
		req.URL.Host = config.OfficialTarget.Host
		req.Host = config.OfficialTarget.Host
		
		// Use the OAuth access token obtained at startup
		req.Header.Set("Authorization", "Bearer "+credentials.AccessToken)
		
		// Ensure host header matches target
		req.Header.Set("Host", config.OfficialTarget.Host)
		
		// Add OAuth beta feature to anthropic-beta header if not already present
		addOAuthBetaHeader(req)
		
		req.Header["X-Forwarded-For"] = nil
	}
	
	// Intercept response for billing
	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp.StatusCode == http.StatusOK && 
		   strings.Contains(resp.Request.URL.Path, "/messages") {
			
			// Read the entire response body first
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			
			// Replace response body with the original data for the client
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			
			// Send raw response body to billing service asynchronously
			go func() {
				// Get identity token for service-to-service authentication
				log.Printf("Getting identity token for audience: %s", config.BillingServiceURL)
				idToken, err := getIdentityToken(config.BillingServiceURL)
				if err != nil {
					log.Printf("Error getting identity token: %v", err)
					return
				}
				log.Printf("Successfully obtained identity token")
				
				// First test connectivity with health endpoint
				healthURL := config.BillingServiceURL + "/health"
				healthReq, _ := http.NewRequest("GET", healthURL, nil)
				healthReq.Header.Set("Authorization", "Bearer "+idToken)
				healthResp, err := (&http.Client{Timeout: 5 * time.Second}).Do(healthReq)
				if err != nil {
					log.Printf("Health check failed: %v", err)
				} else {
					healthBody, _ := io.ReadAll(healthResp.Body)
					log.Printf("Health check response: %d, body: %s", healthResp.StatusCode, string(healthBody))
					healthResp.Body.Close()
				}

				req, err := http.NewRequest("POST", config.BillingServiceURL, bytes.NewReader(bodyBytes))
				if err != nil {
					log.Printf("Error creating billing request: %v", err)
					return
				}
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+idToken)
				// TODO: implement subscription system - this hardcoded user ID will be replaced
				// with actual user identification from subscription management
				req.Header.Set("X-User-ID", "hardcoded-user-123")
				
				// Forward all response headers to billing service
				for key, values := range resp.Header {
					for _, value := range values {
						req.Header.Add(key, value)
					}
				}
				
				log.Printf("Sending billing request to: %s", config.BillingServiceURL)
				client := &http.Client{Timeout: 10 * time.Second}
				billingResp, err := client.Do(req)
				if err != nil {
					log.Printf("Error sending billing request: %v", err)
					return
				}
				defer billingResp.Body.Close()
				
				if billingResp.StatusCode != http.StatusOK {
					// Read response body for detailed error information
					respBody, _ := io.ReadAll(billingResp.Body)
					log.Printf("Billing service returned non-200 status: %d, response: %s, URL: %s", billingResp.StatusCode, string(respBody), config.BillingServiceURL)
				} else {
					log.Printf("Billing request successful")
				}
			}()
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


