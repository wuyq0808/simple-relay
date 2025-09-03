package services

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	lru "github.com/hashicorp/golang-lru/v2"
)

// ApiKeyBinding represents an API key binding document
type ApiKeyBinding struct {
	ApiKey    string `firestore:"api_key" json:"api_key"`
	UserEmail string `firestore:"user_email" json:"user_email"`
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
		collection:    "api_key_bindings",
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

	// Direct lookup using API key as document ID
	fmt.Printf("[DEBUG-APIKEY] Looking up document with ID: %s\n", apiKey)
	doc, err := s.client.Collection(s.collection).Doc(apiKey).Get(ctx)
	if err != nil {
		if doc != nil && !doc.Exists() {
			fmt.Printf("[DEBUG-APIKEY] API key document not found: %s\n", apiKey)
			return "", nil // API key not found
		}
		fmt.Printf("[DEBUG-APIKEY] Error fetching API key document: %v\n", err)
		return "", fmt.Errorf("error fetching API key: %w", err)
	}

	var binding ApiKeyBinding
	if err := doc.DataTo(&binding); err != nil {
		fmt.Printf("[DEBUG-APIKEY] Error parsing API key binding: %v\n", err)
		return "", fmt.Errorf("error parsing API key binding: %w", err)
	}

	fmt.Printf("[DEBUG-APIKEY] Found API key binding - user_email: %s\n", binding.UserEmail)

	// Cache the result
	s.cache.Add(apiKey, &CacheEntry{
		UserEmail: binding.UserEmail,
		Timestamp: time.Now(),
	})

	fmt.Printf("[DEBUG-APIKEY] Successfully authenticated user: %s for API key: %s\n", binding.UserEmail, apiKey)
	return binding.UserEmail, nil
}


