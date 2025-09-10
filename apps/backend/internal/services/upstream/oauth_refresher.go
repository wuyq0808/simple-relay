package upstream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type OAuthRefreshRequest struct {
	GrantType    string `json:"grant_type"`
	RefreshToken string `json:"refresh_token"`
	ClientID     string `json:"client_id"`
}

type Organization struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

type Account struct {
	UUID         string `json:"uuid"`
	EmailAddress string `json:"email_address"`
}

type OAuthRefreshResponse struct {
	TokenType    string       `json:"token_type"`
	AccessToken  string       `json:"access_token"`
	ExpiresIn    int          `json:"expires_in"`
	RefreshToken string       `json:"refresh_token"`
	Scope        string       `json:"scope"`
	Organization Organization `json:"organization"`
	Account      Account      `json:"account"`
}

type OAuthRefresher struct {
	oauthStore *OAuthStore
}

func NewOAuthRefresher(oauthStore *OAuthStore) *OAuthRefresher {
	return &OAuthRefresher{
		oauthStore: oauthStore,
	}
}

func (or *OAuthRefresher) RefreshCredentials(credentials *OAuthCredentials) (*OAuthCredentials, error) {
	log.Printf("[OAUTH] RefreshCredentials called for account: %s", credentials.AccountUUID)
	ctx := context.Background()

	var refreshedCredentials *OAuthCredentials
	err := or.oauthStore.db.Client().RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// Read current credentials
		docRef := or.oauthStore.db.Client().Collection("oauth_tokens").Doc(credentials.AccountUUID)
		log.Printf("[OAUTH] Looking for oauth_tokens document with ID: %s", credentials.AccountUUID)
		doc, err := tx.Get(docRef)
		if err != nil && status.Code(err) != codes.NotFound {
			log.Printf("[OAUTH] Error reading credentials document: %v", err)
			return fmt.Errorf("failed to read credentials: %w", err)
		}

		if !doc.Exists() {
			log.Printf("[OAUTH] ERROR: Credentials document not found for account UUID: %s", credentials.AccountUUID)
			return fmt.Errorf("credentials document not found")
		}
		log.Printf("[OAUTH] Found credentials document for account %s", credentials.AccountUUID)

		var currentCreds OAuthCredentials
		if err := doc.DataTo(&currentCreds); err != nil {
			return fmt.Errorf("failed to parse current credentials: %w", err)
		}

		now := time.Now()

		// Check if credentials are not expired anymore
		if now.Before(currentCreds.ExpiresAt) {
			log.Printf("[OAUTH] Credentials for account %s were already refreshed by another process (expires=%s)", 
				credentials.AccountUUID, currentCreds.ExpiresAt.Format(time.RFC3339))
			refreshedCredentials = &currentCreds
			return nil
		}
		log.Printf("[OAUTH] Credentials need refresh: expires=%s, now=%s", 
			currentCreds.ExpiresAt.Format(time.RFC3339), now.Format(time.RFC3339))

		// Write to acquire lock
		refreshStartedAt := now
		currentCreds.RefreshStartedAt = refreshStartedAt

		err = tx.Set(docRef, currentCreds)
		if err != nil {
			return fmt.Errorf("failed to acquire refresh lock: %w", err)
		}

		log.Printf("[OAUTH] Starting OAuth refresh for account %s", credentials.AccountUUID)

		// HTTP request within transaction
		reqData := OAuthRefreshRequest{
			GrantType:    "refresh_token",
			RefreshToken: credentials.RefreshToken,
			ClientID:     "9d1c250a-e61b-44d9-88ed-5944d1962f5e", // Claude Code's OAuth client ID
		}

		jsonData, err := json.Marshal(reqData)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}

		req, err := http.NewRequest("POST", "https://console.anthropic.com/v1/oauth/token", bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Accept", "application/json, text/plain, */*")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "axios/1.8.4")
		req.Header.Set("Connection", "close")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to execute request: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("[OAUTH] OAuth refresh failed with status %d, response: %s", resp.StatusCode, string(respBody))
			return fmt.Errorf("credentials refresh failed with status: %d", resp.StatusCode)
		}
		log.Printf("[OAUTH] OAuth refresh API returned status 200")

		var refreshResp OAuthRefreshResponse
		if err := json.Unmarshal(respBody, &refreshResp); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}

		// Write updated credentials
		now = time.Now()
		expiresAt := now.Add(time.Duration(refreshResp.ExpiresIn) * time.Second)

		newCredentials := OAuthCredentials{
			AccessToken:      refreshResp.AccessToken,
			RefreshToken:     refreshResp.RefreshToken,
			ExpiresAt:        expiresAt,
			Scope:            refreshResp.Scope,
			OrganizationUUID: refreshResp.Organization.UUID,
			OrganizationName: refreshResp.Organization.Name,
			AccountUUID:      refreshResp.Account.UUID,
			AccountEmail:     refreshResp.Account.EmailAddress,
			UpdatedAt:        now,
			RefreshStartedAt: refreshStartedAt,
		}

		err = tx.Set(docRef, newCredentials)
		if err != nil {
			return fmt.Errorf("failed to save refreshed credentials: %w", err)
		}

		// Store refreshed credentials to return
		refreshedCredentials = &newCredentials

		log.Printf("[OAUTH] Successfully refreshed credentials for account %s, new expiry: %s", 
			refreshResp.Account.UUID, expiresAt.Format(time.RFC3339))
		return nil
	})

	if err != nil {
		return nil, err
	}

	return refreshedCredentials, nil
}
