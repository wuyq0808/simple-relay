package services

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
)

// DailyPointsLimit represents a daily points limit document
type DailyPointsLimit struct {
	UserID      string `firestore:"userId" json:"userId"`
	PointsLimit int    `firestore:"pointsLimit" json:"pointsLimit"`
	UpdateTime  string `firestore:"updateTime" json:"updateTime"`
}

// PointsLimitService handles daily points limit operations
type PointsLimitService struct {
	client     *firestore.Client
	collection string
}

// NewPointsLimitService creates a new points limit service
func NewPointsLimitService(client *firestore.Client) *PointsLimitService {
	return &PointsLimitService{
		client:     client,
		collection: "daily_points_limits",
	}
}

// GetPointsLimit retrieves a daily points limit for a user
// Returns 0 if no points limit is set
func (s *PointsLimitService) GetPointsLimit(ctx context.Context, userID string) (int, error) {
	docRef := s.client.Collection(s.collection).Doc(userID)
	doc, err := docRef.Get(ctx)
	if err != nil {
		if doc != nil && !doc.Exists() {
			return 0, nil // Default to 0 if not found
		}
		return 0, fmt.Errorf("error fetching points limit: %w", err)
	}

	var limit DailyPointsLimit
	if err := doc.DataTo(&limit); err != nil {
		return 0, fmt.Errorf("error parsing points limit: %w", err)
	}

	return limit.PointsLimit, nil
}
