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
	ProjectID    string
	DatabaseName string
}

func NewDatabaseServiceWithDatabase(projectID, databaseName string) (*DatabaseService, error) {
	ctx := context.Background()
	
	var client *firestore.Client
	var err error
	
	// Always use named database - no more implicit defaults
	client, err = firestore.NewClientWithDatabase(ctx, projectID, databaseName)
	
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