package services

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	lru "github.com/hashicorp/golang-lru/v2"
)

// UsageCacheEntry represents a cached usage check result
type UsageCacheEntry struct {
	RemainingPoints int
	Timestamp       time.Time
}

// UsageChecker handles daily points limit checking
type UsageChecker struct {
	client              *firestore.Client
	pointsLimitService  *PointsLimitService
	cache               *lru.Cache[string, *UsageCacheEntry]
	cacheDuration       time.Duration
}

// NewUsageChecker creates a new usage checker
func NewUsageChecker(client *firestore.Client) *UsageChecker {
	// Create LRU cache with capacity of 1000 entries
	cache, _ := lru.New[string, *UsageCacheEntry](1000)

	return &UsageChecker{
		client:             client,
		pointsLimitService: NewPointsLimitService(client),
		cache:              cache,
		cacheDuration:      24 * time.Hour, // 24 hour cache
	}
}

// cleanupExpiredEntry checks if cache entry is expired and removes it if so
// Returns the entry if still valid, nil if expired or not found
func (uc *UsageChecker) cleanupExpiredEntry(userID string) *UsageCacheEntry {
	if entry, exists := uc.cache.Get(userID); exists {
		if time.Since(entry.Timestamp) < uc.cacheDuration {
			return entry
		}
		// Remove expired entry
		uc.cache.Remove(userID)
	}
	return nil
}

// calculateRemainingPointsFromDB calculates remaining points by querying database
func (uc *UsageChecker) calculateRemainingPointsFromDB(ctx context.Context, userID string) (int, error) {
	// Get user's points limit (defaults to 0 if not set)
	// Points are stored as cost * 10 in the database
	pointsLimit, err := uc.pointsLimitService.GetPointsLimit(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("error getting points limit: %w", err)
	}

	// If limit is 0, return 0 directly (no usage allowed) - don't cache
	if pointsLimit == 0 {
		return 0, nil
	}

	// Calculate current 24-hour usage (8pm-8pm UTC window)
	// This returns points from the database (cost * 10)
	currentUsagePoints, err := uc.getCurrentDailyUsage(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("error getting current usage: %w", err)
	}

	// Both pointsLimit and currentUsagePoints are points (cost * 10)
	// Return remaining points directly
	remainingPoints := pointsLimit - currentUsagePoints

	return remainingPoints, nil
}

// refreshCacheInBackground updates cache entry in background
func (uc *UsageChecker) refreshCacheInBackground(userID string) {
	bgCtx := context.Background()
	if freshPoints, err := uc.calculateRemainingPointsFromDB(bgCtx, userID); err == nil {
		// Only cache if not zero (zero limits are not cached)
		if freshPoints != 0 {
			uc.cache.Add(userID, &UsageCacheEntry{
				RemainingPoints: freshPoints,
				Timestamp:       time.Now(),
			})
		}
	}
}

// CheckDailyPointsLimit checks if user has exceeded their daily points limit
// Returns remaining points (negative if over limit, positive if under limit)
func (uc *UsageChecker) CheckDailyPointsLimit(ctx context.Context, userID string) (int, error) {
	// Check cache first
	if entry := uc.cleanupExpiredEntry(userID); entry != nil {
		// If cache is older than 1 minute, refresh in background
		if time.Since(entry.Timestamp) > 1*time.Minute {
			go uc.refreshCacheInBackground(userID)
		}
		return entry.RemainingPoints, nil
	}

	// Calculate from database
	remainingPoints, err := uc.calculateRemainingPointsFromDB(ctx, userID)
	if err != nil {
		return 0, err
	}

	// Cache the result (only if not zero)
	if remainingPoints != 0 {
		uc.cache.Add(userID, &UsageCacheEntry{
			RemainingPoints: remainingPoints,
			Timestamp:       time.Now(),
		})
	}

	return remainingPoints, nil
}

// getCurrentDailyUsage calculates the total points for the current 24-hour period (8pm-8pm UTC)
func (uc *UsageChecker) getCurrentDailyUsage(ctx context.Context, userID string) (int, error) {
	startTime, endTime := uc.getCurrentDailyWindow()

	// Query hourly aggregates for the 8pm-8pm UTC window
	query := uc.client.Collection("hourly_aggregates").
		Where("user_id", "==", userID).
		Where("hour", ">=", startTime).
		Where("hour", "<", endTime)

	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return 0, fmt.Errorf("failed to query hourly aggregates: %w", err)
	}

	var totalPoints int
	for _, doc := range docs {
		data := doc.Data()
		if points, ok := data["total_points"].(int64); ok {
			totalPoints += int(points)
		}
	}

	return totalPoints, nil
}

// getCurrentDailyWindow returns the start and end times for the current 8pm-8pm UTC window
func (uc *UsageChecker) getCurrentDailyWindow() (time.Time, time.Time) {
	now := time.Now().UTC()

	// Find the most recent 8pm (20:00)
	var windowStart time.Time
	if now.Hour() >= 20 {
		// If it's after 8pm today, the window started at 8pm today
		windowStart = time.Date(now.Year(), now.Month(), now.Day(), 20, 0, 0, 0, time.UTC)
	} else {
		// If it's before 8pm today, the window started at 8pm yesterday
		yesterday := now.AddDate(0, 0, -1)
		windowStart = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 20, 0, 0, 0, time.UTC)
	}

	// Window ends 24 hours later
	windowEnd := windowStart.Add(24 * time.Hour)

	return windowStart, windowEnd
}
