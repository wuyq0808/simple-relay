package helpers

import (
	"context"
	"os"
	
	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"simple-relay/shared/database"
)

// CreateTestDatabaseService creates a database.Service that connects to Firestore emulator
func CreateTestDatabaseService(projectID string) (*database.Service, error) {
	// Set the emulator host environment variable
	os.Setenv("FIRESTORE_EMULATOR_HOST", "localhost:8080")
	
	// Create the service normally - it will detect the emulator env var
	return database.NewService(projectID, "(default)")
}

// CreateTestFirestoreClient creates a Firestore client for the emulator
func CreateTestFirestoreClient(ctx context.Context, projectID string) (*firestore.Client, error) {
	return firestore.NewClient(ctx, projectID,
		option.WithEndpoint("localhost:8080"),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
}