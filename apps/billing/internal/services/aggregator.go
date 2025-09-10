package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/firestore"
)

// AggregatorService 数据聚合服务
type AggregatorService struct {
	db             *firestore.Client
	billingService *BillingService
}

// HourlyAggregate 每小时聚合数据
type HourlyAggregate struct {
	Hour              time.Time `firestore:"hour" json:"hour"`
	UserID            string    `firestore:"user_id" json:"user_id"`
	TotalRequests     int       `firestore:"total_requests" json:"total_requests"`
	TotalInputTokens  int       `firestore:"total_input_tokens" json:"total_input_tokens"`
	TotalOutputTokens int       `firestore:"total_output_tokens" json:"total_output_tokens"`
	TotalCost         float64   `firestore:"total_cost" json:"total_cost"`
	TotalPoints       int       `firestore:"total_points" json:"total_points"`
	// Note: ModelUsage is stored as flattened fields like "model_usage.{model}.{metric}"
	// due to atomic increment requirements, not as a nested map
	ModelUsage map[string]ModelStats `firestore:"-" json:"model_usage"`
	CreatedAt  time.Time             `firestore:"created_at" json:"created_at"`
	UpdatedAt  time.Time             `firestore:"updated_at" json:"updated_at"`
}

// ModelStats 模型使用统计
type ModelStats struct {
	RequestCount int     `firestore:"request_count" json:"request_count"`
	InputTokens  int     `firestore:"input_tokens" json:"input_tokens"`
	OutputTokens int     `firestore:"output_tokens" json:"output_tokens"`
	TotalCost    float64 `firestore:"total_cost" json:"total_cost"`
	TotalPoints  int     `firestore:"total_points" json:"total_points"`
}

// MemoryAggregate 内存聚合数据
type MemoryAggregate struct {
	UserID               string                      `json:"user_id"`
	Hour                 string                      `json:"hour"`
	TotalRequests        int                         `json:"total_requests"`
	TotalInputTokens     int                         `json:"total_input_tokens"`
	TotalOutputTokens    int                         `json:"total_output_tokens"`
	TotalCacheReadTokens int                         `json:"total_cache_read_tokens"`
	TotalCacheWriteTokens int                        `json:"total_cache_write_tokens"`
	TotalCost            float64                     `json:"total_cost"`
	TotalPoints          int                         `json:"total_points"`
	ModelUsage           map[string]MemoryModelStats `json:"model_usage"`
}

// MemoryModelStats 内存中的模型使用统计
type MemoryModelStats struct {
	RequestCount     int     `json:"request_count"`
	InputTokens      int     `json:"input_tokens"`
	OutputTokens     int     `json:"output_tokens"`
	CacheReadTokens  int     `json:"cache_read_tokens"`
	CacheWriteTokens int     `json:"cache_write_tokens"`
	TotalCost        float64 `json:"total_cost"`
	TotalPoints      int     `json:"total_points"`
}

// MonthlyUsage 月度使用统计
type MonthlyUsage struct {
	UserID            string                `json:"user_id"`
	Year              int                   `json:"year"`
	Month             int                   `json:"month"`
	TotalRequests     int                   `json:"total_requests"`
	TotalInputTokens  int                   `json:"total_input_tokens"`
	TotalOutputTokens int                   `json:"total_output_tokens"`
	TotalCost         float64               `json:"total_cost"`
	HourlyUsage       []HourlyAggregate     `json:"hourly_usage"`
	ModelUsage        map[string]ModelStats `json:"model_usage"`
}

// NewAggregatorService 创建新的聚合服务
func NewAggregatorService(db *firestore.Client, billingService *BillingService) *AggregatorService {
	return &AggregatorService{
		db:             db,
		billingService: billingService,
	}
}

// AggregateRecords 聚合使用记录并更新小时聚合数据
func (as *AggregatorService) AggregateRecords(ctx context.Context, records []*UsageRecord) error {
	if len(records) == 0 {
		return nil
	}

	// Group records by user and hour for aggregation
	aggregateMap := make(map[string]*MemoryAggregate)

	for _, record := range records {
		// 按小时分组
		hourStr := record.Timestamp.Format("2006-01-02T15")
		key := fmt.Sprintf("%s_%s", record.UserID, hourStr)

		aggregate, exists := aggregateMap[key]
		if !exists {
			aggregate = &MemoryAggregate{
				UserID:               record.UserID,
				Hour:                 hourStr,
				TotalRequests:        0,
				TotalInputTokens:     0,
				TotalOutputTokens:    0,
				TotalCacheReadTokens: 0,
				TotalCacheWriteTokens: 0,
				TotalCost:            0.0,
				TotalPoints:          0,
				ModelUsage:           make(map[string]MemoryModelStats),
			}
			aggregateMap[key] = aggregate
		}

		// 在内存中累加数据
		points := ConvertCostToPoints(record.TotalCost)
		aggregate.TotalRequests++
		aggregate.TotalInputTokens += record.InputTokens
		aggregate.TotalOutputTokens += record.OutputTokens
		aggregate.TotalCacheReadTokens += record.CacheReadTokens
		aggregate.TotalCacheWriteTokens += record.CacheWriteTokens
		aggregate.TotalCost += record.TotalCost
		aggregate.TotalPoints += points

		// 更新模型统计数据
		modelStats := aggregate.ModelUsage[record.Model]
		modelStats.RequestCount++
		modelStats.InputTokens += record.InputTokens
		modelStats.OutputTokens += record.OutputTokens
		modelStats.CacheReadTokens += record.CacheReadTokens
		modelStats.CacheWriteTokens += record.CacheWriteTokens
		modelStats.TotalCost += record.TotalCost
		modelStats.TotalPoints += points
		aggregate.ModelUsage[record.Model] = modelStats
	}

	// 对每个小时聚合执行原子增量更新
	for key, memAggregate := range aggregateMap {
		if err := as.atomicIncrementHourlyAggregate(ctx, key, memAggregate); err != nil {
			log.Printf("Error atomically updating hourly aggregate %s: %v", key, err)
			continue
		}
	}

	log.Printf("Successfully aggregated %d records into %d hourly aggregates using atomic increments", len(records), len(aggregateMap))
	return nil
}

// atomicIncrementHourlyAggregate 使用原子增量更新小时聚合文档
func (as *AggregatorService) atomicIncrementHourlyAggregate(ctx context.Context, docID string, memAggregate *MemoryAggregate) error {
	docRef := as.db.Collection("hourly_aggregates").Doc(docID)

	// 构建原子增量和元数据的upsert数据
	upsertData := map[string]interface{}{
		// 原子增量字段
		"total_requests":         firestore.Increment(memAggregate.TotalRequests),
		"total_input_tokens":     firestore.Increment(memAggregate.TotalInputTokens),
		"total_output_tokens":    firestore.Increment(memAggregate.TotalOutputTokens),
		"total_cache_read_tokens": firestore.Increment(memAggregate.TotalCacheReadTokens),
		"total_cache_write_tokens": firestore.Increment(memAggregate.TotalCacheWriteTokens),
		"total_cost":             firestore.Increment(memAggregate.TotalCost),
		"total_points":           firestore.Increment(memAggregate.TotalPoints),

		// 元数据字段
		"user_id":    memAggregate.UserID,
		"updated_at": time.Now(),
	}

	// 解析并设置小时字段
	if hour, err := time.Parse("2006-01-02T15", memAggregate.Hour); err == nil {
		upsertData["hour"] = hour
		upsertData["created_at"] = time.Now()
	}

	// 添加模型相关的原子增量
	for model, stats := range memAggregate.ModelUsage {
		modelPath := fmt.Sprintf("model_usage.%s", model)
		upsertData[fmt.Sprintf("%s.request_count", modelPath)] = firestore.Increment(stats.RequestCount)
		upsertData[fmt.Sprintf("%s.input_tokens", modelPath)] = firestore.Increment(stats.InputTokens)
		upsertData[fmt.Sprintf("%s.output_tokens", modelPath)] = firestore.Increment(stats.OutputTokens)
		upsertData[fmt.Sprintf("%s.cache_read_tokens", modelPath)] = firestore.Increment(stats.CacheReadTokens)
		upsertData[fmt.Sprintf("%s.cache_write_tokens", modelPath)] = firestore.Increment(stats.CacheWriteTokens)
		upsertData[fmt.Sprintf("%s.total_cost", modelPath)] = firestore.Increment(stats.TotalCost)
		upsertData[fmt.Sprintf("%s.total_points", modelPath)] = firestore.Increment(stats.TotalPoints)
	}

	// 使用MergeAll执行upsert操作
	_, err := docRef.Set(ctx, upsertData, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("failed to atomically upsert hourly aggregate: %w", err)
	}

	log.Printf("Atomically upserted hourly aggregate %s: +%d requests, +%d input tokens, +%d output tokens, +$%.6f cost, +%d points",
		docID, memAggregate.TotalRequests, memAggregate.TotalInputTokens, memAggregate.TotalOutputTokens, memAggregate.TotalCost, memAggregate.TotalPoints)

	return nil
}

// GetUserMonthlyUsage 获取用户月度使用统计
func (as *AggregatorService) GetUserMonthlyUsage(ctx context.Context, userID string, year int, month time.Month) (*MonthlyUsage, error) {
	startOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	query := as.db.Collection("hourly_aggregates").
		Where("user_id", "==", userID).
		Where("hour", ">=", startOfMonth).
		Where("hour", "<", endOfMonth)

	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to query hourly aggregates: %w", err)
	}

	monthly := &MonthlyUsage{
		UserID:            userID,
		Year:              year,
		Month:             int(month),
		TotalRequests:     0,
		TotalInputTokens:  0,
		TotalOutputTokens: 0,
		TotalCost:         0,
		HourlyUsage:       make([]HourlyAggregate, 0, len(docs)),
		ModelUsage:        make(map[string]ModelStats),
	}

	for _, doc := range docs {
		var hourly HourlyAggregate
		if err := doc.DataTo(&hourly); err != nil {
			log.Printf("Error parsing hourly aggregate: %v", err)
			continue
		}

		monthly.TotalRequests += hourly.TotalRequests
		monthly.TotalInputTokens += hourly.TotalInputTokens
		monthly.TotalOutputTokens += hourly.TotalOutputTokens
		monthly.TotalCost += hourly.TotalCost
		monthly.HourlyUsage = append(monthly.HourlyUsage, hourly)

		// 合并模型统计
		for model, stats := range hourly.ModelUsage {
			monthlyStats := monthly.ModelUsage[model]
			monthlyStats.RequestCount += stats.RequestCount
			monthlyStats.InputTokens += stats.InputTokens
			monthlyStats.OutputTokens += stats.OutputTokens
			monthlyStats.TotalCost += stats.TotalCost
			monthly.ModelUsage[model] = monthlyStats
		}
	}

	return monthly, nil
}
