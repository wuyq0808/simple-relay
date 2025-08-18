package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
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
		log.Println("✅ No expired credentials found")
		return
	}

	log.Printf("🔄 Found %d expired credentials to refresh", len(expiredCredentials))

	successCount := 0
	for _, credentials := range expiredCredentials {
		err := or.RefreshSingleCredentials(credentials)
		if err != nil {
			log.Printf("❌ Failed to refresh credentials for account %s: %v", credentials.AccountUUID, err)
		} else {
			log.Printf("✅ Successfully refreshed credentials for account %s", credentials.AccountUUID)
			successCount++
		}
	}
	
	log.Printf("📊 OAuth refresh summary: %d/%d credentials refreshed successfully", successCount, len(expiredCredentials))
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
	req.Header.Set("Accept-Encoding", "gzip, compress, deflate, br")
	req.Header.Set("Connection", "close")

	reqDump, err := httputil.DumpRequestOut(req, false)
	if err != nil {
		log.Printf("OAuth Refresh Request: %s %s | Body: %s | Headers: Failed to dump", req.Method, req.URL.String(), string(jsonData))
	} else {
		log.Printf("OAuth Refresh Request: %s %s | Body: %s | Headers: %s", req.Method, req.URL.String(), string(jsonData), string(reqDump))
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("OAuth refresh request failed: %v", err)
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("OAuth Refresh Response: Failed to read body: %v", err)
		return fmt.Errorf("failed to read response body: %w", err)
	}
	
	respDump, err := httputil.DumpResponse(resp, false)
	if err != nil {
		log.Printf("OAuth Refresh Response: %d %s | Body: %s | Headers: Failed to dump", resp.StatusCode, resp.Status, string(respBody))
	} else {
		log.Printf("OAuth Refresh Response: %d %s | Body: %s | Headers: %s", resp.StatusCode, resp.Status, string(respBody), string(respDump))
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("credentials refresh failed with status: %d", resp.StatusCode)
	}

	var refreshResp OAuthRefreshResponse
	if err := json.Unmarshal(respBody, &refreshResp); err != nil {
		log.Printf("OAuth Refresh JSON Parse Error: %v | Response Body: %s", err, string(respBody))
		return fmt.Errorf("failed to decode response: %w", err)
	}
	
	log.Printf("OAuth Refresh Success: Token %s..., ExpiresIn: %ds", refreshResp.AccessToken[:20], refreshResp.ExpiresIn)

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