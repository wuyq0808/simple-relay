package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"simple-relay/billing/internal/services"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

type Config struct {
	ProjectID      string
	BillingEnabled bool
}


func loadConfig() *Config {
	// Load .env file for local development
	godotenv.Load()

	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		log.Fatal("GCP_PROJECT_ID environment variable is required")
	}

	billingEnabled := os.Getenv("BILLING_ENABLED") == "true"

	return &Config{
		ProjectID:      projectID,
		BillingEnabled: billingEnabled,
	}
}

// parseSSEForUsageData extracts usage data from message_delta events only (simplified)
// TODO: Add message_start parsing to get proper message ID and model information
// TODO: For now using placeholder values - model field will be "unknown-model" 
// TODO: This affects billing records but usage data (tokens) is accurate
func parseSSEForUsageData(sseData string) (*services.ClaudeMessage, error) {
	lines := strings.Split(sseData, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data: ") {
			jsonData := strings.TrimPrefix(line, "data: ")
			if jsonData == "[DONE]" {
				continue
			}
			
			var event map[string]interface{}
			if err := json.Unmarshal([]byte(jsonData), &event); err != nil {
				log.Printf("Failed to parse SSE event: %v, data: %s", err, jsonData)
				continue
			}
			
			// Only look for message_delta events with usage data
			if eventType, _ := event["type"].(string); eventType == "message_delta" {
				if usage, ok := event["usage"].(map[string]interface{}); ok {
					log.Printf("Found usage data: input=%v, output=%v", 
						usage["input_tokens"], usage["output_tokens"])
					
					// Create minimal message with just usage data
					messageData := map[string]interface{}{
						"id":    fmt.Sprintf("msg-%d", time.Now().UnixNano()), // TODO: get real ID from message_start
						"model": "unknown-model", // TODO: get real model from message_start  
						"usage": usage,
					}
					
					// Convert to ClaudeMessage struct
					messageJSON, err := json.Marshal(messageData)
					if err != nil {
						return nil, fmt.Errorf("failed to marshal message: %w", err)
					}
					
					var message services.ClaudeMessage
					if err := json.Unmarshal(messageJSON, &message); err != nil {
						return nil, fmt.Errorf("failed to unmarshal into ClaudeMessage: %w", err)
					}
					
					return &message, nil
				}
			}
		}
	}
	
	return nil, fmt.Errorf("no message_delta event with usage data found")
}


func main() {
	config := loadConfig()

	// Initialize database service
	dbService, err := services.NewDatabaseService(config.ProjectID)
	if err != nil {
		log.Fatalf("Failed to initialize database service: %v", err)
	}
	defer dbService.Close()

	// Initialize billing service
	var billingService *services.BillingService
	if config.BillingEnabled {
		billingService = services.NewBillingService(dbService, true)
		defer billingService.Close()
		log.Printf("Billing service initialized for project: %s", config.ProjectID)
	} else {
		log.Println("Billing service is disabled")
	}

	r := mux.NewRouter()

	// Health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// Root endpoint to accept billing requests
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if billingService == nil {
			http.Error(w, "Billing service not enabled", http.StatusServiceUnavailable)
			return
		}

		// Get user ID from header
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			http.Error(w, "X-User-ID header is required", http.StatusBadRequest)
			return
		}

		// Read raw response body (Claude API response)
		responseBody, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading response body: %v", err)
			http.Error(w, "Error reading request body", http.StatusBadRequest)
			return
		}

		// Extract additional metadata from headers if available
		requestID := r.Header.Get("X-Request-Id") // From Claude API response

		// Process SSE data - extract message_stop and pass to ProcessResponse
		bodyStr := string(responseBody)
		
		// Only process SSE streams - use guard clause for early return
		if !strings.HasPrefix(bodyStr, "event:") && !strings.HasPrefix(bodyStr, "data:") {
			log.Printf("Skipping non-SSE response for billing")
			http.Error(w, "Only SSE streams are supported for billing", http.StatusBadRequest)
			return
		}
		
		// Parse SSE stream to extract usage data from message_start and message_delta events
		message, err := parseSSEForUsageData(bodyStr)
		if err != nil {
			log.Printf("Error parsing SSE stream for user %s: %v", userID, err)
			http.Error(w, "Error parsing SSE stream", http.StatusBadRequest)
			return
		}
		
		// Use ProcessRequest with the parsed message
		err = billingService.ProcessRequest(message, userID, requestID)
		if err != nil {
			log.Printf("Error processing billing request for user %s: %v", userID, err)
			http.Error(w, "Error processing billing", http.StatusInternalServerError)
			return
		}

		log.Printf("Billing processed successfully for user: %s", userID)

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "Billing processed successfully",
		})
	}).Methods("POST")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Billing service starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}