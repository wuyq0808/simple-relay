package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
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


func (or *OAuthRefresher) RefreshExpiredCredentials() {
	expiredCredentials, err := or.oauthStore.GetExpiredCredentials()
	if err != nil {
		log.Printf("❌ Failed to get expired credentials: %v", err)
		return
	}

	if len(expiredCredentials) == 0 {
		return
	}


	successCount := 0
	for _, credentials := range expiredCredentials {
		err := or.RefreshSingleCredentials(credentials)
		if err == nil {
			successCount++
		}
	}
	
}

func (or *OAuthRefresher) RefreshSingleCredentials(credentials *OAuthCredentials) error {
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
		return fmt.Errorf("credentials refresh failed with status: %d", resp.StatusCode)
	}

	var refreshResp OAuthRefreshResponse
	if err := json.Unmarshal(respBody, &refreshResp); err != nil {
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