package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	lru "github.com/hashicorp/golang-lru/v2"
)

// ApiKeyBinding represents an API key binding document
type ApiKeyBinding struct {
	ApiKey    string    `firestore:"api_key" json:"api_key"`
	UserEmail string    `firestore:"user_email" json:"user_email"`
	CreatedAt time.Time `firestore:"created_at" json:"created_at"`
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
			log.Printf("DEBUG: Cache hit for API key, user: %s", entry.UserEmail)
			return entry.UserEmail, nil
		}
		// Remove expired entry
		s.cache.Remove(apiKey)
		log.Printf("DEBUG: Cache entry expired, removed from cache")
	}

	log.Printf("DEBUG: Querying Firestore for API key in collection: %s", s.collection)

	// Fetch from Firestore
	doc, err := s.client.Collection(s.collection).Doc(apiKey).Get(ctx)
	if err != nil {
		log.Printf("DEBUG: Firestore query error: %v", err)
		if doc != nil && !doc.Exists() {
			log.Printf("DEBUG: API key document does not exist")
			return "", nil
		}
		return "", fmt.Errorf("error fetching API key: %w", err)
	}

	log.Printf("DEBUG: Document exists: %t", doc.Exists())

	var binding ApiKeyBinding
	if err := doc.DataTo(&binding); err != nil {
		log.Printf("DEBUG: Error parsing document data: %v", err)
		return "", fmt.Errorf("error parsing API key binding: %w", err)
	}

	log.Printf("DEBUG: Found user email in database: %s", binding.UserEmail)

	// Cache the result
	s.cache.Add(apiKey, &CacheEntry{
		UserEmail: binding.UserEmail,
		Timestamp: time.Now(),
	})
	
	return binding.UserEmail, nil
}


