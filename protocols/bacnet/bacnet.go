package bacnet

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"sensor-edge/protocols"
	"sync"
	"time"

	"github.com/Knetic/govaluate"
)

// BacnetClient 是 BACnet 协议的客户端实现（模拟器/占位实现）
type BacnetClient struct {
	deviceID      string
	connected     bool
	lock          sync.Mutex
	points        map[string]BacnetPoint // 支持点位物模型
	idToName      map[string]string      // id -> name
	addressToName map[string]string      // address -> name
	// 扩展能力
	covSubs       map[string]bool // COV订阅状态
	discovered    bool            // 设备发现标志
	priorityWrite map[string]int  // 点位优先级写入
	retryCount    map[string]int  // 点位错误重试计数
	offline       bool            // 设备离线标志
}

type BacnetPoint struct {
	ID                string            // 点位唯一标识（如 2228316.ai0）
	ObjectType        string            // BACnet对象类型（如 analogInput）
	Instance          int               // BACnet实例号
	Description       string            // 点位描述
	Access            string            // 读写权限（read/write）
	Name              string            // 点位名称
	Address           string            // 点位地址
	Type              string            // 点位数据类型
	Format            string            // 点位格式
	Unit              string            // 点位单位
	Value             interface{}       // 点位当前值
	Property          string            // 属性名，如 presentValue
	PropertyValueType PropertyValueType // 属性值类型，参考 type.go
	Writable          bool              // 是否可写
	Transform         string            // 变换表达式
}

// 初始化链接信息
func (c *BacnetClient) Init(config map[string]interface{}) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	// 优先从 object_device 获取设备ID
	if v, ok := config["object_device"]; ok {
		c.deviceID = fmt.Sprintf("%v", v)
	} else {
		c.deviceID = fmt.Sprintf("%v", config["device_id"])
	}
	c.connected = true
	c.points = make(map[string]BacnetPoint)
	c.idToName = make(map[string]string)
	c.addressToName = make(map[string]string)
	if pts, ok := config["points"].([]interface{}); ok {
		for _, p := range pts {
			if m, ok := p.(map[string]interface{}); ok {
				pt := BacnetPoint{}
				if v, ok := m["id"]; ok {
					pt.ID = fmt.Sprintf("%v", v)
				} else if c.deviceID != "" && m["name"] != nil {
					pt.ID = fmt.Sprintf("%s.%s", c.deviceID, m["name"])
				}
				if v, ok := m["object_type"]; ok {
					pt.ObjectType = fmt.Sprintf("%v", v)
				}
				if v, ok := m["instance"]; ok {
					if i, ok2 := v.(int); ok2 {
						pt.Instance = i
					} else if f, ok2 := v.(float64); ok2 {
						pt.Instance = int(f)
					} else {
						pt.Instance = 0
					}
				}
				if v, ok := m["description"]; ok {
					pt.Description = fmt.Sprintf("%v", v)
				}
				if v, ok := m["access"]; ok {
					pt.Access = fmt.Sprintf("%v", v)
				}
				if v, ok := m["name"]; ok {
					pt.Name = fmt.Sprintf("%v", v)
				}
				if v, ok := m["address"]; ok {
					pt.Address = fmt.Sprintf("%v", v)
				}
				if v, ok := m["type"]; ok {
					pt.Type = fmt.Sprintf("%v", v)
				}
				if v, ok := m["format"]; ok {
					pt.Format = fmt.Sprintf("%v", v)
				}
				if v, ok := m["unit"]; ok {
					pt.Unit = fmt.Sprintf("%v", v)
				}
				if v, ok := m["init_value"]; ok {
					pt.Value = v
				}
				if v, ok := m["property"]; ok {
					pt.Property = fmt.Sprintf("%v", v)
				}
				if v, ok := m["transform"]; ok {
					pt.Transform = fmt.Sprintf("%v", v)
				}
				pt.Writable = false
				if w, ok := m["writable"].(bool); ok {
					pt.Writable = w
				}
				if pvt, ok := m["property_value_type"].(string); ok {
					switch pvt {
					case "REAL":
						pt.PropertyValueType = TypeReal
					case "INTEGER":
						pt.PropertyValueType = TypeSignedInt
					case "BOOLEAN":
						pt.PropertyValueType = TypeBoolean
					case "ENUMERATED":
						pt.PropertyValueType = TypeEnumerated
					case "CHARACTERSTRING":
						pt.PropertyValueType = TypeCharacterString
					case "OCTETSTRING":
						pt.PropertyValueType = TypeOctetString
					case "BITSTRING":
						pt.PropertyValueType = TypeBitString
					case "OBJECTID":
						pt.PropertyValueType = TypeObjectID
					default:
						pt.PropertyValueType = TypeNull
					}
				}
				// 自动从 address 解析 object_type/instance
				if pt.ObjectType == "" || pt.Instance == 0 {
					if segs := parseAddressFields(pt.Address); segs != nil {
						if pt.ObjectType == "" {
							if s, ok := segs[0].(string); ok {
								pt.ObjectType = s
							}
						}
						if pt.Instance == 0 {
							if i, ok := segs[1].(int); ok {
								pt.Instance = i
							} else if f, ok := segs[1].(float64); ok {
								pt.Instance = int(f)
							}
						}
					}
				}
				c.points[pt.Name] = pt
				if pt.ID != "" {
					c.idToName[pt.ID] = pt.Name
				}
				if pt.Address != "" {
					c.addressToName[pt.Address] = pt.Name
				}
			}
		}
	}
	return nil
}

// parseAddressFields 解析 address 字段，返回 [object_type, instance]，如 analogInput:0
func parseAddressFields(addr string) []interface{} {
	var res []interface{}
	if addr == "" {
		return nil
	}
	var objType string
	var inst int
	_, err := fmt.Sscanf(addr, "%[^:]:%d", &objType, &inst)
	if err == nil {
		res = append(res, objType)
		res = append(res, inst)
		return res
	}
	return nil
}

func (c *BacnetClient) Read(deviceID string) ([]protocols.PointValue, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if !c.connected {
		return nil, errors.New("bacnet: not connected")
	}
	var result []protocols.PointValue
	for name, pt := range c.points {
		val := pt.Value
		if val == nil {
			// 默认值
			if pt.Type == "bool" {
				val = false
			} else {
				val = 42.0
			}
		}
		result = append(result, protocols.PointValue{
			PointID:   name,
			Value:     val,
			Quality:   "good",
			Timestamp: time.Now().Unix(),
		})
	}
	return result, nil
}

// 采集时生成 BACnetReadRequest（此处仅打印，便于后续对接真实协议栈）
func (c *BacnetClient) ReadBatch(deviceID string, function string, points []string) ([]protocols.PointValue, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if !c.connected {
		return nil, errors.New("bacnet: not connected")
	}
	result := make([]protocols.PointValue, 0, len(points))
	for _, key := range points {
		pt, ok := c.points[key]
		if !ok {
			if name2, ok2 := c.addressToName[key]; ok2 {
				pt = c.points[name2]
			} else if name3, ok3 := c.idToName[key]; ok3 {
				pt = c.points[name3]
			} else {
				result = append(result, protocols.PointValue{
					PointID:   key,
					Value:     nil,
					Quality:   "bad",
					Timestamp: time.Now().Unix(),
				})
				continue
			}
		}
		// 采集请求结构体
		req := BACnetReadRequest{
			DeviceID:   c.deviceID,
			ObjectType: pt.ObjectType,
			Instance:   pt.Instance,
			Property:   pt.Property,
		}
		fmt.Printf("[BACnet] ReadRequest: %+v\n", req)
		val := pt.Value
		if val == nil {
			if pt.Type == "bool" {
				val = false
			} else {
				val = 42.0
			}
		}
		if pt.Transform != "" {
			if v2, err := parseTransform(pt.Transform, val); err == nil {
				val = v2
			}
		}
		result = append(result, protocols.PointValue{
			PointID:   pt.Name,
			Value:     val,
			Quality:   "good",
			Timestamp: time.Now().Unix(),
		})
	}
	return result, nil
}

func (c *BacnetClient) Write(point string, value interface{}) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if !c.connected {
		return errors.New("bacnet: not connected")
	}
	pt, ok := c.points[point]
	if !ok {
		if name2, ok2 := c.addressToName[point]; ok2 {
			pt = c.points[name2]
			point = name2
		} else if name3, ok3 := c.idToName[point]; ok3 {
			pt = c.points[name3]
			point = name3
		} else {
			return fmt.Errorf("bacnet: point %s not found", point)
		}
	}
	if !pt.Writable {
		return fmt.Errorf("bacnet: point %s is not writable", point)
	}
	typeCheck := func(expect, actual reflect.Kind) error {
		if actual != expect {
			return fmt.Errorf("bacnet: point %s expect %s value, got %s", point, expect, actual)
		}
		return nil
	}
	switch pt.PropertyValueType {
	case TypeReal, TypeDouble:
		if err := typeCheck(reflect.Float64, reflect.TypeOf(value).Kind()); err != nil {
			return err
		}
	case TypeSignedInt, TypeEnumerated:
		if err := typeCheck(reflect.Int, reflect.TypeOf(value).Kind()); err != nil {
			return err
		}
	case TypeUnsignedInt:
		if err := typeCheck(reflect.Uint64, reflect.TypeOf(value).Kind()); err != nil {
			return err
		}
	case TypeBoolean:
		if err := typeCheck(reflect.Bool, reflect.TypeOf(value).Kind()); err != nil {
			return err
		}
	case TypeCharacterString, TypeDate, TypeTime:
		if err := typeCheck(reflect.String, reflect.TypeOf(value).Kind()); err != nil {
			return err
		}
	case TypeOctetString:
		if reflect.TypeOf(value).Kind() != reflect.Slice || reflect.TypeOf(value).Elem().Kind() != reflect.Uint8 {
			return fmt.Errorf("bacnet: point %s expect []byte value", point)
		}
	case TypeBitString:
		if reflect.TypeOf(value).Kind() != reflect.Slice || reflect.TypeOf(value).Elem().Kind() != reflect.Bool {
			return fmt.Errorf("bacnet: point %s expect []bool value", point)
		}
	case TypeObjectID:
		if reflect.TypeOf(value) != reflect.TypeOf(ObjectID{}) {
			return fmt.Errorf("bacnet: point %s expect ObjectID value", point)
		}
	}
	pt.Value = value
	c.points[point] = pt
	return nil
}

func (c *BacnetClient) Close() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.connected = false
	return nil
}

func (c *BacnetClient) Reconnect() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	time.Sleep(100 * time.Millisecond)
	c.connected = true
	return nil
}

// BACnet 发现到的设备信息结构体
type BACnetDeviceInfo struct {
	DeviceID string
	Address  string
	Vendor   string
	Model    string
}

// 设备发现与 I-Am 解析（模拟/占位实现）
func (c *BacnetClient) DiscoverDevices() ([]BACnetDeviceInfo, error) {
	fmt.Println("[BACnet] DiscoverDevices: Who-Is → I-Am (模拟)")
	// 模拟返回2台设备
	devices := []BACnetDeviceInfo{
		{DeviceID: "2228316", Address: "192.168.1.10", Vendor: "SimVendor", Model: "SimModelA"},
		{DeviceID: "2228317", Address: "192.168.1.11", Vendor: "SimVendor", Model: "SimModelB"},
	}
	c.discovered = true
	return devices, nil
}

// 真实网络发现 Who-Is/I-Am（占位实现）
func (c *BacnetClient) DiscoverDevicesReal(timeout time.Duration) ([]BACnetDeviceInfo, error) {
	var devices []BACnetDeviceInfo
	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	whoIs := []byte{0x81, 0x0b, 0x00, 0x0c, 0x01, 0x20, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00}
	broadcastAddr, _ := net.ResolveUDPAddr("udp4", "255.255.255.255:47808")
	conn.WriteTo(whoIs, broadcastAddr)
	conn.SetDeadline(time.Now().Add(timeout))
	buf := make([]byte, 1024)
	for {
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			break
		}
		// 简单判断I-Am响应（实际应解析BACnet NPDU/APDU）
		if n > 10 && buf[1] == 0x0a {
			devices = append(devices, BACnetDeviceInfo{
				DeviceID: "解析DeviceID", // TODO: 解析真实DeviceID
				Address:  addr.String(),
				Vendor:   "Unknown",
				Model:    "Unknown",
			})
		}
	}
	return devices, nil
}

// 导出点位物模型
func (c *BacnetClient) GetPointModel() map[string]BacnetPoint {
	c.lock.Lock()
	defer c.lock.Unlock()
	model := make(map[string]BacnetPoint)
	for k, v := range c.points {
		model[k] = v
	}
	return model
}

// 支持简单表达式的 transform 解析
func parseTransform(expr string, value interface{}) (interface{}, error) {
	if expr == "" {
		return value, nil
	}
	parameters := make(map[string]interface{})
	parameters["value"] = value
	expression, err := govaluate.NewEvaluableExpression(expr)
	if err != nil {
		return value, err
	}
	return expression.Evaluate(parameters)
}

func init() {
	protocols.Register("bacnet", func() protocols.Protocol {
		return &BacnetClient{}
	})
}

// BACnetReadRequest 结构体（模拟/占位）
type BACnetReadRequest struct {
	DeviceID   string
	ObjectType string
	Instance   int
	Property   string
}
