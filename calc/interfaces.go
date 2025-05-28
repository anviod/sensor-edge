package calc

// Aggregator 聚合计算接口
// Add: 添加新值，Result: 获取聚合结果

type Aggregator interface {
	Add(value float64)
	Result() float64
}

// RuleEngine 边缘规则引擎接口
// Apply: 应用规则，支持聚合、报警、联动等

type RuleEngine interface {
	Apply(deviceID string, points map[string]interface{}) error
}
