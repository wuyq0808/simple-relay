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

func main() {
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
	
	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(target)
	
	// Set target URL for all requests and add API key
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	r := mux.NewRouter()
	
	// Proxy all requests with API key validation
	r.PathPrefix("/").HandlerFunc(clientApiKeyValidationMiddleware(allowedClientSecretKey, proxy.ServeHTTP))
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("TCP Proxy server starting on port %s", port)
	log.Printf("Proxying to %s", target.String())
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


