package config

import (
	"os"
	"sensor-edge/types"

	"gopkg.in/yaml.v3"
)

func LoadAggregateRules(file string) ([]types.AggregateRule, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var rules []types.AggregateRule
	err = yaml.Unmarshal(data, &rules)
	return rules, err
}

func LoadAlarmRulesEdge(file string) ([]types.AlarmRuleEdge, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var rules []types.AlarmRuleEdge
	err = yaml.Unmarshal(data, &rules)
	return rules, err
}

func LoadLinkageRules(file string) ([]types.LinkageRule, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var rules []types.LinkageRule
	err = yaml.Unmarshal(data, &rules)
	return rules, err
}

// 新增：加载新版分组式边缘规则
func LoadDeviceEdgeRules(file string) ([]types.DeviceEdgeRules, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var rules []types.DeviceEdgeRules
	err = yaml.Unmarshal(data, &rules)
	return rules, err
}

// 新增：从新版 DeviceEdgeRules 拆分出所有聚合规则
func ExtractAggregateRules(devRules []types.DeviceEdgeRules) []types.AggregateRule {
	var out []types.AggregateRule
	for _, dr := range devRules {
		for _, rule := range dr.Aggregate {
			rule.DeviceID = dr.DeviceID
			rule.Type = "aggregate"
			out = append(out, rule)
		}
	}
	return out
}

// 新增：从新版 DeviceEdgeRules 拆分出所有报警规则
func ExtractAlarmRules(devRules []types.DeviceEdgeRules) []types.AlarmRuleEdge {
	var out []types.AlarmRuleEdge
	for _, dr := range devRules {
		for _, rule := range dr.Alarm {
			rule.DeviceID = dr.DeviceID
			rule.Type = "alarm"
			out = append(out, rule)
		}
	}
	return out
}

// 新增：从新版 DeviceEdgeRules 拆分出所有联动规则
func ExtractLinkageRules(devRules []types.DeviceEdgeRules) []types.LinkageRule {
	var out []types.LinkageRule
	for _, dr := range devRules {
		for _, rule := range dr.Linkage {
			rule.SourceDevice = dr.DeviceID
			rule.Type = "linkage"
			out = append(out, rule)
		}
	}
	return out
}

// configs/edge_rules.yaml
