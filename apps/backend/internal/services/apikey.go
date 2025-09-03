package services

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	lru "github.com/hashicorp/golang-lru/v2"
)

// ApiKey represents an API key within a user document
type ApiKey struct {
	api_key    string
	created_at string
}

// User represents a user document with API keys
type User struct {
	email       string
	api_keys    []ApiKey
	api_enabled bool
}

// CacheEntry represents a cached API key lookup result
type CacheEntry struct {
	UserEmail string
	Timestamp time.Time
}

// ApiKeyService handles API key operations with caching
type ApiKeyService struct {
	client        *firestore.Client
	collection    string
	cache         *lru.Cache[string, *CacheEntry]
	cacheDuration time.Duration
}

// NewApiKeyService creates a new API key service with caching
func NewApiKeyService(client *firestore.Client) *ApiKeyService {
	// Create LRU cache with capacity of 1000 entries
	cache, _ := lru.New[string, *CacheEntry](1000)
	
	return &ApiKeyService{
		client:        client,
		collection:    "users",
		cache:         cache,
		cacheDuration: time.Minute, // 1 minute cache
	}
}

// FindUserEmailByApiKey looks up the user email associated with an API key
// Returns the user email or empty string if not found
func (s *ApiKeyService) FindUserEmailByApiKey(ctx context.Context, apiKey string) (string, error) {
	fmt.Printf("[DEBUG-APIKEY] Looking up API key: %s\n", apiKey)
	
	// Check cache first
	if entry, exists := s.cache.Get(apiKey); exists {
		if time.Since(entry.Timestamp) < s.cacheDuration {
			fmt.Printf("[DEBUG-APIKEY] Cache hit for API key: %s, userId: %s\n", apiKey, entry.UserEmail)
			return entry.UserEmail, nil
		}
		// Remove expired entry
		s.cache.Remove(apiKey)
		fmt.Printf("[DEBUG-APIKEY] Cache expired for API key: %s\n", apiKey)
	}

	// Query for users that have this API key in their api_keys array
	query := s.client.Collection(s.collection).Where("api_keys", "array-contains", map[string]interface{}{
		"api_key": apiKey,
	})

	fmt.Printf("[DEBUG-APIKEY] Executing Firestore query for API key: %s\n", apiKey)
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		fmt.Printf("[DEBUG-APIKEY] Query error for API key %s: %v\n", apiKey, err)
		return "", fmt.Errorf("error querying users: %w", err)
	}

	fmt.Printf("[DEBUG-APIKEY] Query returned %d documents for API key: %s\n", len(docs), apiKey)

	// Should only be one user with this API key
	if len(docs) == 0 {
		fmt.Printf("[DEBUG-APIKEY] No user found for API key: %s\n", apiKey)
		return "", nil // API key not found
	}

	if len(docs) > 1 {
		fmt.Printf("[DEBUG-APIKEY] Multiple users found for API key: %s\n", apiKey)
		return "", fmt.Errorf("data integrity error: API key %s found in multiple users", apiKey)
	}

	var user User
	if err := docs[0].DataTo(&user); err != nil {
		fmt.Printf("[DEBUG-APIKEY] Error parsing user data for API key %s: %v\n", apiKey, err)
		return "", fmt.Errorf("error parsing user data: %w", err)
	}

	fmt.Printf("[DEBUG-APIKEY] Found user: email=%s, api_enabled=%t for API key: %s\n", user.email, user.api_enabled, apiKey)

	// Check if API is enabled for this user
	if !user.api_enabled {
		fmt.Printf("[DEBUG-APIKEY] API access disabled for user: %s\n", user.email)
		return "", nil // API access disabled
	}

	// Cache the result
	s.cache.Add(apiKey, &CacheEntry{
		UserEmail: user.email,
		Timestamp: time.Now(),
	})

	fmt.Printf("[DEBUG-APIKEY] Successfully authenticated user: %s for API key: %s\n", user.email, apiKey)
	return user.email, nil
}


