package main

import (
	"fmt"
	"os"
	"os/signal"
	"sensor-edge/config"
	"sensor-edge/core"
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
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		panic(err)
	}

	uplinkCfgs, _ := config.LoadUplinkConfigs("configs/uplinks.yaml")
	uplinkMgr := uplink.NewUplinkManagerFromConfig(uplinkCfgs)

	aggRules, _ := config.LoadAggregateRules("configs/edge_rules.yaml")
	alarmRules, _ := config.LoadAlarmRulesEdge("configs/edge_rules.yaml")
	linkageRules, _ := config.LoadLinkageRules("configs/edge_rules.yaml")
	re := edgecompute.NewRuleEngine(aggRules, alarmRules, linkageRules)

	runners, err := core.StartSchedulerWithRuleEngineAndUplink(cfg.Devices, re, uplinkMgr)
	if err != nil {
		fmt.Println(err)
	}
	_ = runners

	fmt.Println("[System] Device collection, edge rule engine & uplink started...")

	// 支持热加载规则（SIGHUP）
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

	// ========== DEMO: 读取 modbus1 设备所有温度点位并上报 ===========
	fmt.Println("[Demo] 读取并上报 modbus1 设备所有温度点位:")
	pointSets, _ := config.LoadPointMappings("configs/points.yaml")
	var modbusPoints []types.PointMapping
	for _, set := range pointSets {
		if set.DeviceID == "modbus1" {
			modbusPoints = set.Points
		}
	}
	if len(modbusPoints) > 0 {
		// 读取协议参数（如 interval/timeout）
		protoConfRaw, err := os.ReadFile("configs/protocols.yaml")
		if err != nil {
			panic(err)
		}
		var protoConf map[string]map[string]interface{}
		err = yaml.Unmarshal(protoConfRaw, &protoConf)
		if err != nil {
			panic(err)
		}
		modbusTCPConf := protoConf["modbus_tcp"]
		interval := 1
		if v, ok := modbusTCPConf["interval"]; ok {
			switch vv := v.(type) {
			case int:
				interval = vv
			case float64:
				interval = int(vv)
			}
		}
		timeout := 2000
		if v, ok := modbusTCPConf["timeout"]; ok {
			switch vv := v.(type) {
			case int:
				timeout = vv
			case float64:
				timeout = int(vv)
			}
		}
		ip := "127.0.0.1"
		if v, ok := modbusTCPConf["ip"]; ok {
			ip = fmt.Sprintf("%v", v)
		}
		port := 502
		if v, ok := modbusTCPConf["port"]; ok {
			switch vv := v.(type) {
			case int:
				port = vv
			case float64:
				port = int(vv)
			}
		}
		// 构造设备配置
		devConf := types.DeviceConfig{
			ID:       "modbus1",
			Protocol: "modbus_tcp",
			Interval: interval,
			Config: map[string]interface{}{
				"ip":      ip,
				"port":    port,
				"timeout": timeout,
				// 其他参数可按需补充
			},
		}
		client, err := protocols.Create(devConf.Protocol)
		if err != nil {
			panic(err)
		}
		err = client.Init(devConf.Config)
		if err != nil {
			panic(err)
		}
		// 使用转换后的 []string 读取数据，增加重试机制
		var values []protocols.PointValue
		pointAddrs := make([]string, 0, len(modbusPoints))
		for _, p := range modbusPoints {
			pointAddrs = append(pointAddrs, p.Address)
		}
		maxRetry := 3
		for retry := 1; retry <= maxRetry; retry++ {
			values, err = client.(*modbus.ModbusTCP).ReadBatch(devConf.ID, pointAddrs)
			if err == nil {
				break
			}
			fmt.Printf("[WARN] Modbus 读取失败（第%d次）：%v\n", retry, err)
			time.Sleep(2 * time.Second)
		}
		if err != nil {
			panic(err)
		}
		// 构造点位数据
		pointValues := make(map[string]interface{})
		for _, v := range values {
			pointValues[v.PointID] = v.Value
			fmt.Printf("[modbus1] %s = %v\n", v.PointID, v.Value)
		}
		// 编码数据并上报
		payload := uplink.EncodeDataReport(
			devConf.ID,
			pointValues,
			nil, // 无报警信息
			nil, // 无指标数据
		)
		// 通过所有已配置的上行通道发送数据
		err = uplinkMgr.SendToAll(payload)
		if err != nil {
			fmt.Printf("[Error] 数据上报失败: %v\n", err)
		} else {
			fmt.Println("[Success] 数据上报成功")
		}
	}

	// DEMO: 修改设备采集频率
	// 假设有一个runner列表，找到modbus1并修改其采集频率为2秒
	for _, r := range runners {
		if r.ID == "modbus1" {
			fmt.Println("[Demo] 修改 modbus1 采集频率为2秒")
			r.SetInterval(2)
		}
	}

	// 阻塞运行
	select {}
}
