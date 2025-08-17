package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type TokenRefreshRequest struct {
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

type TokenRefreshResponse struct {
	TokenType    string       `json:"token_type"`
	AccessToken  string       `json:"access_token"`
	ExpiresIn    int          `json:"expires_in"`
	RefreshToken string       `json:"refresh_token"`
	Scope        string       `json:"scope"`
	Organization Organization `json:"organization"`
	Account      Account      `json:"account"`
}

type TokenRefreshScheduler struct {
	tokenStore *TokenStore
	done       chan bool
}

func NewTokenRefreshScheduler(tokenStore *TokenStore) *TokenRefreshScheduler {
	return &TokenRefreshScheduler{
		tokenStore: tokenStore,
		done:       make(chan bool),
	}
}

func (trs *TokenRefreshScheduler) Start() {
	log.Println("🔄 Starting OAuth token refresh scheduler (runs every 1 minute)...")
	ticker := time.NewTicker(1 * time.Minute)
	
	go func() {
		for {
			select {
			case <-ticker.C:
				log.Println("🔍 Token refresh scheduler: checking for expired tokens...")
				trs.refreshExpiredTokens()
			case <-trs.done:
				ticker.Stop()
				log.Println("⏹️ Token refresh scheduler stopped")
				return
			}
		}
	}()
}

func (trs *TokenRefreshScheduler) Stop() {
	trs.done <- true
}

func (trs *TokenRefreshScheduler) refreshExpiredTokens() {
	expiredTokens, err := trs.tokenStore.GetExpiredTokens()
	if err != nil {
		log.Printf("❌ Failed to get expired tokens: %v", err)
		return
	}

	if len(expiredTokens) == 0 {
		log.Println("✅ No expired tokens found")
		return
	}

	log.Printf("🔄 Found %d expired tokens to refresh", len(expiredTokens))

	successCount := 0
	for _, token := range expiredTokens {
		err := trs.refreshSingleToken(token)
		if err != nil {
			log.Printf("❌ Failed to refresh token for client_id %s: %v", token.ClientID, err)
		} else {
			log.Printf("✅ Successfully refreshed token for client_id %s", token.ClientID)
			successCount++
		}
	}
	
	log.Printf("📊 Token refresh summary: %d/%d tokens refreshed successfully", successCount, len(expiredTokens))
}

func (trs *TokenRefreshScheduler) refreshSingleToken(token *TokenData) error {
	// Prepare refresh request
	reqData := TokenRefreshRequest{
		GrantType:    "refresh_token",
		RefreshToken: token.RefreshToken,
		ClientID:     token.ClientID,
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
		return fmt.Errorf("token refresh failed with status: %d", resp.StatusCode)
	}

	var tokenResp TokenRefreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Save the new token
	err = trs.tokenStore.SaveToken(
		token.ClientID,
		tokenResp.AccessToken,
		tokenResp.RefreshToken,
		tokenResp.ExpiresIn,
		tokenResp.Scope,
		tokenResp.Organization.UUID,
		tokenResp.Organization.Name,
		tokenResp.Account.UUID,
		tokenResp.Account.EmailAddress,
	)
	if err != nil {
		return fmt.Errorf("failed to save refreshed token: %w", err)
	}

	return nil
}