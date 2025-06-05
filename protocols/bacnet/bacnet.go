package bacnet

import (
	"errors"
	"fmt"
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
	Name    string
	Address string
	Type    string
	Format  string
	Unit    string
	Value   interface{} // 新增，存储当前点位值
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
					Name:    fmt.Sprintf("%v", m["name"]),
					Address: fmt.Sprintf("%v", m["address"]),
					Type:    fmt.Sprintf("%v", m["type"]),
					Format:  fmt.Sprintf("%v", m["format"]),
					Unit:    fmt.Sprintf("%v", m["unit"]),
					Value:   m["init_value"], // 支持初始值
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
	// 权限判断：禁止 writable: false 的点位写入
	writable := true
	if m, ok := c.points[point]; ok {
		if m2, ok := any(m).(map[string]interface{}); ok {
			if w, ok := m2["writable"].(bool); ok {
				writable = w
			}
		}
	}
	// 兼容结构体字段
	if !writable {
		return fmt.Errorf("bacnet: point %s is not writable", point)
	}
	// 类型校验
	switch pt.Type {
	case "float":
		_, ok := value.(float64)
		if !ok {
			return fmt.Errorf("bacnet: point %s expect float64 value", point)
		}
	case "int":
		_, ok := value.(int)
		if !ok {
			_, ok := value.(float64)
			if !ok {
				return fmt.Errorf("bacnet: point %s expect int value", point)
			}
		}
	case "bool":
		_, ok := value.(bool)
		if !ok {
			return fmt.Errorf("bacnet: point %s expect bool value", point)
		}
	}
	// 真实写入：更新点位值
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
