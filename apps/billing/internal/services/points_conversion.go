package services

import "math"

// ConvertCostToPoints 将成本转换为积分 (内部使用)
// 内部存储：积分 = 成本 * 1,000,000 (使用四舍五入)
// 这提供了更高的精度，避免小成本的精度损失
// 显示时需要除以 1,000,000 得到显示积分 (相当于成本 * 1)
func ConvertCostToPoints(cost float64) int {
	return int(math.Round(cost * 1000000))
}
