package provider

import (
	"context"
	"fmt"
	"time"

	"simple-relay/backend/internal/services"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type OAuthCredentials struct {
	ClientID         string    `json:"client_id" firestore:"client_id"`
	AccessToken      string    `json:"access_token" firestore:"access_token"`
	RefreshToken     string    `json:"refresh_token" firestore:"refresh_token"`
	ExpiresAt        time.Time `json:"expires_at" firestore:"expires_at"`
	Scope            string    `json:"scope" firestore:"scope"`
	OrganizationUUID string    `json:"organization_uuid" firestore:"organization_uuid"`
	OrganizationName string    `json:"organization_name" firestore:"organization_name"`
	AccountUUID      string    `json:"account_uuid" firestore:"account_uuid"`
	AccountEmail     string    `json:"account_email" firestore:"account_email"`
	CreatedAt        time.Time `json:"created_at" firestore:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" firestore:"updated_at"`
}

type OAuthStore struct {
	db *services.DatabaseService
}

func NewOAuthStore(db *services.DatabaseService) *OAuthStore {
	return &OAuthStore{db: db}
}

func (os *OAuthStore) SaveCredentials(clientID, accessToken, refreshToken string, expiresIn int, scope, orgUUID, orgName, accountUUID, accountEmail string) error {
	ctx := context.Background()
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)
	now := time.Now()
	
	credentials := OAuthCredentials{
		ClientID:         clientID,
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		ExpiresAt:        expiresAt,
		Scope:            scope,
		OrganizationUUID: orgUUID,
		OrganizationName: orgName,
		AccountUUID:      accountUUID,
		AccountEmail:     accountEmail,
		UpdatedAt:        now,
	}

	// Check if document exists to set CreatedAt
	docRef := os.db.Client().Collection("oauth_tokens").Doc(clientID)
	doc, err := docRef.Get(ctx)
	if err != nil && status.Code(err) != codes.NotFound {
		return fmt.Errorf("failed to check existing credentials: %w", err)
	}

	if !doc.Exists() {
		credentials.CreatedAt = now
	} else {
		// Preserve original creation time
		if data := doc.Data(); data != nil {
			if createdAt, ok := data["created_at"].(time.Time); ok {
				credentials.CreatedAt = createdAt
			} else {
				credentials.CreatedAt = now
			}
		}
	}

	_, err = docRef.Set(ctx, credentials)
	if err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	return nil
}

func (os *OAuthStore) GetCredentials(clientID string) (*OAuthCredentials, error) {
	ctx := context.Background()
	docRef := os.db.Client().Collection("oauth_tokens").Doc(clientID)
	doc, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("credentials not found for client_id: %s", clientID)
		}
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	var credentials OAuthCredentials
	err = doc.DataTo(&credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials data: %w", err)
	}

	return &credentials, nil
}

func (os *OAuthStore) GetExpiredCredentials() ([]*OAuthCredentials, error) {
	ctx := context.Background()
	now := time.Now()
	
	query := os.db.Client().Collection("oauth_tokens").Where("expires_at", "<", now)
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get expired credentials: %w", err)
	}

	var credentials []*OAuthCredentials
	for _, doc := range docs {
		var cred OAuthCredentials
		err := doc.DataTo(&cred)
		if err != nil {
			return nil, fmt.Errorf("failed to parse credentials data: %w", err)
		}
		credentials = append(credentials, &cred)
	}

	return credentials, nil
}