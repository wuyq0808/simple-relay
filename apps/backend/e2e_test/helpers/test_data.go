package helpers

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
)

type TestUser struct {
	Email           string
	APIKey          string
	HasAPIAccess    bool
	DailyPointsLimit int
	CreatedAt       time.Time
}

type TestOAuthToken struct {
	UserID       string
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	AccountUUID  string
	OrgName      string
}

type TestDataManager struct {
	firestoreClient *firestore.Client
	projectID       string
}

func NewTestDataManager(client *firestore.Client, projectID string) *TestDataManager {
	return &TestDataManager{
		firestoreClient: client,
		projectID:       projectID,
	}
}

// SeedUser creates a test user with API key binding
func (tdm *TestDataManager) SeedUser(ctx context.Context, user TestUser) error {
	// Create user document
	userData := map[string]interface{}{
		"email":         user.Email,
		"hasAPIAccess":  user.HasAPIAccess,
		"createdAt":     user.CreatedAt,
		"apiKeyCreated": true,
	}
	
	_, err := tdm.firestoreClient.Collection("users").Doc(user.Email).Set(ctx, userData)
	if err != nil {
		return err
	}
	
	// Create API key binding
	if user.APIKey != "" {
		apiKeyData := map[string]interface{}{
			"user_email": user.Email,  // Changed from "email" to match ApiKeyBinding struct
			"api_key":    user.APIKey,
			"createdAt":  time.Now(),
		}
		_, err = tdm.firestoreClient.Collection("api_key_bindings").Doc(user.APIKey).Set(ctx, apiKeyData)
		if err != nil {
			return err
		}
	}
	
	// Set daily points limit if specified
	if user.DailyPointsLimit > 0 {
		limitData := map[string]interface{}{
			"userId":      user.Email,
			"pointsLimit": user.DailyPointsLimit,
			"updateTime":  time.Now().Format(time.RFC3339), // Store as string to match struct
		}
		_, err = tdm.firestoreClient.Collection("daily_points_limits").Doc(user.Email).Set(ctx, limitData)
		if err != nil {
			return err
		}
	}
	
	return nil
}

// SeedOAuthToken creates an OAuth token for a user
func (tdm *TestDataManager) SeedOAuthToken(ctx context.Context, token TestOAuthToken) error {
	tokenData := map[string]interface{}{
		"accessToken":  token.AccessToken,
		"refreshToken": token.RefreshToken,
		"expiresAt":    token.ExpiresAt,
		"accountUUID":  token.AccountUUID,
		"orgName":      token.OrgName,
		"createdAt":    time.Now(),
		"updatedAt":    time.Now(),
	}
	
	_, err := tdm.firestoreClient.Collection("oauth_tokens").Doc(token.AccessToken).Set(ctx, tokenData)
	if err != nil {
		return err
	}
	
	// Create user token binding
	bindingData := map[string]interface{}{
		"user_id":      token.UserID,      // Changed from userId to match struct
		"account_uuid": token.AccountUUID,  // Added to match UserTokenBinding struct
		"access_token": token.AccessToken,  // Changed from accessToken to match struct
		"expires_at":   token.ExpiresAt,    // Added to match UserTokenBinding struct
	}
	
	_, err = tdm.firestoreClient.Collection("user_token_bindings").Doc(token.UserID).Set(ctx, bindingData)
	return err
}

// CleanupAll removes all test data
func (tdm *TestDataManager) CleanupAll(ctx context.Context) error {
	collections := []string{
		"users",
		"api_key_bindings",
		"oauth_tokens",
		"user_token_bindings",
		"daily_points_limits",
		"usage_records",
		"hourly_aggregates",
	}
	
	for _, collection := range collections {
		docs, err := tdm.firestoreClient.Collection(collection).Documents(ctx).GetAll()
		if err != nil {
			continue // Collection might not exist
		}
		
		for _, doc := range docs {
			if _, err := doc.Ref.Delete(ctx); err != nil {
				return err
			}
		}
	}
	
	return nil
}