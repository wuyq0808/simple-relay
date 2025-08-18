package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type OAuthRefreshRequest struct {
	GrantType    string `json:"grant_type"`
	RefreshToken string `json:"refresh_token"`
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

func (or *OAuthRefresher) RefreshExpiredCredentials() {
	expiredCredentials, err := or.oauthStore.GetExpiredCredentials()
	if err != nil {
		log.Printf("❌ Failed to get expired credentials: %v", err)
		return
	}

	if len(expiredCredentials) == 0 {
		log.Println("✅ No expired credentials found")
		return
	}

	log.Printf("🔄 Found %d expired credentials to refresh", len(expiredCredentials))

	successCount := 0
	for _, credentials := range expiredCredentials {
		err := or.refreshSingleCredentials(credentials)
		if err != nil {
			log.Printf("❌ Failed to refresh credentials for account %s: %v", credentials.AccountUUID, err)
		} else {
			log.Printf("✅ Successfully refreshed credentials for account %s", credentials.AccountUUID)
			successCount++
		}
	}
	
	log.Printf("📊 OAuth refresh summary: %d/%d credentials refreshed successfully", successCount, len(expiredCredentials))
}

func (or *OAuthRefresher) refreshSingleCredentials(credentials *OAuthCredentials) error {
	// Prepare refresh request
	reqData := OAuthRefreshRequest{
		GrantType:    "refresh_token",
		RefreshToken: credentials.RefreshToken,
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make refresh request to OAuth provider
	// Note: This URL should be configurable or passed as parameter
	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/oauth/token", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", "simple-relay/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("credentials refresh failed with status: %d", resp.StatusCode)
	}

	var refreshResp OAuthRefreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&refreshResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Save the new credentials
	err = or.oauthStore.SaveCredentials(
		refreshResp.AccessToken,
		refreshResp.RefreshToken,
		refreshResp.ExpiresIn,
		refreshResp.Scope,
		refreshResp.Organization.UUID,
		refreshResp.Organization.Name,
		refreshResp.Account.UUID,
		refreshResp.Account.EmailAddress,
	)
	if err != nil {
		return fmt.Errorf("failed to save refreshed credentials: %w", err)
	}

	return nil
}