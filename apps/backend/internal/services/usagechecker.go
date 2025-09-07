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
	RemainingCost float64
	Timestamp     time.Time
}

// UsageChecker handles daily cost limit checking
type UsageChecker struct {
	client           *firestore.Client
	costLimitService *CostLimitService
	cache            *lru.Cache[string, *UsageCacheEntry]
	cacheDuration    time.Duration
}

// NewUsageChecker creates a new usage checker
func NewUsageChecker(client *firestore.Client) *UsageChecker {
	// Create LRU cache with capacity of 1000 entries
	cache, _ := lru.New[string, *UsageCacheEntry](1000)

	return &UsageChecker{
		client:           client,
		costLimitService: NewCostLimitService(client),
		cache:            cache,
		cacheDuration:    24 * time.Hour, // 24 hour cache
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

// calculateRemainingCostFromDB calculates remaining cost by querying database
func (uc *UsageChecker) calculateRemainingCostFromDB(ctx context.Context, userID string) (float64, error) {
	// Get user's cost limit (defaults to 0 if not set)
	limitAmount, err := uc.costLimitService.GetCostLimit(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("error getting cost limit: %w", err)
	}

	// If limit is 0, return 0 directly (no usage allowed) - don't cache
	if limitAmount == 0 {
		return 0, nil
	}

	// Calculate current 24-hour usage (8pm-8pm UTC window)
	currentUsage, err := uc.getCurrentDailyUsage(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("error getting current usage: %w", err)
	}

	// Calculate remaining cost (positive = under limit, negative = over limit)
	remainingCost := limitAmount - currentUsage

	return remainingCost, nil
}

// refreshCacheInBackground updates cache entry in background
func (uc *UsageChecker) refreshCacheInBackground(userID string) {
	bgCtx := context.Background()
	if freshCost, err := uc.calculateRemainingCostFromDB(bgCtx, userID); err == nil {
		// Only cache if not zero (zero limits are not cached)
		if freshCost != 0 {
			uc.cache.Add(userID, &UsageCacheEntry{
				RemainingCost: freshCost,
				Timestamp:     time.Now(),
			})
		}
	}
}

// CheckDailyCostLimit checks if user has exceeded their daily cost limit
// Returns remaining cost (negative if over limit, positive if under limit)
func (uc *UsageChecker) CheckDailyCostLimit(ctx context.Context, userID string) (float64, error) {
	// Check cache first
	if entry := uc.cleanupExpiredEntry(userID); entry != nil {
		// If cache is older than 1 minute, refresh in background
		if time.Since(entry.Timestamp) > 1*time.Minute {
			go uc.refreshCacheInBackground(userID)
		}
		return entry.RemainingCost, nil
	}

	// Calculate from database
	remainingCost, err := uc.calculateRemainingCostFromDB(ctx, userID)
	if err != nil {
		return 0, err
	}

	// Cache the result (only if not zero)
	if remainingCost != 0 {
		uc.cache.Add(userID, &UsageCacheEntry{
			RemainingCost: remainingCost,
			Timestamp:     time.Now(),
		})
	}

	return remainingCost, nil
}

// getCurrentDailyUsage calculates the total cost for the current 24-hour period (8pm-8pm UTC)
func (uc *UsageChecker) getCurrentDailyUsage(ctx context.Context, userID string) (float64, error) {
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

	var totalCost float64
	for _, doc := range docs {
		data := doc.Data()
		if cost, ok := data["total_cost"].(float64); ok {
			totalCost += cost
		}
	}

	return totalCost, nil
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
