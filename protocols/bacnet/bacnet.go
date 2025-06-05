package bacnet

import (
	"errors"
	"fmt"
	"reflect"
	"sensor-edge/protocols"
	"sync"
	"time"
)

// BacnetClient 是 BACnet 协议的客户端实现（模拟器/占位实现）
type BacnetClient struct {
	deviceID  string
	connected bool
	lock      sync.Mutex
	points    map[string]BacnetPoint // 支持点位物模型
}

type BacnetPoint struct {
	Name              string            // 点位名称
	Address           string            // 点位地址
	Type              string            // 点位数据类型
	Format            string            // 点位格式
	Unit              string            // 点位单位
	Value             interface{}       // 点位当前值
	Property          string            // 属性名，如 presentValue
	PropertyValueType PropertyValueType // 属性值类型，参考 type.go
	Writable          bool              // 是否可写
}

// 初始化链接信息
func (c *BacnetClient) Init(config map[string]interface{}) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.deviceID = fmt.Sprintf("%v", config["device_id"])
	c.connected = true
	if pts, ok := config["points"].([]interface{}); ok {
		c.points = make(map[string]BacnetPoint)
		for _, p := range pts {
			if m, ok := p.(map[string]interface{}); ok {
				pt := BacnetPoint{
					Name:     fmt.Sprintf("%v", m["name"]),
					Address:  fmt.Sprintf("%v", m["address"]),
					Type:     fmt.Sprintf("%v", m["type"]),
					Format:   fmt.Sprintf("%v", m["format"]),
					Unit:     fmt.Sprintf("%v", m["unit"]),
					Value:    m["init_value"],
					Property: fmt.Sprintf("%v", m["property"]),
					Writable: false,
				}
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
					// 可扩展更多类型
					default:
						pt.PropertyValueType = TypeNull
					}
				}
				c.points[pt.Name] = pt
			}
		}
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

func (c *BacnetClient) ReadBatch(deviceID string, function string, points []string) ([]protocols.PointValue, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if !c.connected {
		return nil, errors.New("bacnet: not connected")
	}
	result := make([]protocols.PointValue, 0, len(points))
	for _, name := range points {
		pt := c.points[name]
		val := pt.Value
		if val == nil {
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

func (c *BacnetClient) Write(point string, value interface{}) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if !c.connected {
		return errors.New("bacnet: not connected")
	}
	pt, ok := c.points[point]
	if !ok {
		return fmt.Errorf("bacnet: point %s not found", point)
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

func init() {
	protocols.Register("bacnet", func() protocols.Protocol {
		return &BacnetClient{}
	})
}
