package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/firestore"
)

// UpstreamAggregatorService handles aggregation for upstream OAuth accounts
type UpstreamAggregatorService struct {
	db             *firestore.Client
	billingService *BillingService
}

// NewUpstreamAggregatorService creates a new upstream aggregator service
func NewUpstreamAggregatorService(db *firestore.Client, billingService *BillingService) *UpstreamAggregatorService {
	return &UpstreamAggregatorService{
		db:             db,
		billingService: billingService,
	}
}

// UpstreamAccountHourlyAggregate represents hourly aggregated data for an upstream account
type UpstreamAccountHourlyAggregate struct {
	Hour                 time.Time             `firestore:"hour" json:"hour"`
	UpstreamAccountUUID  string                `firestore:"upstream_account_uuid" json:"upstream_account_uuid"`
	TotalRequests        int                   `firestore:"total_requests" json:"total_requests"`
	TotalInputTokens     int                   `firestore:"total_input_tokens" json:"total_input_tokens"`
	TotalOutputTokens    int                   `firestore:"total_output_tokens" json:"total_output_tokens"`
	TotalCost            float64               `firestore:"total_cost" json:"total_cost"`
	TotalPoints          int                   `firestore:"total_points" json:"total_points"`
	ModelUsage           map[string]ModelStats `firestore:"-" json:"model_usage"`
	CreatedAt            time.Time             `firestore:"created_at" json:"created_at"`
	UpdatedAt            time.Time             `firestore:"updated_at" json:"updated_at"`
}

// MemoryUpstreamAggregate represents in-memory aggregation before persistence
type MemoryUpstreamAggregate struct {
	UpstreamAccountUUID  string                      `json:"upstream_account_uuid"`
	Hour                 string                      `json:"hour"`
	TotalRequests        int                         `json:"total_requests"`
	TotalInputTokens     int                         `json:"total_input_tokens"`
	TotalOutputTokens    int                         `json:"total_output_tokens"`
	TotalCost            float64                     `json:"total_cost"`
	TotalPoints          int                         `json:"total_points"`
	ModelUsage           map[string]MemoryModelStats `json:"model_usage"`
}

// AggregateRecords aggregates usage records by upstream account
func (uas *UpstreamAggregatorService) AggregateRecords(ctx context.Context, records []*UsageRecord) error {
	if len(records) == 0 {
		return nil
	}

	// Group records by upstream account UUID and hour for aggregation
	aggregateMap := make(map[string]*MemoryUpstreamAggregate)

	for _, record := range records {
		// Skip if no upstream account UUID
		if record.UpstreamAccountUUID == "" {
			continue
		}

		// Group by hour
		hourStr := record.Timestamp.Format("2006-01-02T15")
		// Use upstream account UUID and hour as composite key for document ID
		key := fmt.Sprintf("%s_%s", record.UpstreamAccountUUID, hourStr)

		aggregate, exists := aggregateMap[key]
		if !exists {
			aggregate = &MemoryUpstreamAggregate{
				UpstreamAccountUUID: record.UpstreamAccountUUID,
				Hour:                hourStr,
				TotalRequests:       0,
				TotalInputTokens:    0,
				TotalOutputTokens:   0,
				TotalCost:           0.0,
				TotalPoints:         0,
				ModelUsage:          make(map[string]MemoryModelStats),
			}
			aggregateMap[key] = aggregate
		}

		// Accumulate data in memory
		points := ConvertCostToPoints(record.TotalCost)
		aggregate.TotalRequests++
		aggregate.TotalInputTokens += record.InputTokens
		aggregate.TotalOutputTokens += record.OutputTokens
		aggregate.TotalCost += record.TotalCost
		aggregate.TotalPoints += points

		// Update model statistics
		modelStats := aggregate.ModelUsage[record.Model]
		modelStats.RequestCount++
		modelStats.InputTokens += record.InputTokens
		modelStats.OutputTokens += record.OutputTokens
		modelStats.TotalCost += record.TotalCost
		modelStats.TotalPoints += points
		aggregate.ModelUsage[record.Model] = modelStats
	}

	// Execute atomic incremental updates for each hourly aggregate
	for key, memAggregate := range aggregateMap {
		if err := uas.atomicIncrementHourlyAggregate(ctx, key, memAggregate); err != nil {
			log.Printf("Error atomically updating upstream account hourly aggregate %s: %v", key, err)
			continue
		}
	}

	log.Printf("Successfully aggregated %d records into %d upstream account hourly aggregates using atomic increments", 
		len(records), len(aggregateMap))
	return nil
}

// atomicIncrementHourlyAggregate performs atomic incremental updates to hourly aggregate document
func (uas *UpstreamAggregatorService) atomicIncrementHourlyAggregate(ctx context.Context, docID string, memAggregate *MemoryUpstreamAggregate) error {
	docRef := uas.db.Collection("upstream_account_hourly_aggregates").Doc(docID)

	// Build atomic increment and metadata upsert data
	upsertData := map[string]interface{}{
		// Atomic increment fields
		"total_requests":      firestore.Increment(memAggregate.TotalRequests),
		"total_input_tokens":  firestore.Increment(memAggregate.TotalInputTokens),
		"total_output_tokens": firestore.Increment(memAggregate.TotalOutputTokens),
		"total_cost":          firestore.Increment(memAggregate.TotalCost),
		"total_points":        firestore.Increment(memAggregate.TotalPoints),

		// Metadata fields
		"upstream_account_uuid": memAggregate.UpstreamAccountUUID,
		"updated_at":            time.Now(),
	}

	// Parse and set hour field
	if hour, err := time.Parse("2006-01-02T15", memAggregate.Hour); err == nil {
		upsertData["hour"] = hour
		upsertData["created_at"] = time.Now()
	}

	// Add model-related atomic increments
	for model, stats := range memAggregate.ModelUsage {
		modelPath := fmt.Sprintf("model_usage.%s", model)
		upsertData[fmt.Sprintf("%s.request_count", modelPath)] = firestore.Increment(stats.RequestCount)
		upsertData[fmt.Sprintf("%s.input_tokens", modelPath)] = firestore.Increment(stats.InputTokens)
		upsertData[fmt.Sprintf("%s.output_tokens", modelPath)] = firestore.Increment(stats.OutputTokens)
		upsertData[fmt.Sprintf("%s.total_cost", modelPath)] = firestore.Increment(stats.TotalCost)
		upsertData[fmt.Sprintf("%s.total_points", modelPath)] = firestore.Increment(stats.TotalPoints)
	}

	// Execute upsert operation with MergeAll
	_, err := docRef.Set(ctx, upsertData, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("failed to atomically upsert upstream account hourly aggregate: %w", err)
	}

	log.Printf("Atomically upserted upstream account hourly aggregate %s: +%d requests, +$%.6f cost",
		docID, memAggregate.TotalRequests, memAggregate.TotalCost)

	return nil
}

