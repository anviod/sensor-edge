package main

import (
	"fmt"
	"math"
	"os"
	"os/signal"
	"reflect"
	"regexp"
	"sensor-edge/config"
	"sensor-edge/edgecompute"
	"sensor-edge/protocols"
	"sensor-edge/protocols/modbus"
	"sensor-edge/types"
	"sensor-edge/uplink"
	"sensor-edge/utils"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Knetic/govaluate"

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

// parseTransform 支持复杂表达式和内置函数
func parseTransform(expr string, value interface{}) (interface{}, error) {
	parameters := make(map[string]interface{})
	var v float64
	switch vv := value.(type) {
	case int, int32, int64, float32, float64, uint16, uint32, uint64:
		v = reflect.ValueOf(vv).Convert(reflect.TypeOf(float64(0))).Float()
	case string:
		var err error
		v, err = strconv.ParseFloat(vv, 64)
		if err != nil {
			return value, err
		}
	default:
		return value, fmt.Errorf("unsupported value type: %T", value)
	}
	parameters["value"] = v
	functions := map[string]govaluate.ExpressionFunction{
		"abs": func(args ...interface{}) (interface{}, error) {
			return math.Abs(args[0].(float64)), nil
		},
		"sqrt": func(args ...interface{}) (interface{}, error) {
			return math.Sqrt(args[0].(float64)), nil
		},
		"log": func(args ...interface{}) (interface{}, error) {
			return math.Log(args[0].(float64)), nil
		},
		"min": func(args ...interface{}) (interface{}, error) {
			return math.Min(args[0].(float64), args[1].(float64)), nil
		},
		"max": func(args ...interface{}) (interface{}, error) {
			return math.Max(args[0].(float64), args[1].(float64)), nil
		},
	}
	expression, err := govaluate.NewEvaluableExpressionWithFunctions(expr, functions)
	if err != nil {
		return value, err
	}
	result, err := expression.Evaluate(parameters)
	if err != nil {
		return value, err
	}
	return result, nil
}

// parseTransformSimple 支持如 "value * 0.1" 的简单表达式
func parseTransformSimple(expr string, value interface{}) (interface{}, error) {
	re := regexp.MustCompile(`(?i)^value\s*([\*\/+\-])\s*([0-9.]+)$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(expr))
	if len(matches) != 3 {
		return value, nil // 不支持的表达式，原样返回
	}
	op := matches[1]
	numStr := matches[2]
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return value, nil
	}
	var v float64
	switch vv := value.(type) {
	case int:
		v = float64(vv)
	case int32:
		v = float64(vv)
	case int64:
		v = float64(vv)
	case float32:
		v = float64(vv)
	case float64:
		v = vv
	case uint16:
		v = float64(vv)
	case uint32:
		v = float64(vv)
	case uint64:
		v = float64(vv)
	case string:
		v, err = strconv.ParseFloat(vv, 64)
		if err != nil {
			return value, nil
		}
	default:
		return value, nil
	}
	switch op {
	case "*":
		return v * num, nil
	case "/":
		if num == 0 {
			return value, nil
		}
		return v / num, nil
	case "+":
		return v + num, nil
	case "-":
		return v - num, nil
	}
	return value, nil
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

	// 7. 采集主流程：每个设备独立采集周期并发采集
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
		// 解析采集周期
		interval := 5 * time.Second
		if v, ok := devConf.Config["interval"]; ok {
			switch vv := v.(type) {
			case int:
				interval = time.Duration(vv) * time.Second
			case float64:
				interval = time.Duration(int(vv)) * time.Second
			case string:
				d, err := time.ParseDuration(vv)
				if err == nil {
					interval = d
				}
			}
		}
		go func(set types.DevicePointSet, devConf types.DeviceConfigWithMeta, client protocols.Protocol, interval time.Duration) {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				pointAddrs := extractPointAddresses(set.Points)
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
				values, err := readWithRetry(client, set.DeviceID, pointAddrs, 3)
				if err != nil {
					fmt.Printf("[ERROR] 设备 %s 采集失败: %v\n", set.DeviceID, err)
				} else {
					pointValues := make(map[string]interface{})
					// 先用name初始化，保证所有点位都有
					for _, p := range set.Points {
						pointValues[p.Name] = nil
					}
					// 用name为key填充值
					for _, v := range values {
						for _, p := range set.Points {
							if v.PointID == p.Address || v.PointID == p.Name {
								val := v.Value
								// 1. format 字段优先解析（如为[]byte）
								if p.Format != "" {
									if raw, ok := val.([]byte); ok {
										val2, err := utils.ParseFormat(p.Format, raw)
										if err == nil {
											val = val2
										}
									}
								}
								// 2. transform 表达式
								if p.Transform != "" {
									val2, err := parseTransform(p.Transform, val)
									if err != nil {
										val2, err = parseTransformSimple(p.Transform, val)
									}
									if err == nil {
										val = val2
									}
								}
								// 3. float 类型自动保留2位小数
								if strings.ToLower(p.Type) == "float" {
									switch vv := val.(type) {
									case float32, float64:
										val = math.Round(reflect.ValueOf(vv).Convert(reflect.TypeOf(float64(0))).Float()*100) / 100
									case int, int32, int64, uint16, uint32, uint64:
										valf := reflect.ValueOf(vv).Convert(reflect.TypeOf(float64(0))).Float()
										val = math.Round(valf*100) / 100
									case string:
										if f, err := strconv.ParseFloat(vv, 64); err == nil {
											val = math.Round(f*100) / 100
										}
									}
								}
								pointValues[p.Name] = val
								break
							}
						}
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
				<-ticker.C
			}
		}(set, devConf, client, interval)
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
