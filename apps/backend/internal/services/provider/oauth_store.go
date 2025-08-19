package provider

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"simple-relay/backend/internal/services"
	"cloud.google.com/go/firestore"
)

type OAuthCredentials struct {
	AccessToken      string    `json:"access_token" firestore:"access_token"`
	RefreshToken     string    `json:"refresh_token" firestore:"refresh_token"`
	ExpiresAt        time.Time `json:"expires_at" firestore:"expires_at"`
	Scope            string    `json:"scope" firestore:"scope"`
	OrganizationUUID string    `json:"organization_uuid" firestore:"organization_uuid"`
	OrganizationName string    `json:"organization_name" firestore:"organization_name"`
	AccountUUID      string    `json:"account_uuid" firestore:"account_uuid"`
	AccountEmail     string    `json:"account_email" firestore:"account_email"`
	UpdatedAt        time.Time `json:"updated_at" firestore:"updated_at"`
}

type OAuthStore struct {
	db                *services.DatabaseService
	cachedCredentials atomic.Pointer[OAuthCredentials]
}

func NewOAuthStore(db *services.DatabaseService) *OAuthStore {
	return &OAuthStore{
		db: db,
	}
}


func (store *OAuthStore) GetLatestAccessToken() (*OAuthCredentials, error) {
	// Check cache first
	if cached := store.cachedCredentials.Load(); cached != nil {
		return cached, nil
	}
	
	// Cache miss, fetch from database
	ctx := context.Background()
	
	query := store.db.Client().Collection("oauth_tokens").OrderBy("updated_at", firestore.Desc).Limit(1)
	iter := query.Documents(ctx)
	doc, err := iter.Next()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest credentials: %w", err)
	}
	iter.Stop()

	var credentials OAuthCredentials
	err = doc.DataTo(&credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials data: %w", err)
	}

	// Cache the result atomically
	store.cachedCredentials.Store(&credentials)

	return &credentials, nil
}


func (store *OAuthStore) GetExpiredCredentials() ([]*OAuthCredentials, error) {
	ctx := context.Background()
	now := time.Now()
	
	query := store.db.Client().Collection("oauth_tokens").Where("expires_at", "<", now)
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