package services

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
)

type DatabaseService struct {
	client *firestore.Client
}

type DatabaseConfig struct {
	ProjectID string
}

func NewDatabaseService(config DatabaseConfig) (*DatabaseService, error) {
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, config.ProjectID)
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