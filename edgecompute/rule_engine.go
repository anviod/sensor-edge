package edgecompute

import (
	"encoding/json"
	"fmt"
	"os"
	"sensor-edge/mapping"
	"sensor-edge/types"
	"time"
)

// RuleEngine 边缘计算规则引擎
// 支持聚合、报警、联动等多种规则
type RuleEngine struct {
	AggRules     []types.AggregateRule
	AlarmRules   []types.AlarmRuleEdge
	LinkageRules []types.LinkageRule
	Buffers      map[string]*types.PointBuffer // key: device.point
}

// NewRuleEngine 创建新的规则引擎实例
func NewRuleEngine(agg []types.AggregateRule, alarm []types.AlarmRuleEdge, linkage []types.LinkageRule) *RuleEngine {
	return &RuleEngine{
		AggRules:     agg,
		AlarmRules:   alarm,
		LinkageRules: linkage,
		Buffers:      make(map[string]*types.PointBuffer),
	}
}

// ApplyRules 应用所有边缘规则
func (r *RuleEngine) ApplyRules(deviceID string, pointMap map[string]any) {
	// 1. 聚合规则
	for _, rule := range r.AggRules {
		if rule.DeviceID == deviceID {
			key := deviceID + "." + rule.Point
			val, ok := pointMap[rule.Point]
			if !ok {
				continue
			}
			fval, ok := val.(float64)
			if !ok {
				continue
			}
			buf, ok := r.Buffers[key]
			if !ok {
				buf = &types.PointBuffer{Size: rule.Window}
				r.Buffers[key] = buf
			}
			buf.Add(fval)
			if rule.Method == "avg" {
				fmt.Printf("[EdgeCalc] %s: %.2f\n", rule.Description, buf.Avg())
			}
		}
	}
	// 2. 报警规则
	for _, rule := range r.AlarmRules {
		if rule.DeviceID == deviceID {
			val, ok := pointMap[rule.Point]
			if !ok {
				continue
			}
			triggered, err := mapping.EvalExpression(rule.Condition, val)
			if err == nil && triggered == true {
				fmt.Printf("[EdgeCalc] %s - %s: %s\n", rule.Description, rule.Level, rule.Message)
			}
		}
	}
	// 3. 联动规则（仅演示输出，不实际写入）
	for _, rule := range r.LinkageRules {
		if rule.SourceDevice == deviceID {
			val, ok := pointMap[rule.SourcePoint]
			if !ok {
				continue
			}
			triggered, err := mapping.EvalExpression(rule.Condition, val)
			if err == nil && triggered == true {
				fmt.Printf("[Linkage] 执行控制 %s.%s ← %v\n", rule.ActionDevice, rule.ActionAddress, rule.ActionValue)
				// 可调用设备写入接口
			}
		}
	}
	// 规则结果本地持久化
	f, _ := os.OpenFile("edge_rule.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	json.NewEncoder(f).Encode(map[string]any{"device": deviceID, "points": pointMap, "ts": time.Now().Unix()})
}
