package database

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
)

type Service struct {
	client *firestore.Client
}

type Config struct {
	ProjectID    string
	DatabaseName string
}

func NewService(projectID, databaseName string) (*Service, error) {
	ctx := context.Background()
	
	var client *firestore.Client
	var err error
	
	// Always use named database - no more implicit defaults
	client, err = firestore.NewClientWithDatabase(ctx, projectID, databaseName)
	
	if err != nil {
		return nil, fmt.Errorf("firestore.NewClient: %w", err)
	}

	return &Service{client: client}, nil
}

func (s *Service) Close() error {
	return s.client.Close()
}

func (s *Service) Client() *firestore.Client {
	return s.client
}