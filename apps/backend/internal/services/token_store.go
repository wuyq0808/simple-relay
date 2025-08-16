package services

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TokenData struct {
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

type TokenStore struct {
	db *DatabaseService
}

func NewTokenStore(db *DatabaseService) *TokenStore {
	return &TokenStore{db: db}
}

func (ts *TokenStore) SaveToken(clientID string, tokenResp *TokenRefreshResponse) error {
	ctx := context.Background()
	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	now := time.Now()
	
	token := TokenData{
		ClientID:         clientID,
		AccessToken:      tokenResp.AccessToken,
		RefreshToken:     tokenResp.RefreshToken,
		ExpiresAt:        expiresAt,
		Scope:            tokenResp.Scope,
		OrganizationUUID: tokenResp.Organization.UUID,
		OrganizationName: tokenResp.Organization.Name,
		AccountUUID:      tokenResp.Account.UUID,
		AccountEmail:     tokenResp.Account.EmailAddress,
		UpdatedAt:        now,
	}

	// Check if document exists to set CreatedAt
	docRef := ts.db.client.Collection("oauth_tokens").Doc(clientID)
	doc, err := docRef.Get(ctx)
	if err != nil && status.Code(err) != codes.NotFound {
		return fmt.Errorf("failed to check existing token: %w", err)
	}

	if !doc.Exists() {
		token.CreatedAt = now
	} else {
		// Preserve original creation time
		if data := doc.Data(); data != nil {
			if createdAt, ok := data["created_at"].(time.Time); ok {
				token.CreatedAt = createdAt
			} else {
				token.CreatedAt = now
			}
		}
	}

	_, err = docRef.Set(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	return nil
}

func (ts *TokenStore) GetToken(clientID string) (*TokenData, error) {
	ctx := context.Background()
	docRef := ts.db.client.Collection("oauth_tokens").Doc(clientID)
	doc, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("token not found for client_id: %s", clientID)
		}
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	var token TokenData
	err = doc.DataTo(&token)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token data: %w", err)
	}

	return &token, nil
}

func (ts *TokenStore) IsTokenExpired(token *TokenData) bool {
	return time.Now().After(token.ExpiresAt.Add(-5 * time.Minute))
}

func (ts *TokenStore) GetValidToken(clientID string) (*TokenData, error) {
	token, err := ts.GetToken(clientID)
	if err != nil {
		return nil, err
	}

	if ts.IsTokenExpired(token) {
		return nil, fmt.Errorf("token expired for client_id: %s", clientID)
	}

	return token, nil
}

func (ts *TokenStore) DeleteToken(clientID string) error {
	ctx := context.Background()
	docRef := ts.db.client.Collection("oauth_tokens").Doc(clientID)
	_, err := docRef.Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}
	return nil
}

func (ts *TokenStore) GetExpiredTokens() ([]*TokenData, error) {
	ctx := context.Background()
	now := time.Now()
	
	query := ts.db.client.Collection("oauth_tokens").Where("expires_at", "<", now)
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get expired tokens: %w", err)
	}

	var tokens []*TokenData
	for _, doc := range docs {
		var token TokenData
		err := doc.DataTo(&token)
		if err != nil {
			return nil, fmt.Errorf("failed to parse token data: %w", err)
		}
		tokens = append(tokens, &token)
	}

	return tokens, nil
}