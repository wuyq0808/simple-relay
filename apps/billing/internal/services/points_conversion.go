package services

// ConvertCostToPoints 将成本转换为积分
// 积分 = 成本 * 10
func ConvertCostToPoints(cost float64) float64 {
	return cost * 10
}
