package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
)

// UsageRecord 记录单次API调用的使用情况
type UsageRecord struct {
	ID               string    `firestore:"id" json:"id"`
	UserID           string    `firestore:"user_id" json:"user_id"`
	ClientIP         string    `firestore:"client_ip" json:"client_ip"`
	Model            string    `firestore:"model" json:"model"`
	InputTokens      int       `firestore:"input_tokens" json:"input_tokens"`
	OutputTokens     int       `firestore:"output_tokens" json:"output_tokens"`
	CacheReadTokens  int       `firestore:"cache_read_tokens" json:"cache_read_tokens"`
	CacheWriteTokens int       `firestore:"cache_write_tokens" json:"cache_write_tokens"`
	TotalCost        float64   `firestore:"total_cost" json:"total_cost"`
	InputCost        float64   `firestore:"input_cost" json:"input_cost"`
	OutputCost       float64   `firestore:"output_cost" json:"output_cost"`
	RequestID        string    `firestore:"request_id" json:"request_id"`
	Timestamp        time.Time `firestore:"timestamp" json:"timestamp"`
	Status           string    `firestore:"status" json:"status"`
	ErrorMessage     string    `firestore:"error_message,omitempty" json:"error_message,omitempty"`
}

// ClaudeAPIResponse Claude API响应结构
type ClaudeAPIResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Content []struct {
		Text string `json:"text"`
		Type string `json:"type"`
	} `json:"content"`
	Usage struct {
		InputTokens             int `json:"input_tokens"`
		OutputTokens            int `json:"output_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
		CacheReadInputTokens    int `json:"cache_read_input_tokens,omitempty"`
	} `json:"usage"`
	StopReason string `json:"stop_reason"`
}

// ClaudeAPIRequest Claude API请求结构
type ClaudeAPIRequest struct {
	Model    string      `json:"model"`
	Messages []Message   `json:"messages"`
	System   string      `json:"system,omitempty"`
	MaxTokens int        `json:"max_tokens"`
}

// Message 消息结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// BillingService 计费服务
type BillingService struct {
	dbService   *DatabaseService
	batchWriter *BatchWriter
	pricing     *PricingCalculator
	mu          sync.RWMutex
	enabled     bool
}

// NewBillingService 创建新的计费服务
func NewBillingService(dbService *DatabaseService, enabled bool) *BillingService {
	service := &BillingService{
		dbService: dbService,
		pricing:   NewPricingCalculator(),
		enabled:   enabled,
	}
	
	// 初始化批量写入器
	if enabled && dbService != nil {
		service.batchWriter = NewBatchWriter(dbService.client, 100, 5*time.Second)
		service.batchWriter.Start()
	}
	
	return service
}

// RecordUsage 记录API使用情况
func (bs *BillingService) RecordUsage(ctx context.Context, record *UsageRecord) error {
	if !bs.enabled {
		return nil
	}

	bs.mu.Lock()
	defer bs.mu.Unlock()

	// 计算成本
	inputCost, outputCost := bs.pricing.Calculate(record.Model, record.InputTokens, record.OutputTokens)
	record.InputCost = inputCost
	record.OutputCost = outputCost
	record.TotalCost = inputCost + outputCost

	// 添加到批量写入队列
	return bs.batchWriter.Add(record)
}

// ProcessResponse 处理Claude API响应并提取计费信息
func (bs *BillingService) ProcessResponse(responseBody []byte, requestModel string, userID string, clientIP string, requestID string) (*UsageRecord, error) {
	var response ClaudeAPIResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 如果响应中没有model字段，使用请求中的model
	if response.Model == "" {
		response.Model = requestModel
	}

	record := &UsageRecord{
		ID:               fmt.Sprintf("%s_%d", requestID, time.Now().UnixNano()),
		UserID:           userID,
		ClientIP:         clientIP,
		Model:            response.Model,
		InputTokens:      response.Usage.InputTokens,
		OutputTokens:     response.Usage.OutputTokens,
		CacheReadTokens:  response.Usage.CacheReadInputTokens,
		CacheWriteTokens: response.Usage.CacheCreationInputTokens,
		RequestID:        requestID,
		Timestamp:        time.Now(),
		Status:           "success",
	}

	return record, nil
}

// GetUserUsage 获取用户使用统计
func (bs *BillingService) GetUserUsage(ctx context.Context, userID string, startTime, endTime time.Time) ([]UsageRecord, error) {
	if !bs.enabled || bs.dbService == nil {
		return []UsageRecord{}, nil
	}

	query := bs.dbService.Client().Collection("usage_records").
		Where("user_id", "==", userID).
		Where("timestamp", ">=", startTime).
		Where("timestamp", "<=", endTime).
		OrderBy("timestamp", firestore.Desc)

	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to query usage records: %w", err)
	}

	var records []UsageRecord
	for _, doc := range docs {
		var record UsageRecord
		if err := doc.DataTo(&record); err != nil {
			log.Printf("Error parsing usage record: %v", err)
			continue
		}
		records = append(records, record)
	}

	return records, nil
}

// GetDailyAggregate 获取每日聚合数据
func (bs *BillingService) GetDailyAggregate(ctx context.Context, userID string, date time.Time) (map[string]interface{}, error) {
	if !bs.enabled || bs.dbService == nil {
		return nil, nil
	}

	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	records, err := bs.GetUserUsage(ctx, userID, startOfDay, endOfDay)
	if err != nil {
		return nil, err
	}

	// 聚合统计
	aggregate := map[string]interface{}{
		"date":          startOfDay,
		"user_id":       userID,
		"total_requests": len(records),
		"total_input_tokens": 0,
		"total_output_tokens": 0,
		"total_cost":    0.0,
		"models_used":   make(map[string]int),
	}

	for _, record := range records {
		aggregate["total_input_tokens"] = aggregate["total_input_tokens"].(int) + record.InputTokens
		aggregate["total_output_tokens"] = aggregate["total_output_tokens"].(int) + record.OutputTokens
		aggregate["total_cost"] = aggregate["total_cost"].(float64) + record.TotalCost
		
		modelsUsed := aggregate["models_used"].(map[string]int)
		modelsUsed[record.Model]++
	}

	return aggregate, nil
}

// Close 关闭计费服务
func (bs *BillingService) Close() error {
	if bs.batchWriter != nil {
		return bs.batchWriter.Stop()
	}
	return nil
}