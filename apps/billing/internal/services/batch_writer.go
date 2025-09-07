package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
)


// BatchWriter 批量写入器，用于优化数据库写入性能
type BatchWriter struct {
	client     *firestore.Client
	buffer     []*UsageRecord
	bufferMu   sync.Mutex
	maxSize    int
	flushTime  time.Duration
	stopChan   chan struct{}
	wg         sync.WaitGroup
	collection string
	aggregator *AggregatorService
}

// NewBatchWriter 创建新的批量写入器
func NewBatchWriter(client *firestore.Client, maxSize int, flushTime time.Duration, billingService *BillingService) *BatchWriter {
	return &BatchWriter{
		client:     client,
		buffer:     make([]*UsageRecord, 0, maxSize),
		maxSize:    maxSize,
		flushTime:  flushTime,
		stopChan:   make(chan struct{}),
		collection: "usage_records",
		aggregator: NewAggregatorService(client, billingService),
	}
}

// Start 启动批量写入器
func (bw *BatchWriter) Start() {
	bw.wg.Add(1)
	go bw.run()
}

// Stop 停止批量写入器
func (bw *BatchWriter) Stop() error {
	close(bw.stopChan)
	bw.wg.Wait()
	
	// 刷新剩余的数据
	return bw.flush()
}

// Add 添加记录到缓冲区
func (bw *BatchWriter) Add(record *UsageRecord) error {
	bw.bufferMu.Lock()
	defer bw.bufferMu.Unlock()
	
	bw.buffer = append(bw.buffer, record)
	
	// 如果缓冲区满了，立即刷新
	if len(bw.buffer) >= bw.maxSize {
		return bw.flushLocked()
	}
	
	return nil
}

// run 运行批量写入器的主循环
func (bw *BatchWriter) run() {
	defer bw.wg.Done()
	
	ticker := time.NewTicker(bw.flushTime)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if err := bw.flush(); err != nil {
				log.Printf("Error flushing batch: %v", err)
			}
		case <-bw.stopChan:
			return
		}
	}
}

// flush 刷新缓冲区到数据库
func (bw *BatchWriter) flush() error {
	bw.bufferMu.Lock()
	defer bw.bufferMu.Unlock()
	
	return bw.flushLocked()
}

// flushLocked 在已加锁的情况下刷新缓冲区
func (bw *BatchWriter) flushLocked() error {
	if len(bw.buffer) == 0 {
		return nil
	}
	
	ctx := context.Background()
	batch := bw.client.Batch()
	
	// 批量添加使用记录文档
	for _, record := range bw.buffer {
		docRef := bw.client.Collection(bw.collection).Doc(record.ID)
		batch.Set(docRef, record)
	}
	
	// 执行批量写入
	_, err := batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit batch: %w", err)
	}
	
	// 使用聚合服务更新小时聚合数据
	// 清空缓冲区前先复制记录
	recordsCopy := make([]*UsageRecord, len(bw.buffer))
	copy(recordsCopy, bw.buffer)
	
	// 清空缓冲区
	bw.buffer = bw.buffer[:0]
	
	// 执行记录聚合 (includes both cost and points)
	if err := bw.aggregator.AggregateRecords(ctx, recordsCopy); err != nil {
		log.Printf("Error aggregating records: %v", err)
		// 聚合失败不阻塞刷新操作，仅记录日志
	}
	
	log.Printf("Successfully flushed %d records to database", len(recordsCopy))
	
	return nil
}

// GetBufferSize 获取当前缓冲区大小
func (bw *BatchWriter) GetBufferSize() int {
	bw.bufferMu.Lock()
	defer bw.bufferMu.Unlock()
	
	return len(bw.buffer)
}

// SetMaxSize 设置最大缓冲区大小
func (bw *BatchWriter) SetMaxSize(size int) {
	bw.bufferMu.Lock()
	defer bw.bufferMu.Unlock()
	
	bw.maxSize = size
	
	// 如果当前缓冲区超过新的大小限制，立即刷新
	if len(bw.buffer) >= bw.maxSize {
		bw.flushLocked()
	}
}

// SetFlushInterval 设置刷新间隔
func (bw *BatchWriter) SetFlushInterval(interval time.Duration) {
	bw.flushTime = interval
	// 注意：这不会立即影响正在运行的定时器，需要重启批量写入器才能生效
}