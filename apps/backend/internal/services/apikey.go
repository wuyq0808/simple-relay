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

	// Since array-contains doesn't work with partial map matching,
	// we need to iterate through all users to find the one with this API key
	fmt.Printf("[DEBUG-APIKEY] Iterating through all users to find API key: %s\n", apiKey)
	
	iter := s.client.Collection(s.collection).Documents(ctx)
	defer iter.Stop()

	var foundUser *User
	for {
		doc, err := iter.Next()
		if err != nil {
			if err.Error() == "iterator done" {
				break
			}
			fmt.Printf("[DEBUG-APIKEY] Iterator error for API key %s: %v\n", apiKey, err)
			return "", fmt.Errorf("error iterating users: %w", err)
		}

		var user User
		if err := doc.DataTo(&user); err != nil {
			fmt.Printf("[DEBUG-APIKEY] Error parsing user document %s: %v\n", doc.Ref.ID, err)
			continue // Skip this document if we can't parse it
		}

		fmt.Printf("[DEBUG-APIKEY] Checking user %s with %d API keys\n", user.email, len(user.api_keys))

		// Check if this user has the API key
		for _, key := range user.api_keys {
			fmt.Printf("[DEBUG-APIKEY] Comparing API key: %s vs %s\n", key.api_key, apiKey)
			if key.api_key == apiKey {
				fmt.Printf("[DEBUG-APIKEY] Found matching API key for user: %s\n", user.email)
				foundUser = &user
				break
			}
		}
		
		if foundUser != nil {
			break
		}
	}

	if foundUser == nil {
		fmt.Printf("[DEBUG-APIKEY] No user found for API key: %s\n", apiKey)
		return "", nil // API key not found
	}

	fmt.Printf("[DEBUG-APIKEY] Found user: email=%s, api_enabled=%t for API key: %s\n", foundUser.email, foundUser.api_enabled, apiKey)

	// Check if API is enabled for this user
	if !foundUser.api_enabled {
		fmt.Printf("[DEBUG-APIKEY] API access disabled for user: %s\n", foundUser.email)
		return "", nil // API access disabled
	}

	// Cache the result
	s.cache.Add(apiKey, &CacheEntry{
		UserEmail: foundUser.email,
		Timestamp: time.Now(),
	})

	fmt.Printf("[DEBUG-APIKEY] Successfully authenticated user: %s for API key: %s\n", foundUser.email, apiKey)
	return foundUser.email, nil
}


