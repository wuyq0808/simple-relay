package services

import (
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
				InputPricePerMillion:  1.0,
				OutputPricePerMillion: 5.0,
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
	// 标准化模型名称
	modelKey := pc.normalizeModelName(model)

	// 获取定价信息
	pricing, exists := pc.modelPricing[modelKey]
	if !exists {
		// 如果找不到精确匹配，尝试部分匹配
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

// normalizeModelName 标准化模型名称
func (pc *PricingCalculator) normalizeModelName(model string) string {
	// 转换为小写
	model = strings.ToLower(model)

	// 移除常见的版本后缀变体
	model = strings.TrimSuffix(model, "-latest")

	// 如果包含日期格式但不在我们的映射中，尝试提取基础模型名
	if strings.Contains(model, "-20") {
		parts := strings.Split(model, "-20")
		baseModel := parts[0]

		// 检查是否有基础模型的定价
		if _, exists := pc.modelPricing[baseModel]; exists {
			return baseModel
		}
	}

	return model
}

// findBestMatchPricing 查找最匹配的定价
func (pc *PricingCalculator) findBestMatchPricing(modelKey string) ModelPricing {
	// 尝试查找包含关系
	for key, pricing := range pc.modelPricing {
		if strings.Contains(modelKey, key) || strings.Contains(key, modelKey) {
			return pricing
		}
	}

	// 默认定价（使用Sonnet的定价作为默认）
	return ModelPricing{
		InputPricePerMillion:  3.0,
		OutputPricePerMillion: 15.0,
	}
}

// GetModelPricing 获取特定模型的定价信息
func (pc *PricingCalculator) GetModelPricing(model string) (ModelPricing, bool) {
	modelKey := pc.normalizeModelName(model)
	pricing, exists := pc.modelPricing[modelKey]
	return pricing, exists
}

// UpdateModelPricing 更新模型定价（用于动态调整价格）
func (pc *PricingCalculator) UpdateModelPricing(model string, inputPrice, outputPrice float64) {
	modelKey := pc.normalizeModelName(model)
	pc.modelPricing[modelKey] = ModelPricing{
		InputPricePerMillion:  inputPrice,
		OutputPricePerMillion: outputPrice,
	}
}

// GetSupportedModels 获取所有支持的模型列表
func (pc *PricingCalculator) GetSupportedModels() []string {
	models := make([]string, 0, len(pc.modelPricing))
	for model := range pc.modelPricing {
		models = append(models, model)
	}
	return models
}
