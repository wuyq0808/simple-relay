package services

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
)

// UpstreamMinuteAggregatorService handles minute-level aggregation for upstream OAuth accounts
type UpstreamMinuteAggregatorService struct {
	base *UpstreamAggregationBase
}

// NewUpstreamMinuteAggregatorService creates a new upstream minute aggregator service
func NewUpstreamMinuteAggregatorService(db *firestore.Client, billingService *BillingService) *UpstreamMinuteAggregatorService {
	config := UpstreamAggregateConfig{
		CollectionName:  "upstream_account_minute_aggregates",
		TimeFormat:      "2006-01-02T15:04",
		TimeFieldName:   "minute",
		LogDescription:  "minute aggregate",
	}
	return &UpstreamMinuteAggregatorService{
		base: NewUpstreamAggregationBase(db, billingService, config),
	}
}

// UpstreamAccountMinuteAggregate represents minute aggregated data for an upstream account
type UpstreamAccountMinuteAggregate struct {
	Minute               time.Time             `firestore:"minute" json:"minute"`
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

// MemoryUpstreamMinuteAggregate represents in-memory minute aggregation before persistence
type MemoryUpstreamMinuteAggregate struct {
	UpstreamAccountUUID   string                      `json:"upstream_account_uuid"`
	Minute                string                      `json:"minute"`
	TotalRequests         int                         `json:"total_requests"`
	TotalInputTokens      int                         `json:"total_input_tokens"`
	TotalOutputTokens     int                         `json:"total_output_tokens"`
	TotalCacheReadTokens  int                         `json:"total_cache_read_tokens"`
	TotalCacheWriteTokens int                         `json:"total_cache_write_tokens"`
	TotalCost             float64                     `json:"total_cost"`
	TotalPoints           float64                     `json:"total_points"`
	ModelUsage            map[string]MemoryModelStats `json:"model_usage"`
}

// AggregateRecords aggregates usage records by upstream account at minute level
func (umas *UpstreamMinuteAggregatorService) AggregateRecords(ctx context.Context, records []*UsageRecord) error {
	return umas.base.AggregateRecords(ctx, records)
}

