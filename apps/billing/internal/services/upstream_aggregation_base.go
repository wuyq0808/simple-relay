package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/firestore"
)

// UpstreamAggregateConfig defines the configuration for upstream aggregation
type UpstreamAggregateConfig struct {
	CollectionName  string
	TimeFormat      string
	TimeFieldName   string
	LogDescription  string
}

// GenericMemoryUpstreamAggregate represents generic in-memory aggregation before persistence
type GenericMemoryUpstreamAggregate struct {
	UpstreamAccountUUID   string                      `json:"upstream_account_uuid"`
	TimeKey               string                      `json:"time_key"`
	TotalRequests         int                         `json:"total_requests"`
	TotalInputTokens      int                         `json:"total_input_tokens"`
	TotalOutputTokens     int                         `json:"total_output_tokens"`
	TotalCacheReadTokens  int                         `json:"total_cache_read_tokens"`
	TotalCacheWriteTokens int                         `json:"total_cache_write_tokens"`
	TotalCost             float64                     `json:"total_cost"`
	TotalPoints           float64                     `json:"total_points"`
	ModelUsage            map[string]MemoryModelStats `json:"model_usage"`
}

// UpstreamAggregationBase provides shared functionality for upstream aggregation
type UpstreamAggregationBase struct {
	db             *firestore.Client
	billingService *BillingService
	config         UpstreamAggregateConfig
}

// NewUpstreamAggregationBase creates a new base aggregation service
func NewUpstreamAggregationBase(db *firestore.Client, billingService *BillingService, config UpstreamAggregateConfig) *UpstreamAggregationBase {
	return &UpstreamAggregationBase{
		db:             db,
		billingService: billingService,
		config:         config,
	}
}

// AggregateRecords aggregates usage records by upstream account using the configured time granularity
func (uab *UpstreamAggregationBase) AggregateRecords(ctx context.Context, records []*UsageRecord) error {
	if len(records) == 0 {
		return nil
	}

	// Group records by upstream account UUID and time for aggregation
	aggregateMap := make(map[string]*GenericMemoryUpstreamAggregate)

	for _, record := range records {
		// Skip if no upstream account UUID
		if record.UpstreamAccountUUID == "" {
			continue
		}

		// Group by configured time format
		timeStr := record.Timestamp.Format(uab.config.TimeFormat)
		// Use upstream account UUID and time as composite key for document ID
		key := fmt.Sprintf("%s_%s", record.UpstreamAccountUUID, timeStr)

		aggregate, exists := aggregateMap[key]
		if !exists {
			aggregate = &GenericMemoryUpstreamAggregate{
				UpstreamAccountUUID:   record.UpstreamAccountUUID,
				TimeKey:               timeStr,
				TotalRequests:         0,
				TotalInputTokens:      0,
				TotalOutputTokens:     0,
				TotalCacheReadTokens:  0,
				TotalCacheWriteTokens: 0,
				TotalCost:             0.0,
				TotalPoints:           0.0,
				ModelUsage:            make(map[string]MemoryModelStats),
			}
			aggregateMap[key] = aggregate
		}

		// Accumulate data in memory
		points := ConvertCostToPoints(record.TotalCost)
		aggregate.TotalRequests++
		aggregate.TotalInputTokens += record.InputTokens
		aggregate.TotalOutputTokens += record.OutputTokens
		aggregate.TotalCacheReadTokens += record.CacheReadTokens
		aggregate.TotalCacheWriteTokens += record.CacheWriteTokens
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

	// Execute atomic incremental updates for each aggregate
	for key, memAggregate := range aggregateMap {
		if err := uab.atomicIncrementAggregate(ctx, key, memAggregate); err != nil {
			log.Printf("Error atomically updating upstream account %s %s: %v", uab.config.LogDescription, key, err)
			continue
		}
	}

	log.Printf("Successfully aggregated %d records into %d upstream account %s using atomic increments", 
		len(records), len(aggregateMap), uab.config.LogDescription)
	return nil
}

// atomicIncrementAggregate performs atomic incremental updates to aggregate document
func (uab *UpstreamAggregationBase) atomicIncrementAggregate(ctx context.Context, docID string, memAggregate *GenericMemoryUpstreamAggregate) error {
	docRef := uab.db.Collection(uab.config.CollectionName).Doc(docID)

	// Build atomic increment and metadata upsert data
	upsertData := map[string]any{
		// Atomic increment fields
		"total_requests":           firestore.Increment(memAggregate.TotalRequests),
		"total_input_tokens":       firestore.Increment(memAggregate.TotalInputTokens),
		"total_output_tokens":      firestore.Increment(memAggregate.TotalOutputTokens),
		"total_cache_read_tokens":  firestore.Increment(memAggregate.TotalCacheReadTokens),
		"total_cache_write_tokens": firestore.Increment(memAggregate.TotalCacheWriteTokens),
		"total_cost":               firestore.Increment(memAggregate.TotalCost),
		"total_points":             firestore.Increment(memAggregate.TotalPoints),

		// Metadata fields
		"upstream_account_uuid": memAggregate.UpstreamAccountUUID,
		"updated_at":            time.Now(),
	}

	// Parse and set time field
	if parsedTime, err := time.Parse(uab.config.TimeFormat, memAggregate.TimeKey); err == nil {
		upsertData[uab.config.TimeFieldName] = parsedTime
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
		return fmt.Errorf("failed to atomically upsert upstream account %s: %w", uab.config.LogDescription, err)
	}

	log.Printf("Atomically upserted upstream account %s %s: +%d requests, +$%.6f cost",
		uab.config.LogDescription, docID, memAggregate.TotalRequests, memAggregate.TotalCost)

	return nil
}