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
	// Check cache first
	if entry, exists := s.cache.Get(apiKey); exists {
		if time.Since(entry.Timestamp) < s.cacheDuration {
			return entry.UserEmail, nil
		}
		// Remove expired entry
		s.cache.Remove(apiKey)
	}

	// Direct lookup using API key as document ID
	doc, err := s.client.Collection(s.collection).Doc(apiKey).Get(ctx)
	if err != nil {
		if doc != nil && !doc.Exists() {
			return "", nil // API key not found
		}
		return "", fmt.Errorf("error fetching API key: %w", err)
	}

	var binding ApiKeyBinding
	if err := doc.DataTo(&binding); err != nil {
		return "", fmt.Errorf("error parsing API key binding: %w", err)
	}

	// Cache the result
	s.cache.Add(apiKey, &CacheEntry{
		UserEmail: binding.UserEmail,
		Timestamp: time.Now(),
	})

	return binding.UserEmail, nil
}


