package main

import (
	"bytes"
	"context"
	"encoding/json"
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

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

const oauthBetaFlag = "oauth-2025-04-20"

type Config struct {
	APIKey                   string
	AllowedClientSecretKey   string
	OfficialTarget           *url.URL
	BillingEnabled           bool
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

	// Get billing configuration
	billingEnabled := os.Getenv("BILLING_ENABLED") == "true"
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		log.Fatal("GCP_PROJECT_ID environment variable is required")
	}

	return &Config{
		APIKey:                   apiKey,
		AllowedClientSecretKey:   allowedClientSecretKey,
		OfficialTarget:           officialTarget,
		BillingEnabled:           billingEnabled,
		ProjectID:                projectID,
	}
}

func main() {
	config := loadConfig()
	
	// Initialize database service
	dbService, err := services.NewDatabaseService(config.ProjectID)
	if err != nil {
		log.Fatalf("Failed to initialize database service: %v", err)
	}
	defer dbService.Close()
	
	// Initialize OAuth store
	oauthStore := provider.NewOAuthStore(dbService)
	
	// Initialize billing service if enabled
	var billingService *services.BillingService
	if config.BillingEnabled {
		billingService = services.NewBillingService(dbService, true)
		defer billingService.Close()
		log.Printf("Billing service initialized for project: %s", config.ProjectID)
	} else {
		log.Println("Billing service is disabled")
	}
	
	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(config.OfficialTarget)
	
	// Store request model for billing
	var requestModel string
	
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
		// Capture request body for billing if enabled
		if config.BillingEnabled && billingService != nil && strings.Contains(req.URL.Path, "/messages") {
			bodyBytes, err := io.ReadAll(req.Body)
			if err == nil {
				// 重新设置请求体
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				
				// 尝试解析请求获取model
				var apiReq services.ClaudeAPIRequest
				if err := json.Unmarshal(bodyBytes, &apiReq); err == nil {
					requestModel = apiReq.Model
				}
			}
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
		if config.BillingEnabled && billingService != nil && 
		   resp.StatusCode == http.StatusOK && 
		   strings.Contains(resp.Request.URL.Path, "/messages") {
			
			// Read response body
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("Error reading response body for billing: %v", err)
				return err
			}
			
			// Process billing asynchronously
			go func() {
				ctx := context.Background()
				
				// Get user info from request headers
				userID := resp.Request.Header.Get("X-User-ID")
				if userID == "" {
					// 可以从Authorization header或其他地方获取用户标识
					userID = "anonymous"
				}
				
				clientIP := resp.Request.RemoteAddr
				requestID := resp.Header.Get("X-Request-Id")
				if requestID == "" {
					requestID = resp.Header.Get("CF-Ray") // Cloudflare Ray ID作为备选
				}
				
				// Process response for billing
				record, err := billingService.ProcessResponse(bodyBytes, requestModel, userID, clientIP, requestID)
				if err != nil {
					log.Printf("Error processing response for billing: %v", err)
					return
				}
				
				// Record usage
				if err := billingService.RecordUsage(ctx, record); err != nil {
					log.Printf("Error recording usage: %v", err)
				} else {
					log.Printf("Usage recorded: Model=%s, Input=%d, Output=%d, Cost=$%.4f", 
						record.Model, record.InputTokens, record.OutputTokens, record.TotalCost)
				}
			}()
			
			// Reset response body
			resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
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


