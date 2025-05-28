package main

import (
	"fmt"
	"os"
	"os/signal"
	"sensor-edge/config"
	"sensor-edge/edgecompute"
	"sensor-edge/protocols"
	"sensor-edge/protocols/modbus"
	"sensor-edge/types"
	"sensor-edge/uplink"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

// toPointConfig 将 types.PointMapping 转为 protocols.PointConfig
func toPointConfig(points []types.PointMapping) []protocols.PointConfig {
	// 显式处理 nil 或空 slice，提升可读性和语义清晰度
	if points == nil {
		return []protocols.PointConfig{}
	}

	out := make([]protocols.PointConfig, 0, len(points))
	for _, p := range points {
		out = append(out, protocols.PointConfig{
			PointID:   p.Name,
			Address:   p.Address,
			Type:      p.Type,
			Unit:      p.Unit,
			Transform: p.Transform,
		})
	}
	return out
}

func main() {
	// 1. 通信协议接入与注册（已在各协议包init中自动完成）

	// 2. 读取全局配置、协议参数、设备清单
	protoConfRaw, err := os.ReadFile("configs/protocols.yaml")
	if err != nil {
		panic(err)
	}
	var protoConf map[string][]map[string]interface{}
	err = yaml.Unmarshal(protoConfRaw, &protoConf)
	if err != nil {
		panic(err)
	}

	// 3. 加载设备元数据，构建设备映射表
	devs, err := config.LoadDevicesFromYAML("configs/devices.yaml")
	if err != nil {
		panic(err)
	}
	devMap := make(map[string]types.DeviceConfigWithMeta)
	for _, d := range devs {
		devMap[d.ID] = d
	}

	// 4. 加载点位物模型映射
	pointSets, _ := config.LoadPointMappings("configs/points.yaml")

	// 5. 加载边缘计算规则
	aggRules, _ := config.LoadAggregateRules("configs/edge_rules.yaml")
	alarmRules, _ := config.LoadAlarmRulesEdge("configs/edge_rules.yaml")
	linkageRules, _ := config.LoadLinkageRules("configs/edge_rules.yaml")
	re := edgecompute.NewRuleEngine(aggRules, alarmRules, linkageRules)

	// 6. 加载上行通道配置
	uplinkCfgs, _ := config.LoadUplinkConfigs("configs/uplinks.yaml")
	uplinkMgr := uplink.NewUplinkManagerFromConfig(uplinkCfgs)

	fmt.Println("[System] Device collection, edge rule engine & uplink started...")

	// 7. 采集主流程：遍历所有点位配置，自动完成协议参数注入、设备实例化、采集、边缘计算、上报
	for _, set := range pointSets {
		devConf, ok := devMap[set.DeviceID]
		if !ok {
			fmt.Printf("[WARN] 点位配置 device_id=%s 未找到对应设备\n", set.DeviceID)
			continue
		}
		// 优先用点位配置的 protocol/protocol_name 覆盖设备配置
		protocol := devConf.Protocol
		if set.Protocol != "" {
			protocol = set.Protocol
		}
		protocolName := devConf.ProtocolName
		if set.ProtocolName != "" {
			protocolName = set.ProtocolName
		}
		// 匹配协议参数
		var protoParams map[string]interface{}
		for _, p := range protoConf[protocol] {
			if name, ok := p["name"].(string); ok && name == protocolName {
				protoParams = p
				break
			}
		}
		if protoParams == nil {
			fmt.Printf("[WARN] 设备 %s 未找到匹配的协议参数实例\n", set.DeviceID)
			continue
		}
		if devConf.Config == nil {
			devConf.Config = make(map[string]interface{})
		}
		// 自动将设备顶层参数注入Config（如slave_id等）
		devMetaMap := map[string]interface{}{}
		devMetaYaml, _ := yaml.Marshal(devConf.DeviceMeta)
		yaml.Unmarshal(devMetaYaml, &devMetaMap)
		for k, v := range devMetaMap {
			if k != "id" && k != "name" && k != "description" && k != "protocol" && k != "protocol_name" && k != "interval" && k != "enable_ping" {
				if _, exists := devConf.Config[k]; !exists {
					devConf.Config[k] = v
				}
			}
		}
		for k, v := range protoParams {
			if _, exists := devConf.Config[k]; !exists {
				devConf.Config[k] = v
			}
		}
		client, err := protocols.Create(protocol)
		if err != nil {
			fmt.Printf("[ERROR] 设备 %s 协议创建失败: %v\n", set.DeviceID, err)
			continue
		}
		err = client.Init(devConf.Config)
		if err != nil {
			fmt.Printf("[ERROR] 设备 %s 协议初始化失败: %v\n", set.DeviceID, err)
			continue
		}
		// 采集点位
		var values []protocols.PointValue
		pointAddrs := make([]string, 0, len(set.Points))
		for _, p := range set.Points {
			pointAddrs = append(pointAddrs, p.Address)
		}
		maxRetry := 3
		for retry := 1; retry <= maxRetry; retry++ {
			values, err = client.(*modbus.ModbusTCP).ReadBatch(set.DeviceID, pointAddrs)
			if err == nil {
				break
			}
			fmt.Printf("[WARN] 设备 %s Modbus 读取失败（第%d次）：%v\n", set.DeviceID, retry, err)
			time.Sleep(2 * time.Second)
		}
		if err != nil {
			fmt.Printf("[ERROR] 设备 %s 采集失败: %v\n", set.DeviceID, err)
			continue
		}
		// 物模型映射、报警、边缘计算
		pointValues := make(map[string]interface{})
		for _, v := range values {
			pointValues[v.PointID] = v.Value
			fmt.Printf("[%s] %s = %v\n", set.DeviceID, v.PointID, v.Value)
		}
		re.ApplyRules(set.DeviceID, pointValues)
		// 编码数据并上报
		payload := uplink.EncodeDataReport(
			set.DeviceID,
			pointValues,
			nil, // 无报警信息
			nil, // 无指标数据
		)
		err = uplinkMgr.SendToAll(payload)
		if err != nil {
			fmt.Printf("[Error] 设备 %s 数据上报失败: %v\n", set.DeviceID, err)
		} else {
			fmt.Printf("[Success] 设备 %s 数据上报成功\n", set.DeviceID)
		}
	}

	// 8. 支持热加载规则（SIGHUP）
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	go func() {
		for range c {
			aggRules, _ = config.LoadAggregateRules("configs/edge_rules.yaml")
			alarmRules, _ = config.LoadAlarmRulesEdge("configs/edge_rules.yaml")
			linkageRules, _ = config.LoadLinkageRules("configs/edge_rules.yaml")
			re.AggRules = aggRules
			re.AlarmRules = alarmRules
			re.LinkageRules = linkageRules
			fmt.Println("[System] Edge rules reloaded!")
		}
	}()

	// 阻塞运行
	select {}
}
