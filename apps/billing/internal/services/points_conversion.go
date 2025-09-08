package services

import "math"

// ConvertCostToPoints 将成本转换为积分
// 当前实现：积分 = 成本 * 10000 (使用四舍五入)
// 此转换逻辑可能在未来发生变化，因此提取到单独的文件中
func ConvertCostToPoints(cost float64) int {
	return int(math.Round(cost * 10000))
}
