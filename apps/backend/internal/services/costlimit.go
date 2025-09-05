package services

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
)

// DailyCostLimit represents a daily cost limit document
type DailyCostLimit struct {
	UserID     string    `firestore:"userId" json:"userId"`
	CostLimit  float64   `firestore:"costLimit" json:"costLimit"`
	UpdateTime time.Time `firestore:"updateTime" json:"updateTime"`
}

// CostLimitService handles daily cost limit operations
type CostLimitService struct {
	client     *firestore.Client
	collection string
}

// NewCostLimitService creates a new cost limit service
func NewCostLimitService(client *firestore.Client) *CostLimitService {
	return &CostLimitService{
		client:     client,
		collection: "daily_cost_limits",
	}
}

// GetCostLimit retrieves a daily cost limit for a user
// Returns 0 if no cost limit is set
func (s *CostLimitService) GetCostLimit(ctx context.Context, userID string) (float64, error) {
	docRef := s.client.Collection(s.collection).Doc(userID)
	doc, err := docRef.Get(ctx)
	if err != nil {
		if doc != nil && !doc.Exists() {
			return 0, nil // Default to 0 if not found
		}
		return 0, fmt.Errorf("error fetching cost limit: %w", err)
	}

	var limit DailyCostLimit
	if err := doc.DataTo(&limit); err != nil {
		return 0, fmt.Errorf("error parsing cost limit: %w", err)
	}

	return limit.CostLimit, nil
}

