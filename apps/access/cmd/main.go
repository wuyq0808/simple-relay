package main

import (
	"log"
	"net/http"
	"os"
	"simple-relay/access/internal/handlers"
	"simple-relay/shared/database"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	ProjectID    string
	DatabaseName string
}

func loadConfig() *Config {
	godotenv.Load()
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
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
		Port:         port,
		ProjectID:    projectID,
		DatabaseName: databaseName,
	}
}

func main() {
	config := loadConfig()
	
	// Initialize database service
	dbService, err := database.NewService(config.ProjectID, config.DatabaseName)
	if err != nil {
		log.Fatalf("Failed to initialize database service: %v", err)
	}
	defer dbService.Close()
	
	r := mux.NewRouter()
	
	// Health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")
	
	// Access endpoint - receives user ID and returns response
	r.HandleFunc("/access/{userID}", handlers.HandleAccess).Methods("GET")
	
	log.Printf("Access service starting on port %s", config.Port)
	log.Fatal(http.ListenAndServe(":"+config.Port, r))
}