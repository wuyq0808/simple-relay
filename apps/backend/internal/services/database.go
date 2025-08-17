package services

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/firestore"
)

type DatabaseService struct {
	client *firestore.Client
}

type DatabaseConfig struct {
	ProjectID string
}

func NewDatabaseService() (*DatabaseService, error) {
	ctx := context.Background()
	
	// Get project ID from environment
	projectID := os.Getenv("FIRESTORE_PROJECT_ID")
	if projectID == "" {
		return nil, fmt.Errorf("FIRESTORE_PROJECT_ID environment variable is required")
	}
	
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("firestore.NewClient: %w", err)
	}

	return &DatabaseService{client: client}, nil
}

func (ds *DatabaseService) Close() error {
	return ds.client.Close()
}

func (ds *DatabaseService) CreateTokensTable() error {
	// No table creation needed in Firestore - collections are created automatically
	return nil
}