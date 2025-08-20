package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"simple-relay/billing/internal/services"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)


type Config struct {
	ProjectID      string
	DatabaseName   string
	BillingEnabled bool
}


func loadConfig() *Config {
	// Load .env file for local development
	godotenv.Load()

	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		log.Fatal("GCP_PROJECT_ID environment variable is required")
	}

	databaseName := os.Getenv("FIRESTORE_DATABASE_NAME")
	if databaseName == "" {
		log.Fatal("FIRESTORE_DATABASE_NAME environment variable is required")
	}

	billingEnabled := os.Getenv("BILLING_ENABLED") == "true"

	return &Config{
		ProjectID:      projectID,
		DatabaseName:   databaseName,
		BillingEnabled: billingEnabled,
	}
}

func main() {
	config := loadConfig()

	// Initialize database service
	dbService, err := services.NewDatabaseServiceWithDatabase(config.ProjectID, config.DatabaseName)
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

		// Process billing request with raw response body
		err = billingService.ProcessRequest(
			responseBody,
			userID,
			requestID,
		)
		
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