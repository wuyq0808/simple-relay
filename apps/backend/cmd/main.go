package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

type Config struct {
	Target                    *url.URL
	APIKey                   string
	AllowedClientSecretKey   string
	OfficialTarget           *url.URL
}

func loadConfig() *Config {
	// Load .env file for local development
	godotenv.Load()
	
	// Get target URL from environment variable
	apiBaseURL := os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		log.Fatal("API_BASE_URL environment variable is required")
	}
	
	// Parse target URL
	target, err := url.Parse(apiBaseURL)
	if err != nil {
		log.Fatal("Failed to parse target URL:", err)
	}

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

	return &Config{
		Target:                   target,
		APIKey:                   apiKey,
		AllowedClientSecretKey:   allowedClientSecretKey,
		OfficialTarget:           officialTarget,
	}
}

func main() {
	config := loadConfig()
	
	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(config.Target)
	
	// Set target URL for all requests and add API key
	proxy.Director = func(req *http.Request) {
		// Check for X-Official-Key header
		officialKey := req.Header.Get("X-Official-Key")
		
		if officialKey != "" && config.OfficialTarget != nil {
			// Use official target URL and X-Official-Key as bearer
			log.Printf("Using official path: %s %s -> %s (with X-Official-Key)", req.Method, req.URL.Path, config.OfficialTarget.String())
			req.URL.Scheme = config.OfficialTarget.Scheme
			req.URL.Host = config.OfficialTarget.Host
			req.Host = config.OfficialTarget.Host
			req.Header.Set("Authorization", "Bearer "+officialKey)
		} else {
			// Use default target URL and API key
			log.Printf("Using default path: %s %s -> %s", req.Method, req.URL.Path, config.Target.String())
			req.URL.Scheme = config.Target.Scheme
			req.URL.Host = config.Target.Host
			req.Host = config.Target.Host
			req.Header.Set("Authorization", "Bearer "+config.APIKey)
		}
		
		req.Header["X-Forwarded-For"] = nil
	}

	r := mux.NewRouter()
	
	// Proxy all requests with API key validation
	r.PathPrefix("/").HandlerFunc(clientApiKeyValidationMiddleware(config.AllowedClientSecretKey, proxy.ServeHTTP))
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("Server starting on port %s", port)
	log.Printf("Proxying to %s", config.Target.String())
	log.Fatal(http.ListenAndServe(":"+port, r))
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


