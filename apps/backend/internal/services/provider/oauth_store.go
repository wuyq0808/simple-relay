package provider

import (
	"context"
	"fmt"
	"sync"
	"time"

	"simple-relay/backend/internal/services"
	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	CreatedAt        time.Time `json:"created_at" firestore:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" firestore:"updated_at"`
}

type cacheEntry struct {
	credentials *OAuthCredentials
	expiry      time.Time
}

type OAuthStore struct {
	db    *services.DatabaseService
	cache sync.Map // map[string]*cacheEntry
}

func NewOAuthStore(db *services.DatabaseService) *OAuthStore {
	return &OAuthStore{
		db:    db,
		cache: sync.Map{},
	}
}

func (store *OAuthStore) SaveCredentials(accessToken, refreshToken string, expiresIn int, scope, orgUUID, orgName, accountUUID, accountEmail string) error {
	ctx := context.Background()
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)
	now := time.Now()
	
	credentials := OAuthCredentials{
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
	docRef := store.db.Client().Collection("oauth_tokens").Doc(accountUUID)
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

	// Invalidate cache when credentials are updated
	// Note: accountUUID is the user identifier for cache invalidation
	store.cache.Delete(accountUUID)

	return nil
}

func (store *OAuthStore) GetLatestAccessToken(userID string) (*OAuthCredentials, error) {
	// Use user ID as cache key
	cacheKey := userID
	
	// Check cache first
	if cached, ok := store.cache.Load(cacheKey); ok {
		entry := cached.(*cacheEntry)
		// Check if cache entry is still valid (5 minutes cache TTL)
		if time.Now().Before(entry.expiry) {
			return entry.credentials, nil
		}
		// Cache expired, remove it
		store.cache.Delete(cacheKey)
	}
	
	// Cache miss or expired, fetch from database
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

	// Cache the result for 5 minutes
	store.cache.Store(cacheKey, &cacheEntry{
		credentials: &credentials,
		expiry:      time.Now().Add(5 * time.Minute),
	})

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