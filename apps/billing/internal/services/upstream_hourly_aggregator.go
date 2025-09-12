package services

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
)

// UpstreamHourlyAggregatorService handles hourly aggregation for upstream OAuth accounts
type UpstreamHourlyAggregatorService struct {
	base *UpstreamAggregationBase
}

// NewUpstreamHourlyAggregatorService creates a new upstream hourly aggregator service
func NewUpstreamHourlyAggregatorService(db *firestore.Client, billingService *BillingService) *UpstreamHourlyAggregatorService {
	config := UpstreamAggregateConfig{
		CollectionName:  "upstream_account_hourly_aggregates",
		TimeFormat:      "2006-01-02T15",
		TimeFieldName:   "hour",
		LogDescription:  "hourly aggregate",
	}
	return &UpstreamHourlyAggregatorService{
		base: NewUpstreamAggregationBase(db, billingService, config),
	}
}

// UpstreamAccountHourlyAggregate represents hourly aggregated data for an upstream account
type UpstreamAccountHourlyAggregate struct {
	Hour                 time.Time             `firestore:"hour" json:"hour"`
	UpstreamAccountUUID  string                `firestore:"upstream_account_uuid" json:"upstream_account_uuid"`
	TotalRequests        int                   `firestore:"total_requests" json:"total_requests"`
	TotalInputTokens     int                   `firestore:"total_input_tokens" json:"total_input_tokens"`
	TotalOutputTokens    int                   `firestore:"total_output_tokens" json:"total_output_tokens"`
	TotalCacheReadTokens int                   `firestore:"total_cache_read_tokens" json:"total_cache_read_tokens"`
	TotalCacheWriteTokens int                  `firestore:"total_cache_write_tokens" json:"total_cache_write_tokens"`
	TotalCost            float64               `firestore:"total_cost" json:"total_cost"`
	TotalPoints          float64               `firestore:"total_points" json:"total_points"`
	ModelUsage           map[string]ModelStats `firestore:"-" json:"model_usage"`
	CreatedAt            time.Time             `firestore:"created_at" json:"created_at"`
	UpdatedAt            time.Time             `firestore:"updated_at" json:"updated_at"`
}

// MemoryUpstreamHourlyAggregate represents in-memory hourly aggregation before persistence
type MemoryUpstreamHourlyAggregate struct {
	UpstreamAccountUUID   string                      `json:"upstream_account_uuid"`
	Hour                  string                      `json:"hour"`
	TotalRequests         int                         `json:"total_requests"`
	TotalInputTokens      int                         `json:"total_input_tokens"`
	TotalOutputTokens     int                         `json:"total_output_tokens"`
	TotalCacheReadTokens  int                         `json:"total_cache_read_tokens"`
	TotalCacheWriteTokens int                         `json:"total_cache_write_tokens"`
	TotalCost             float64                     `json:"total_cost"`
	TotalPoints           float64                     `json:"total_points"`
	ModelUsage            map[string]MemoryModelStats `json:"model_usage"`
}

// AggregateRecords aggregates usage records by upstream account
func (uhas *UpstreamHourlyAggregatorService) AggregateRecords(ctx context.Context, records []*UsageRecord) error {
	return uhas.base.AggregateRecords(ctx, records)
}


