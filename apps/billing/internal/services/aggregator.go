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
	db              *firestore.Client
	billingService  *BillingService
	aggregateInterval time.Duration
	stopChan        chan struct{}
}

// DailyAggregate 每日聚合数据
type DailyAggregate struct {
	Date             time.Time          `firestore:"date" json:"date"`
	UserID           string             `firestore:"user_id" json:"user_id"`
	TotalRequests    int                `firestore:"total_requests" json:"total_requests"`
	TotalInputTokens int                `firestore:"total_input_tokens" json:"total_input_tokens"`
	TotalOutputTokens int               `firestore:"total_output_tokens" json:"total_output_tokens"`
	TotalCost        float64            `firestore:"total_cost" json:"total_cost"`
	ModelUsage       map[string]ModelStats `firestore:"model_usage" json:"model_usage"`
	CreatedAt        time.Time          `firestore:"created_at" json:"created_at"`
	UpdatedAt        time.Time          `firestore:"updated_at" json:"updated_at"`
}

// ModelStats 模型使用统计
type ModelStats struct {
	RequestCount  int     `firestore:"request_count" json:"request_count"`
	InputTokens   int     `firestore:"input_tokens" json:"input_tokens"`
	OutputTokens  int     `firestore:"output_tokens" json:"output_tokens"`
	TotalCost     float64 `firestore:"total_cost" json:"total_cost"`
}

// NewAggregatorService 创建新的聚合服务
func NewAggregatorService(db *firestore.Client, billingService *BillingService, interval time.Duration) *AggregatorService {
	return &AggregatorService{
		db:                db,
		billingService:    billingService,
		aggregateInterval: interval,
		stopChan:          make(chan struct{}),
	}
}

// Start 启动聚合服务
func (as *AggregatorService) Start() {
	go as.run()
	log.Println("Aggregator service started")
}

// Stop 停止聚合服务
func (as *AggregatorService) Stop() {
	close(as.stopChan)
	log.Println("Aggregator service stopped")
}

// run 运行聚合服务主循环
func (as *AggregatorService) run() {
	ticker := time.NewTicker(as.aggregateInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if err := as.performAggregation(); err != nil {
				log.Printf("Error performing aggregation: %v", err)
			}
		case <-as.stopChan:
			return
		}
	}
}

// performAggregation 执行聚合操作
func (as *AggregatorService) performAggregation() error {
	ctx := context.Background()
	now := time.Now()
	
	// 聚合前一小时的数据
	endTime := now.Truncate(time.Hour)
	startTime := endTime.Add(-time.Hour)
	
	log.Printf("Starting aggregation for period: %s to %s", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	
	// 获取所有需要聚合的用户
	users, err := as.getActiveUsers(ctx, startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to get active users: %w", err)
	}
	
	// 为每个用户执行聚合
	for _, userID := range users {
		if err := as.aggregateUserData(ctx, userID, startTime, endTime); err != nil {
			log.Printf("Error aggregating data for user %s: %v", userID, err)
			continue
		}
	}
	
	log.Printf("Aggregation completed for %d users", len(users))
	return nil
}

// getActiveUsers 获取在指定时间段内有活动的用户列表
func (as *AggregatorService) getActiveUsers(ctx context.Context, startTime, endTime time.Time) ([]string, error) {
	query := as.db.Collection("usage_records").
		Where("timestamp", ">=", startTime).
		Where("timestamp", "<", endTime).
		Select("user_id")
	
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	
	// 使用map去重
	userMap := make(map[string]bool)
	for _, doc := range docs {
		userID, ok := doc.Data()["user_id"].(string)
		if ok && userID != "" {
			userMap[userID] = true
		}
	}
	
	// 转换为slice
	users := make([]string, 0, len(userMap))
	for userID := range userMap {
		users = append(users, userID)
	}
	
	return users, nil
}

// aggregateUserData 聚合单个用户的数据
func (as *AggregatorService) aggregateUserData(ctx context.Context, userID string, startTime, endTime time.Time) error {
	// 获取用户在该时间段的所有记录
	records, err := as.billingService.GetUserUsage(ctx, userID, startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to get user usage: %w", err)
	}
	
	if len(records) == 0 {
		return nil // 没有数据需要聚合
	}
	
	// 创建聚合数据
	aggregate := &DailyAggregate{
		Date:             startTime.Truncate(24 * time.Hour),
		UserID:           userID,
		TotalRequests:    len(records),
		TotalInputTokens: 0,
		TotalOutputTokens: 0,
		TotalCost:        0,
		ModelUsage:       make(map[string]ModelStats),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	
	// 计算聚合数据
	for _, record := range records {
		aggregate.TotalInputTokens += record.InputTokens
		aggregate.TotalOutputTokens += record.OutputTokens
		aggregate.TotalCost += record.TotalCost
		
		// 更新模型统计
		stats := aggregate.ModelUsage[record.Model]
		stats.RequestCount++
		stats.InputTokens += record.InputTokens
		stats.OutputTokens += record.OutputTokens
		stats.TotalCost += record.TotalCost
		aggregate.ModelUsage[record.Model] = stats
	}
	
	// 保存到数据库
	docID := fmt.Sprintf("%s_%s", userID, startTime.Format("2006-01-02"))
	docRef := as.db.Collection("daily_aggregates").Doc(docID)
	
	_, err = docRef.Set(ctx, aggregate)
	if err != nil {
		return fmt.Errorf("failed to save aggregate: %w", err)
	}
	
	log.Printf("Saved aggregate for user %s: %d requests, $%.4f total cost", 
		userID, aggregate.TotalRequests, aggregate.TotalCost)
	
	return nil
}

// GetUserMonthlyUsage 获取用户月度使用统计
func (as *AggregatorService) GetUserMonthlyUsage(ctx context.Context, userID string, year int, month time.Month) (*MonthlyUsage, error) {
	startOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)
	
	query := as.db.Collection("daily_aggregates").
		Where("user_id", "==", userID).
		Where("date", ">=", startOfMonth).
		Where("date", "<", endOfMonth)
	
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to query daily aggregates: %w", err)
	}
	
	monthly := &MonthlyUsage{
		UserID:           userID,
		Year:             year,
		Month:            int(month),
		TotalRequests:    0,
		TotalInputTokens: 0,
		TotalOutputTokens: 0,
		TotalCost:        0,
		DailyUsage:       make([]DailyAggregate, 0, len(docs)),
		ModelUsage:       make(map[string]ModelStats),
	}
	
	for _, doc := range docs {
		var daily DailyAggregate
		if err := doc.DataTo(&daily); err != nil {
			log.Printf("Error parsing daily aggregate: %v", err)
			continue
		}
		
		monthly.TotalRequests += daily.TotalRequests
		monthly.TotalInputTokens += daily.TotalInputTokens
		monthly.TotalOutputTokens += daily.TotalOutputTokens
		monthly.TotalCost += daily.TotalCost
		monthly.DailyUsage = append(monthly.DailyUsage, daily)
		
		// 合并模型统计
		for model, stats := range daily.ModelUsage {
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

// MonthlyUsage 月度使用统计
type MonthlyUsage struct {
	UserID            string                `json:"user_id"`
	Year              int                   `json:"year"`
	Month             int                   `json:"month"`
	TotalRequests     int                   `json:"total_requests"`
	TotalInputTokens  int                   `json:"total_input_tokens"`
	TotalOutputTokens int                   `json:"total_output_tokens"`
	TotalCost         float64               `json:"total_cost"`
	DailyUsage        []DailyAggregate      `json:"daily_usage"`
	ModelUsage        map[string]ModelStats `json:"model_usage"`
}