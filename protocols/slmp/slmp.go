package slmp

import (
	"net"
	"sensor-edge/protocols"
	"time"
)

type SLMPClient struct {
	conn net.Conn
	ip   string
	port string
}

func (s *SLMPClient) Init(config map[string]interface{}) error {
	s.ip = config["ip"].(string)
	s.port = config["port"].(string)

	var err error
	s.conn, err = net.DialTimeout("tcp", s.ip+":"+s.port, 3*time.Second)
	return err
}

func (s *SLMPClient) Read(deviceID string) ([]protocols.PointValue, error) {
	// 示例：读 D100（需根据三菱 SLMP 指令组装报文）
	readCommand := []byte{
		0x50, 0x00, 0x00, 0xFF, 0xFF, 0x03, 0x00, // 子头
		0x0C, 0x00, 0x10, 0x00, // 请求长度/监视定时器
		0x01, 0x04, 0x00, 0x00, // 读取命令（批量读取字）
		0x00, 0x64, 0x00, 0x00, 0xA8, 0x00, // 读取 D100 开始 1 点
	}

	s.conn.Write(readCommand)
	buf := make([]byte, 1024)
	n, err := s.conn.Read(buf)
	if err != nil || n < 14 {
		return nil, err
	}

	val := int(buf[14]) + int(buf[15])<<8
	return []protocols.PointValue{
		{
			PointID:   "d100",
			Value:     val,
			Quality:   "good",
			Timestamp: time.Now().Unix(),
		},
	}, nil
}

// ReadBatch 批量读取接口，返回指定数据点
func (s *SLMPClient) ReadBatch(deviceID string, function string, points []string) ([]protocols.PointValue, error) {
	// function 参数暂未用到，保留兼容
	if len(points) == 0 {
		return nil, nil
	}
	var values []protocols.PointValue
	for _, pt := range points {
		readCommand := []byte{
			0x50, 0x00, 0x00, 0xFF, 0xFF, 0x03, 0x00,
			0x0C, 0x00, 0x10, 0x00,
			0x01, 0x04, 0x00, 0x00,
			0x00, 0x64, 0x00, 0x00, 0xA8, 0x00,
		}
		s.conn.Write(readCommand)
		buf := make([]byte, 1024)
		n, err := s.conn.Read(buf)
		if err != nil || n < 14 {
			values = append(values, protocols.PointValue{PointID: pt, Quality: "bad", Value: nil, Timestamp: time.Now().Unix()})
			continue
		}
		val := int(buf[14]) + int(buf[15])<<8
		values = append(values, protocols.PointValue{PointID: pt, Quality: "good", Value: val, Timestamp: time.Now().Unix()})
	}
	return values, nil
}

func (s *SLMPClient) Write(point string, value interface{}) error {
	return nil // TODO: 实现写操作
}
func (s *SLMPClient) Close() error {
	return s.conn.Close()
}

func NewSLMPClient() protocols.Protocol {
	return &SLMPClient{}
}

func init() {
	protocols.Register("slmp", NewSLMPClient)
}


func (s *SLMPClient) Reconnect() error {
	if s.conn != nil {
		s.conn.Close() // 先关闭旧连接
	}
	var err error
	s.conn, err = net.DialTimeout("tcp", s.ip+":"+s.port, 3*time.Second)
	return err
	 
}