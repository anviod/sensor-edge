package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/signal"
	"reflect"
	"sensor-edge/config"
	"sensor-edge/edgecompute"
	"sensor-edge/protocols"
	"sensor-edge/protocols/modbus"
	"sensor-edge/schema"
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
func readWithRetry(client protocols.Protocol, devID string, function string, addrs []string, maxRetry int) ([]protocols.PointValue, error) {
	var values []protocols.PointValue
	var err error
	for retry := 1; retry <= maxRetry; retry++ {
		values, err = client.ReadBatch(devID, function, addrs)
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
	pointSetsV2, _ := config.LoadPointMappingsV2("configs/points.yaml")

	// 5. 加载新版分组式边缘规则
	devRules, _ := config.LoadDeviceEdgeRules("configs/edge_rules.yaml")
	aggRules := config.ExtractAggregateRules(devRules)
	alarmRules := config.ExtractAlarmRules(devRules)
	linkageRules := config.ExtractLinkageRules(devRules)
	re := edgecompute.NewRuleEngine(aggRules, alarmRules, linkageRules)

	// 6. 加载上行通道配置
	uplinkCfgs, _ := config.LoadUplinkConfigs("configs/uplinks.yaml")
	uplinkMgr := uplink.NewUplinkManagerFromConfig(uplinkCfgs)

	fmt.Println("[System] Device collection, edge rule engine & uplink started...")

	// 7. 采集主流程：每个设备独立采集周期并发采集
	for _, set := range pointSetsV2 {
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
		for _, funcGroup := range set.Functions {
			go func(set types.DevicePointSetV2, devConf types.DeviceConfigWithMeta, client protocols.Protocol, interval time.Duration, funcGroup types.FunctionPointGroup) {
				ticker := time.NewTicker(interval)
				defer ticker.Stop()
				for {
					pointAddrs := extractPointAddresses(funcGroup.Points)
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
					// 采集时传递功能码 funcGroup.Function 给驱动
					values, err := readWithRetry(client, set.DeviceID, funcGroup.Function, pointAddrs, 3)
					if err != nil {
						fmt.Printf("[ERROR] 设备 %s 采集失败: %v\n", set.DeviceID, err)
						continue
					}
					pointValues := make(map[string]interface{})
					for _, p := range funcGroup.Points {
						pointValues[p.Name] = nil
					}
					for _, v := range values {
						for _, p := range funcGroup.Points {
							if v.PointID == p.Address || v.PointID == p.Name {
								val := v.Value
								// 自动兼容驱动返回 [uint16,uint16] 的 float/double 点位
								if arr, ok := val.([]uint16); ok && len(arr) == 2 && strings.HasPrefix(strings.ToUpper(p.Format), "FLOAT") {
									b := make([]byte, 4)
									binary.BigEndian.PutUint16(b[0:2], arr[0])
									binary.BigEndian.PutUint16(b[2:4], arr[1])
									val = b
								}
								if arr, ok := val.([]uint16); ok && len(arr) == 4 && strings.HasPrefix(strings.ToUpper(p.Format), "DOUBLE") {
									b := make([]byte, 8)
									binary.BigEndian.PutUint16(b[0:2], arr[0])
									binary.BigEndian.PutUint16(b[2:4], arr[1])
									binary.BigEndian.PutUint16(b[4:6], arr[2])
									binary.BigEndian.PutUint16(b[6:8], arr[3])
									val = b
								}
								// 兼容驱动直接返回 uint32 且 format 为 float 的情况
								if u32, ok := val.(uint32); ok && strings.HasPrefix(strings.ToUpper(p.Format), "FLOAT") {
									b := make([]byte, 4)
									binary.BigEndian.PutUint32(b, u32)
									val = b
								}
								// 新增：兼容 format 为 float 且只收到单个 uint16 的情况，自动补齐为4字节 float32
								if strings.HasPrefix(strings.ToUpper(p.Format), "FLOAT") {
									switch vv := val.(type) {
									case uint16:
										b := make([]byte, 4)
										binary.BigEndian.PutUint16(b[0:2], vv)
										val = b
									case []uint16:
										if len(vv) == 1 {
											b := make([]byte, 4)
											binary.BigEndian.PutUint16(b[0:2], vv[0])
											val = b
										}
									}
								}
								// 使用 Format 字段进行格式化
								if p.Format != "" {
									val2, err := utils.ParseAndCastFormat(p.Format, val)
									if err == nil {
										val = val2
									}
								}
								// 使用 Transform 表达式
								if p.Transform != "" {
									val, err := parseTransform(p.Transform, val)
									if err != nil {

										fmt.Printf("[WARN] 设备 %s 点位 %s 转换失败: %v\n", set.DeviceID, p.Name, err)
									} else {
										// 如果转换结果是字符串，尝试转换为 float
										if strVal, ok := val.(string); ok {
											if f, err := strconv.ParseFloat(strVal, 64); err == nil {
												val = f
											}
										}
									}

								}
								// 根据 Type 进行类型转换
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
								if strings.ToLower(p.Type) == "int" {
									switch vv := val.(type) {
									case float32, float64:
										val = int(math.Round(reflect.ValueOf(vv).Convert(reflect.TypeOf(float64(0))).Float()))
									case string:
										if f, err := strconv.ParseFloat(vv, 64); err == nil {
											val = int(math.Round(f))
										}
									}
								}
								pointValues[p.Name] = val
								// 日志输出也用最终val，保证与上报一致
								fmt.Printf("[%s] %s = %v\n", set.DeviceID, v.PointID, val)
								break
							}
						}
					}
					// 先推进边缘规则引擎，聚合窗口和报警状态
					if re != nil {
						re.ApplyRules(set.DeviceID, pointValues)
					}
					// 边缘规则引擎处理，收集报警和聚合结果
					alarms := []schema.AlarmInfo{}
					metrics := map[string]interface{}{}
					if re != nil {
						// 1. 聚合规则（如 avg）
						for _, rule := range re.AggRules {
							if rule.DeviceID == set.DeviceID {
								key := set.DeviceID + "." + rule.Point
								buf, ok := re.Buffers[key]
								if ok && rule.Method == "avg" {
									metrics[rule.Point+"_avg"] = buf.Avg()
								}
							}
						}
						// 2. 直接读取规则引擎的 LastAlarms
						alarms = append(alarms, re.LastAlarms...)
					}
					// 上报数据+报警+聚合
					payload := uplink.EncodeDataReport(set.DeviceID, pointValues, alarms, metrics)
					err = uplinkMgr.SendToAll(payload)
					if err != nil {
						fmt.Printf("[Error] 设备 %s 数据上报失败: %v\n", set.DeviceID, err)
					} else {
						fmt.Printf("[Success] 设备 %s 数据上报成功\n", set.DeviceID)
					}
					<-ticker.C
				}
			}(set, devConf, client, interval, funcGroup)
		}
	}

	// 8. 支持热加载规则（SIGHUP）
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	go func() {
		for range c {
			devRules, _ := config.LoadDeviceEdgeRules("configs/edge_rules.yaml")
			re.AggRules = config.ExtractAggregateRules(devRules)
			re.AlarmRules = config.ExtractAlarmRules(devRules)
			re.LinkageRules = config.ExtractLinkageRules(devRules)
			fmt.Println("[System] Edge rules reloaded!")
		}
	}()

	// 阻塞运行
	select {}
}
