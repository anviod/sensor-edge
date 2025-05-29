package main

import (
	"fmt"
	"os"
	"os/signal"
	"reflect"
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

// 客户端池Key
type ClientKey struct {
	Protocol string
	IP       string
	Port     int
}

var clientCache = make(map[ClientKey]protocols.Protocol)

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

// 辅助函数：注入设备元数据到Config
func injectDeviceMeta(dev *types.DeviceConfigWithMeta, meta types.DeviceMeta) {
	val := reflect.ValueOf(meta)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("yaml")
		if tag == "" {
			continue
		}
		if dev.Config == nil {
			dev.Config = make(map[string]interface{})
		}
		if _, exists := dev.Config[tag]; !exists {
			dev.Config[tag] = val.Field(i).Interface()
		}
	}
}

// 辅助函数：提取点位地址
func extractPointAddresses(points []types.PointMapping) []string {
	addrs := make([]string, 0, len(points))
	for _, p := range points {
		addrs = append(addrs, p.Address)
	}
	return addrs
}

// 辅助函数：带重试的批量读取
func readWithRetry(client protocols.Protocol, devID string, addrs []string, maxRetry int) ([]protocols.PointValue, error) {
	var values []protocols.PointValue
	var err error
	for retry := 1; retry <= maxRetry; retry++ {
		values, err = client.ReadBatch(devID, addrs)
		if err == nil {
			break
		}
		fmt.Printf("[WARN] 设备 %s 读取失败（第%d次）：%v\n", devID, retry, err)
		time.Sleep(2 * time.Second)
	}
	return values, err
}

// 获取或创建协议客户端（池化）
func getOrCreateClient(protocol string, config map[string]interface{}) (protocols.Protocol, error) {
	ip, _ := config["ip"].(string)
	port := 0
	switch v := config["port"].(type) {
	case int:
		port = v
	case float64:
		port = int(v)
	}
	key := ClientKey{Protocol: protocol, IP: ip, Port: port}
	if client, exists := clientCache[key]; exists {
		return client, nil
	}
	client, err := protocols.Create(protocol)
	if err != nil {
		return nil, err
	}
	err = client.Init(config)
	if err != nil {
		return nil, err
	}
	clientCache[key] = client
	return client, nil
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
		protocol := devConf.Protocol
		if set.Protocol != "" {
			protocol = set.Protocol
		}
		protocolName := devConf.ProtocolName
		if set.ProtocolName != "" {
			protocolName = set.ProtocolName
		}
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
		injectDeviceMeta(&devConf, devConf.DeviceMeta)
		for k, v := range protoParams {
			if _, exists := devConf.Config[k]; !exists {
				devConf.Config[k] = v
			}
		}
		client, err := getOrCreateClient(protocol, devConf.Config)
		if err != nil {
			fmt.Printf("[ERROR] 设备 %s 协议初始化失败: %v\n", set.DeviceID, err)
			continue
		}
		// 动态设置slave_id
		if m, ok := client.(*modbus.ModbusTCP); ok {
			slaveId := byte(1)
			if v, ok := devConf.Config["slave_id"]; ok {
				switch vv := v.(type) {
				case int:
					slaveId = byte(vv)
				case float64:
					slaveId = byte(vv)
				}
			}
			m.SetSlave(slaveId)
		}
		pointAddrs := extractPointAddresses(set.Points)
		values, err := readWithRetry(client, set.DeviceID, pointAddrs, 3)
		if err != nil {
			fmt.Printf("[ERROR] 设备 %s 采集失败: %v\n", set.DeviceID, err)
			continue
		}
		pointValues := make(map[string]interface{})
		for _, v := range values {
			pointValues[v.PointID] = v.Value
			fmt.Printf("[%s] %s = %v\n", set.DeviceID, v.PointID, v.Value)
		}
		re.ApplyRules(set.DeviceID, pointValues)
		payload := uplink.EncodeDataReport(set.DeviceID, pointValues, nil, nil)
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
