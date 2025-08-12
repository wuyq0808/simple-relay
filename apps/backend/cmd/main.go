package main

import (
	"encoding/json"
	"log"
	"net"
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
	
	// Get IP whitelist from environment variable
	ipWhitelistEnv := os.Getenv("IP_WHITELIST")
	if ipWhitelistEnv == "" {
		log.Fatal("IP_WHITELIST environment variable is required")
	}
	
	var ipWhitelist []string
	if err := json.Unmarshal([]byte(ipWhitelistEnv), &ipWhitelist); err != nil {
		log.Fatal("Failed to parse IP_WHITELIST JSON:", err)
	}
	
	// Parse target URL
	target, err := url.Parse(apiBaseURL)
	if err != nil {
		log.Fatal("Failed to parse target URL:", err)
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(target)
	
	// Set target URL for all requests
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host
	}

	r := mux.NewRouter()
	
	// Proxy all requests with IP filtering
	r.PathPrefix("/").HandlerFunc(ipFilterMiddleware(ipWhitelist, proxy.ServeHTTP))
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("TCP Proxy server starting on port %s", port)
	log.Printf("Proxying to %s", target.String())
	log.Printf("IP whitelist: %v", ipWhitelist)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func ipFilterMiddleware(whitelist []string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip IP filtering for local development
		deploymentEnv := os.Getenv("DEPLOYMENT_ENV")
		if deploymentEnv != "production" {
			next(w, r)
			return
		}
		
		clientIP := getClientIP(r)
		
		// Check if client IP is in whitelist
		allowed := false
		for _, allowedIP := range whitelist {
			if clientIP == allowedIP {
				allowed = true
				break
			}
		}
		
		if !allowed {
			log.Printf("Blocked request from IP: %s", clientIP)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		
		next(w, r)
	}
}

func getClientIP(r *http.Request) string {
	// For Google Cloud Run, X-Forwarded-For contains:
	// <unverified IPs>, <immediate client IP>, <load balancer IP>
	// The immediate client IP is the second-to-last entry
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) >= 2 {
			// Get the second-to-last IP (immediate client IP)
			clientIP := strings.TrimSpace(ips[len(ips)-2])
			return clientIP
		} else if len(ips) == 1 {
			// Fallback to the only IP if there's just one
			return strings.TrimSpace(ips[0])
		}
	}
	
	// Fallback to RemoteAddr (direct connection)
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	
	return ip
}

