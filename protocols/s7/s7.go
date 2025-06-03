package s7

import (
	"sensor-edge/protocols"
	"time"

	gos7 "github.com/robinson/gos7"
)

type S7Client struct {
	client  gos7.Client
	handler *gos7.TCPClientHandler
}

func (s *S7Client) Init(config map[string]interface{}) error {
	ip := config["ip"].(string)
	// 兼容int/float64类型
	rackRaw := config["rack"]
	slotRaw := config["slot"]
	var rack, slot int
	switch v := rackRaw.(type) {
	case float64:
		rack = int(v)
	case int:
		rack = v
	}
	switch v := slotRaw.(type) {
	case float64:
		slot = int(v)
	case int:
		slot = v
	}

	s.handler = gos7.NewTCPClientHandler(ip, rack, slot)
	s.handler.Timeout = 3 * time.Second

	err := s.handler.Connect()
	if err != nil {
		return err
	}

	s.client = gos7.NewClient(s.handler)
	return nil
}

func (s *S7Client) Read(deviceID string) ([]protocols.PointValue, error) {
	buffer := make([]byte, 2)
	err := s.client.AGReadDB(1, 0, 2, buffer) // 读取 DB1.DBX0.0 (示例)
	if err != nil {
		return nil, err
	}

	running := buffer[0]&1 == 1

	return []protocols.PointValue{
		{
			PointID:   "fan_status",
			Value:     running,
			Quality:   "good",
			Timestamp: time.Now().Unix(),
		},
	}, nil
}
func (s *S7Client) Write(point string, value interface{}) error {
	return nil
}

func (s *S7Client) Close() error {
	return s.handler.Close()
}

func NewS7Client() protocols.Protocol {
	return &S7Client{}
}

func init() {
	protocols.Register("s7", NewS7Client)
}

// ReadBatch 批量读取接口，返回指定数据点
func (c *S7Client) ReadBatch(deviceID string, function string, points []string) ([]protocols.PointValue, error) {
	// function 参数暂未用到，保留兼容
	if len(points) == 0 {
		return nil, nil
	}
	var values []protocols.PointValue
	for _, pt := range points {
		buffer := make([]byte, 2)
		err := c.client.AGReadDB(1, 0, 2, buffer)
		if err != nil {
			values = append(values, protocols.PointValue{PointID: pt, Quality: "bad", Value: nil, Timestamp: time.Now().Unix()})
			continue
		}
		val := buffer[0]&1 == 1
		values = append(values, protocols.PointValue{PointID: pt, Quality: "good", Value: val, Timestamp: time.Now().Unix()})
	}
	return values, nil
}

func(c *S7Client) Reconnect() error {
	if c.handler == nil {
		return nil // 未初始化时不需要重连
	}
	err := c.handler.Connect()
	if err != nil {
		return err
	}
	c.client = gos7.NewClient(c.handler)
	return nil
}