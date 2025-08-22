package provider

import (
	"context"
	"fmt"
	"time"

	"simple-relay/shared/database"
	"github.com/hashicorp/golang-lru/v2/expirable"
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
	RefreshStartedAt time.Time `json:"refresh_started_at" firestore:"refresh_started_at"`
}

type UserTokenBinding struct {
	UserID      string    `json:"user_id" firestore:"user_id"`
	AccountUUID string    `json:"account_uuid" firestore:"account_uuid"`
	AccessToken string    `json:"access_token" firestore:"access_token"`
	ExpiresAt   time.Time `json:"expires_at" firestore:"expires_at"`
}

type OAuthStore struct {
	db             *database.Service
	userTokenCache *expirable.LRU[string, *UserTokenBinding]
}

func NewOAuthStore(db *database.Service) *OAuthStore {
	cache := expirable.NewLRU[string, *UserTokenBinding](10000, nil, 24*time.Hour)
	
	return &OAuthStore{
		db:             db,
		userTokenCache: cache,
	}
}


func (store *OAuthStore) GetRandomCredentials() (*OAuthCredentials, error) {
	ctx := context.Background()
	
	query := store.db.Client().Collection("oauth_tokens")
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}
	
	if len(docs) == 0 {
		return nil, fmt.Errorf("no credentials found in database")
	}
	
	randomIndex := time.Now().UnixNano() % int64(len(docs))
	doc := docs[randomIndex]

	var credentials OAuthCredentials
	err = doc.DataTo(&credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials data: %w", err)
	}

	return &credentials, nil
}



func (store *OAuthStore) GetValidCredentials() (*OAuthCredentials, error) {
	credentials, err := store.GetRandomCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth access token: %w", err)
	}
	
	now := time.Now()
	if credentials.ExpiresAt.After(now) {
		return credentials, nil
	}
	
	refresher := NewOAuthRefresher(store)
	refreshedCredentials, err := refresher.RefreshCredentials(credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh OAuth credentials: %w", err)
	}
	
	return refreshedCredentials, nil
}

func (store *OAuthStore) GetUserTokenBinding(userID string) (*UserTokenBinding, error) {
	ctx := context.Background()
	
	doc, err := store.db.Client().Collection("user_token_bindings").Doc(userID).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user token binding for %s: %w", userID, err)
	}
	
	if !doc.Exists() {
		return nil, fmt.Errorf("no token binding found for user %s", userID)
	}
	
	var binding UserTokenBinding
	err = doc.DataTo(&binding)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user token binding: %w", err)
	}
	
	return &binding, nil
}

func (store *OAuthStore) SaveUserTokenBinding(binding *UserTokenBinding) error {
	ctx := context.Background()
	
	_, err := store.db.Client().Collection("user_token_bindings").Doc(binding.UserID).Set(ctx, binding)
	if err != nil {
		return fmt.Errorf("failed to save user token binding: %w", err)
	}
	
	return nil
}

func (store *OAuthStore) GetValidTokenForUser(userID string) (*UserTokenBinding, error) {
	if cached, exists := store.userTokenCache.Get(userID); exists {
		if cached.ExpiresAt.After(time.Now()) {
			return cached, nil
		}
		store.userTokenCache.Remove(userID)
	}
	
	ctx := context.Background()
	var resultBinding *UserTokenBinding
	
	err := store.db.Client().RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		docRef := store.db.Client().Collection("user_token_bindings").Doc(userID)
		doc, err := tx.Get(docRef)
		
		var binding *UserTokenBinding
		if err != nil {
			randomCreds, err := store.GetValidCredentials()
			if err != nil {
				return fmt.Errorf("failed to get valid token for new user binding: %w", err)
			}
			
			binding = &UserTokenBinding{
				UserID:      userID,
				AccountUUID: randomCreds.AccountUUID,
				AccessToken: randomCreds.AccessToken,
				ExpiresAt:   randomCreds.ExpiresAt,
			}
			
			err = tx.Set(docRef, binding)
			if err != nil {
				return fmt.Errorf("failed to save new user token binding: %w", err)
			}
			
			resultBinding = binding
			return nil
		}
		
		if err := doc.DataTo(&binding); err != nil {
			return fmt.Errorf("failed to parse user token binding: %w", err)
		}
		
		now := time.Now()
		if binding.ExpiresAt.After(now) {
			resultBinding = binding
			return nil
		}
		
		freshCreds, err := store.GetValidCredentials()
		if err != nil {
			return fmt.Errorf("failed to get fresh token for user %s: %w", userID, err)
		}
		
		binding.AccessToken = freshCreds.AccessToken
		binding.ExpiresAt = freshCreds.ExpiresAt
		binding.AccountUUID = freshCreds.AccountUUID
		
		err = tx.Set(docRef, binding)
		if err != nil {
			return fmt.Errorf("failed to save refreshed user token binding: %w", err)
		}
		
		resultBinding = binding
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	store.userTokenCache.Add(resultBinding.UserID, resultBinding)
	
	return resultBinding, nil
}