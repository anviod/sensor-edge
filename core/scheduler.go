package core

import (
	"fmt"
	"sensor-edge/config"
	"sensor-edge/edgecompute"
	"sensor-edge/mapping"
	"sensor-edge/protocols"
	"sensor-edge/schema"
	"sensor-edge/types"
	"sensor-edge/uplink"
	"time"
)

type DeviceRunner struct {
	ID     string
	Client protocols.Protocol
	Ticker *time.Ticker
	Stop   chan struct{}
}

func StartScheduler(devices []types.DeviceConfig) ([]*DeviceRunner, error) {
	var runners []*DeviceRunner

	// 加载点位物模型映射
	pointSets, _ := config.LoadPointMappings("sensor-edge/config/points.yaml")
	pointMap := make(map[string][]types.PointMapping)
	for _, set := range pointSets {
		pointMap[set.DeviceID] = set.Points
	}

	// 加载边缘计算规则
	aggRules, _ := config.LoadAggregateRules("sensor-edge/config/edge_rules.yaml")
	alarmRules, _ := config.LoadAlarmRulesEdge("sensor-edge/config/edge_rules.yaml")
	linkageRules, _ := config.LoadLinkageRules("sensor-edge/config/edge_rules.yaml")
	re := edgecompute.NewRuleEngine(aggRules, alarmRules, linkageRules)

	for _, dev := range devices {
		client, err := protocols.Create(dev.Protocol)
		if err != nil {
			return nil, fmt.Errorf("protocol %s not supported: %v", dev.Protocol, err)
		}

		err = client.Init(dev.Config)
		if err != nil {
			return nil, fmt.Errorf("init device %s failed: %v", dev.ID, err)
		}

		runner := &DeviceRunner{
			ID:     dev.ID,
			Client: client,
			Ticker: time.NewTicker(time.Duration(dev.Interval) * time.Second),
			Stop:   make(chan struct{}),
		}

		// 启动采集协程
		go func(r *DeviceRunner) {
			for {
				select {
				case <-r.Ticker.C:
					values, err := r.Client.Read(r.ID)
					if err != nil {
						fmt.Printf("[ERROR] [%s] read failed: %v\n", r.ID, err)
						continue
					}
					// 组装原始点位map
					raw := make(map[string]any)
					for _, v := range values {
						raw[v.PointID] = v.Value
					}
					// 物模型映射与报警
					mapped := make(map[string]any)
					for _, p := range pointMap[r.ID] {
						val := raw[p.Address]
						if p.Transform != "" {
							result, err := mapping.EvalExpression(p.Transform, val)
							if err == nil {
								val = result
							}
						}
						if p.Alarm.Enable {
							triggered, err := mapping.EvalExpression(p.Alarm.Condition, val)
							if err == nil && triggered == true {
								fmt.Printf("[ALARM] [%s] %s - %s [%s]\n", r.ID, p.Name, p.Alarm.Message, p.Alarm.Level)
							}
						}
						fmt.Printf("[MAPPED] [%s] %s (%s): %v %s\n", r.ID, p.Name, p.Type, val, p.Unit)
						mapped[p.Name] = val
					}
					// 边缘计算规则引擎
					re.ApplyRules(r.ID, mapped)
				case <-r.Stop:
					return
				}
			}
		}(runner)

		runners = append(runners, runner)
	}

	return runners, nil
}

// 新增：支持边缘规则引擎的采集调度器
func StartSchedulerWithRuleEngine(devices []types.DeviceConfig, re *edgecompute.RuleEngine) ([]*DeviceRunner, error) {
	var runners []*DeviceRunner

	pointSets, _ := config.LoadPointMappings("sensor-edge/config/points.yaml")
	pointMap := make(map[string][]types.PointMapping)
	for _, set := range pointSets {
		pointMap[set.DeviceID] = set.Points
	}

	for _, dev := range devices {
		client, err := protocols.Create(dev.Protocol)
		if err != nil {
			return nil, fmt.Errorf("protocol %s not supported: %v", dev.Protocol, err)
		}
		err = client.Init(dev.Config)
		if err != nil {
			return nil, fmt.Errorf("init device %s failed: %v", dev.ID, err)
		}
		runner := &DeviceRunner{
			ID:     dev.ID,
			Client: client,
			Ticker: time.NewTicker(time.Duration(dev.Interval) * time.Second),
			Stop:   make(chan struct{}),
		}
		go func(r *DeviceRunner) {
			for {
				select {
				case <-r.Ticker.C:
					values, err := r.Client.Read(r.ID)
					if err != nil {
						fmt.Printf("[ERROR] [%s] read failed: %v\n", r.ID, err)
						continue
					}
					raw := make(map[string]any)
					for _, v := range values {
						raw[v.PointID] = v.Value
					}
					// 物模型映射与报警
					mapped := make(map[string]any)
					for _, p := range pointMap[r.ID] {
						val := raw[p.Address]
						if p.Transform != "" {
							result, err := mapping.EvalExpression(p.Transform, val)
							if err == nil {
								val = result
							}
						}
						mapped[p.Name] = val
						if p.Alarm.Enable {
							triggered, err := mapping.EvalExpression(p.Alarm.Condition, val)
							if err == nil && triggered == true {
								fmt.Printf("[ALARM] [%s] %s - %s [%s]\n", r.ID, p.Name, p.Alarm.Message, p.Alarm.Level)
							}
						}
						fmt.Printf("[MAPPED] [%s] %s (%s): %v %s\n", r.ID, p.Name, p.Type, val, p.Unit)
					}
					// 边缘规则引擎处理
					re.ApplyRules(r.ID, mapped)
				case <-r.Stop:
					return
				}
			}
		}(runner)
		runners = append(runners, runner)
	}
	return runners, nil
}

// 新增：支持边缘规则引擎和上报的全流程调度器
func StartSchedulerWithRuleEngineAndUplink(devices []types.DeviceConfig, re *edgecompute.RuleEngine, uplinkMgr *uplink.UplinkManager) ([]*DeviceRunner, error) {
	var runners []*DeviceRunner

	pointSets, _ := config.LoadPointMappings("config/points.yaml")
	pointMap := make(map[string][]types.PointMapping)
	for _, set := range pointSets {
		pointMap[set.DeviceID] = set.Points
	}

	for _, dev := range devices {
		client, err := protocols.Create(dev.Protocol)
		if err != nil {
			return nil, fmt.Errorf("protocol %s not supported: %v", dev.Protocol, err)
		}
		err = client.Init(dev.Config)
		if err != nil {
			return nil, fmt.Errorf("init device %s failed: %v", dev.ID, err)
		}
		runner := &DeviceRunner{
			ID:     dev.ID,
			Client: client,
			Ticker: time.NewTicker(time.Duration(dev.Interval) * time.Second),
			Stop:   make(chan struct{}),
		}
		go func(r *DeviceRunner) {
			for {
				select {
				case <-r.Ticker.C:
					values, err := r.Client.Read(r.ID)
					if err != nil {
						fmt.Printf("[ERROR] [%s] read failed: %v\n", r.ID, err)
						continue
					}
					raw := make(map[string]any)
					for _, v := range values {
						raw[v.PointID] = v.Value
					}
					mapped := make(map[string]any)
					alarms := []schema.AlarmInfo{}
					for _, p := range pointMap[r.ID] {
						val := raw[p.Address]
						if p.Transform != "" {
							result, err := mapping.EvalExpression(p.Transform, val)
							if err == nil {
								val = result
							}
						}
						mapped[p.Name] = val
						if p.Alarm.Enable {
							triggered, err := mapping.EvalExpression(p.Alarm.Condition, val)
							if err == nil && triggered == true {
								alarms = append(alarms, schema.AlarmInfo{Name: p.Name, Level: p.Alarm.Level, Message: p.Alarm.Message})
								fmt.Printf("[ALARM] [%s] %s - %s [%s]\n", r.ID, p.Name, p.Alarm.Message, p.Alarm.Level)
							}
						}
						fmt.Printf("[MAPPED] [%s] %s (%s): %v %s\n", r.ID, p.Name, p.Type, val, p.Unit)
					}
					// 边缘规则引擎
					re.ApplyRules(r.ID, mapped)
					// 上报数据
					uplinkMgr.SendToAll(uplink.EncodeDataReport(r.ID, mapped, alarms, nil))
				case <-r.Stop:
					return
				}
			}
		}(runner)
		runners = append(runners, runner)
	}
	return runners, nil
}

// 支持动态修改采集频率
func (r *DeviceRunner) SetInterval(seconds int) {
	r.Ticker.Stop()
	r.Ticker = time.NewTicker(time.Duration(seconds) * time.Second)
}
