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

func NewDatabaseService(projectID string) (*DatabaseService, error) {
	ctx := context.Background()
	
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("firestore.NewClient: %w", err)
	}

	return &DatabaseService{client: client}, nil
}

func (ds *DatabaseService) Close() error {
	return ds.client.Close()
}


func (ds *DatabaseService) Client() *firestore.Client {
	return ds.client
}