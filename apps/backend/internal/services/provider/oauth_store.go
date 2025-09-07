package provider

import (
	"context"
	"fmt"
	"time"

	"simple-relay/shared/database"

	"cloud.google.com/go/firestore"
	"github.com/hashicorp/golang-lru/v2/expirable"
)

type OAuthCredentials struct {
	AccessToken      string            `json:"access_token" firestore:"access_token"`
	RefreshToken     string            `json:"refresh_token" firestore:"refresh_token"`
	ExpiresAt        time.Time         `json:"expires_at" firestore:"expires_at"`
	Scope            string            `json:"scope" firestore:"scope"`
	OrganizationUUID string            `json:"organization_uuid" firestore:"organization_uuid"`
	OrganizationName string            `json:"organization_name" firestore:"organization_name"`
	AccountUUID      string            `json:"account_uuid" firestore:"account_uuid"`
	AccountEmail     string            `json:"account_email" firestore:"account_email"`
	UpdatedAt        time.Time         `json:"updated_at" firestore:"updated_at"`
	RefreshStartedAt time.Time         `json:"refresh_started_at" firestore:"refresh_started_at"`
	RateLimitHeaders map[string]string `json:"rate_limit_headers,omitempty" firestore:"rate_limit_headers,omitempty"`
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

	query := store.db.Client().Collection("oauth_tokens").
		Where("rate_limit_headers", "==", nil)
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

	// Verify the refreshed credentials are actually valid
	if refreshedCredentials.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("refreshed credentials are still expired")
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

func (store *OAuthStore) GetValidTokenForUser(userID string) (*UserTokenBinding, error) {
	// Check cache first for valid tokens
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
		doc, txErr := tx.Get(docRef)

		var binding *UserTokenBinding

		// Case 1: No binding exists for user - create new binding
		if txErr != nil {
			// Any error here means document doesn't exist (NotFound) or other transient issues
			// In either case, we'll create a new binding with fresh credentials
			validCreds, credsErr := store.GetValidCredentials()
			if credsErr != nil {
				return fmt.Errorf("failed to get valid token for new user binding: %w", credsErr)
			}

			binding = &UserTokenBinding{
				UserID:      userID,
				AccountUUID: validCreds.AccountUUID,
				AccessToken: validCreds.AccessToken,
				ExpiresAt:   validCreds.ExpiresAt,
			}

			if setErr := tx.Set(docRef, binding); setErr != nil {
				return fmt.Errorf("failed to save new user token binding: %w", setErr)
			}

			resultBinding = binding
			store.userTokenCache.Add(resultBinding.UserID, resultBinding)
			return nil
		}

		// Case 2: Binding exists - parse and check validity
		if parseErr := doc.DataTo(&binding); parseErr != nil {
			return fmt.Errorf("failed to parse user token binding: %w", parseErr)
		}

		now := time.Now()
		if binding.ExpiresAt.After(now) {
			// Token is still valid, use as-is
			resultBinding = binding
			store.userTokenCache.Add(resultBinding.UserID, resultBinding)
			return nil
		}

		// Case 3: Binding exists but token is expired - refresh with new credentials
		freshCreds, credsErr := store.GetValidCredentials()
		if credsErr != nil {
			return fmt.Errorf("failed to get fresh token for user %s: %w", userID, credsErr)
		}

		binding.AccessToken = freshCreds.AccessToken
		binding.ExpiresAt = freshCreds.ExpiresAt
		binding.AccountUUID = freshCreds.AccountUUID

		if setErr := tx.Set(docRef, binding); setErr != nil {
			return fmt.Errorf("failed to save refreshed user token binding: %w", setErr)
		}

		resultBinding = binding
		store.userTokenCache.Add(resultBinding.UserID, resultBinding)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return resultBinding, nil
}

func (store *OAuthStore) ClearUserTokenBinding(userID string) error {
	ctx := context.Background()

	// Delete from Firestore
	_, err := store.db.Client().Collection("user_token_bindings").Doc(userID).Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to clear user token binding for %s: %w", userID, err)
	}

	// Remove from cache after successful database operation
	store.userTokenCache.Remove(userID)

	return nil
}

func (store *OAuthStore) SaveRateLimitHeadersByToken(accessToken string, headers map[string]string) error {
	ctx := context.Background()

	// Find the OAuth token document by access_token
	query := store.db.Client().Collection("oauth_tokens").Where("access_token", "==", accessToken).Limit(1)
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return fmt.Errorf("failed to find OAuth token by access token: %w", err)
	}

	if len(docs) == 0 {
		return fmt.Errorf("no OAuth token found with access token")
	}

	// Update the document with rate limit headers
	docRef := docs[0].Ref
	_, err = docRef.Update(ctx, []firestore.Update{
		{Path: "rate_limit_headers", Value: headers},
		{Path: "updated_at", Value: time.Now()},
	})
	if err != nil {
		return fmt.Errorf("failed to save rate limit headers: %w", err)
	}

	return nil
}
