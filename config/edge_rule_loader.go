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

// configs/edge_rules.yaml
