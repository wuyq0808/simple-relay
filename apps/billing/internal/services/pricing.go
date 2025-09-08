package services

import (
	"log"
	"strings"
)

// ModelPricing 模型定价信息
type ModelPricing struct {
	InputPricePerMillion  float64 // 每百万输入token的价格
	OutputPricePerMillion float64 // 每百万输出token的价格
}

// PricingCalculator 价格计算器
type PricingCalculator struct {
	// 模型定价映射
	modelPricing map[string]ModelPricing
}

// NewPricingCalculator 创建新的价格计算器
func NewPricingCalculator() *PricingCalculator {
	return &PricingCalculator{
		modelPricing: map[string]ModelPricing{
			// Claude 3.5 系列
			"claude-3-5-sonnet": {
				InputPricePerMillion:  3.0,
				OutputPricePerMillion: 15.0,
			},
			"claude-3-5-sonnet-20241022": {
				InputPricePerMillion:  3.0,
				OutputPricePerMillion: 15.0,
			},
			"claude-3-5-haiku": {
				InputPricePerMillion:  0.80,
				OutputPricePerMillion: 4.0,
			},
			"claude-3-5-haiku-20241022": {
				InputPricePerMillion:  0.80,
				OutputPricePerMillion: 4.0,
			},

			// Claude 4 系列
			"claude-opus-4-1-20250805": {
				InputPricePerMillion:  15.0,
				OutputPricePerMillion: 75.0,
			},
			"claude-sonnet-4-20250514": {
				InputPricePerMillion:  3.0,
				OutputPricePerMillion: 15.0,
			},

			// Claude 3 系列
			"claude-3-opus": {
				InputPricePerMillion:  15.0,
				OutputPricePerMillion: 75.0,
			},
			"claude-3-opus-20240229": {
				InputPricePerMillion:  15.0,
				OutputPricePerMillion: 75.0,
			},
			"claude-3-sonnet": {
				InputPricePerMillion:  3.0,
				OutputPricePerMillion: 15.0,
			},
			"claude-3-sonnet-20240229": {
				InputPricePerMillion:  3.0,
				OutputPricePerMillion: 15.0,
			},
			"claude-3-haiku": {
				InputPricePerMillion:  0.25,
				OutputPricePerMillion: 1.25,
			},
			"claude-3-haiku-20240307": {
				InputPricePerMillion:  0.25,
				OutputPricePerMillion: 1.25,
			},

			// Claude 2 系列
			"claude-2.1": {
				InputPricePerMillion:  8.0,
				OutputPricePerMillion: 24.0,
			},
			"claude-2.0": {
				InputPricePerMillion:  8.0,
				OutputPricePerMillion: 24.0,
			},

			// Claude Instant
			"claude-instant-1.2": {
				InputPricePerMillion:  0.8,
				OutputPricePerMillion: 2.4,
			},
		},
	}
}

// Calculate 计算给定模型和token数量的成本
func (pc *PricingCalculator) Calculate(model string, inputTokens int, outputTokens int) (inputCost float64, outputCost float64) {
	// 转换为小写以进行不区分大小写的匹配
	modelKey := strings.ToLower(model)

	// 获取定价信息
	pricing, exists := pc.modelPricing[modelKey]
	if !exists {
		// 如果找不到精确匹配，尝试基于模型类型的匹配
		pricing = pc.findBestMatchPricing(modelKey)
	}

	// 计算成本（价格是per million tokens）
	inputCost = float64(inputTokens) * pricing.InputPricePerMillion / 1_000_000
	outputCost = float64(outputTokens) * pricing.OutputPricePerMillion / 1_000_000

	return inputCost, outputCost
}

// GetTotalCost 获取总成本
func (pc *PricingCalculator) GetTotalCost(model string, inputTokens int, outputTokens int) float64 {
	inputCost, outputCost := pc.Calculate(model, inputTokens, outputTokens)
	return inputCost + outputCost
}

// findBestMatchPricing 基于模型名称模式查找定价
func (pc *PricingCalculator) findBestMatchPricing(modelKey string) ModelPricing {
	// 基于模型类型的简单模式匹配
	if strings.Contains(modelKey, "opus") {
		// Opus models: $15/$75
		return ModelPricing{
			InputPricePerMillion:  15.0,
			OutputPricePerMillion: 75.0,
		}
	} else if strings.Contains(modelKey, "sonnet") {
		// Sonnet models: $3/$15
		return ModelPricing{
			InputPricePerMillion:  3.0,
			OutputPricePerMillion: 15.0,
		}
	} else if strings.Contains(modelKey, "haiku") {
		// Haiku models: Use latest 3.5 pricing $0.80/$4
		return ModelPricing{
			InputPricePerMillion:  0.80,
			OutputPricePerMillion: 4.0,
		}
	}

	// 默认定价（使用Sonnet的定价作为默认）
	log.Printf("ERROR: Model '%s' doesn't match any known pattern (opus/sonnet/haiku), using default Sonnet pricing ($3/$15 per million tokens)", modelKey)
	return ModelPricing{
		InputPricePerMillion:  3.0,
		OutputPricePerMillion: 15.0,
	}
}
