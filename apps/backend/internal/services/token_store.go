package services

import (
	"database/sql"
	"fmt"
	"time"
)

type TokenData struct {
	ID               int       `json:"id"`
	ClientID         string    `json:"client_id"`
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	ExpiresAt        time.Time `json:"expires_at"`
	Scope            string    `json:"scope"`
	OrganizationUUID string    `json:"organization_uuid"`
	OrganizationName string    `json:"organization_name"`
	AccountUUID      string    `json:"account_uuid"`
	AccountEmail     string    `json:"account_email"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type TokenStore struct {
	db *DatabaseService
}

func NewTokenStore(db *DatabaseService) *TokenStore {
	return &TokenStore{db: db}
}

func (ts *TokenStore) SaveToken(clientID string, tokenResp *TokenRefreshResponse) error {
	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	
	query := `
	INSERT INTO oauth_tokens 
	(client_id, access_token, refresh_token, expires_at, scope, 
	 organization_uuid, organization_name, account_uuid, account_email)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON DUPLICATE KEY UPDATE 
		access_token = VALUES(access_token),
		refresh_token = VALUES(refresh_token),
		expires_at = VALUES(expires_at),
		scope = VALUES(scope),
		organization_uuid = VALUES(organization_uuid),
		organization_name = VALUES(organization_name),
		account_uuid = VALUES(account_uuid),
		account_email = VALUES(account_email),
		updated_at = CURRENT_TIMESTAMP
	`

	_, err := ts.db.db.Exec(query,
		clientID,
		tokenResp.AccessToken,
		tokenResp.RefreshToken,
		expiresAt,
		tokenResp.Scope,
		tokenResp.Organization.UUID,
		tokenResp.Organization.Name,
		tokenResp.Account.UUID,
		tokenResp.Account.EmailAddress,
	)

	if err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	return nil
}

func (ts *TokenStore) GetToken(clientID string) (*TokenData, error) {
	query := `
	SELECT id, client_id, access_token, refresh_token, expires_at, scope,
		   organization_uuid, organization_name, account_uuid, account_email,
		   created_at, updated_at
	FROM oauth_tokens 
	WHERE client_id = ?
	`

	var token TokenData
	err := ts.db.db.QueryRow(query, clientID).Scan(
		&token.ID,
		&token.ClientID,
		&token.AccessToken,
		&token.RefreshToken,
		&token.ExpiresAt,
		&token.Scope,
		&token.OrganizationUUID,
		&token.OrganizationName,
		&token.AccountUUID,
		&token.AccountEmail,
		&token.CreatedAt,
		&token.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("token not found for client_id: %s", clientID)
		}
		return nil, fmt.Errorf("failed to get token: %w", err)
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
	query := "DELETE FROM oauth_tokens WHERE client_id = ?"
	_, err := ts.db.db.Exec(query, clientID)
	if err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}
	return nil
}

func (ts *TokenStore) GetExpiredTokens() ([]*TokenData, error) {
	query := `
	SELECT id, client_id, access_token, refresh_token, expires_at, scope,
		   organization_uuid, organization_name, account_uuid, account_email,
		   created_at, updated_at
	FROM oauth_tokens 
	WHERE expires_at < CURRENT_TIMESTAMP
	`

	rows, err := ts.db.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*TokenData
	for rows.Next() {
		var token TokenData
		err := rows.Scan(
			&token.ID,
			&token.ClientID,
			&token.AccessToken,
			&token.RefreshToken,
			&token.ExpiresAt,
			&token.Scope,
			&token.OrganizationUUID,
			&token.OrganizationName,
			&token.AccountUUID,
			&token.AccountEmail,
			&token.CreatedAt,
			&token.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan token: %w", err)
		}
		tokens = append(tokens, &token)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tokens: %w", err)
	}

	return tokens, nil
}