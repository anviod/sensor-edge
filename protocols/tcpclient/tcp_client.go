package tcpclient

import (
	"net"
	"sensor-edge/protocols"
	"time"
)

type TCPClient struct {
	conn    net.Conn
	ip      string
	port    string
	request []byte
}

func (t *TCPClient) Init(config map[string]interface{}) error {
	t.ip = config["ip"].(string)
	t.port = config["port"].(string)
	t.request = []byte(config["request_hex"].(string)) // 示例："010300000002C40B"

	var err error
	t.conn, err = net.DialTimeout("tcp", t.ip+":"+t.port, 3*time.Second)
	return err
}

func (t *TCPClient) Read(deviceID string) ([]protocols.PointValue, error) {
	t.conn.Write(t.request)

	buf := make([]byte, 1024)
	n, err := t.conn.Read(buf)
	if err != nil || n < 7 {
		return nil, err
	}

	val := int(buf[3])<<8 + int(buf[4])
	return []protocols.PointValue{
		{
			PointID:   "tcp_data",
			Value:     val,
			Quality:   "good",
			Timestamp: time.Now().Unix(),
		},
	}, nil
}

// ReadBatch 批量读取接口，返回指定数据点
func (t *TCPClient) ReadBatch(deviceID string, points []string) ([]protocols.PointValue, error) {
	if len(points) == 0 {
		return nil, nil
	}
	var values []protocols.PointValue
	for _, pt := range points {
		// 这里只做示例，实际应根据点位协议组装报文
		t.conn.Write(t.request)
		buf := make([]byte, 1024)
		n, err := t.conn.Read(buf)
		if err != nil || n < 7 {
			values = append(values, protocols.PointValue{PointID: pt, Quality: "bad", Value: nil, Timestamp: time.Now().Unix()})
			continue
		}
		val := int(buf[3])<<8 + int(buf[4])
		values = append(values, protocols.PointValue{PointID: pt, Quality: "good", Value: val, Timestamp: time.Now().Unix()})
	}
	return values, nil
}

func (t *TCPClient) Close() error {
	return t.conn.Close()
}

func (t *TCPClient) Write(point string, value interface{}) error {

	return nil // TODO: 实现写操作
}

func NewTCPClient() protocols.Protocol {
	return &TCPClient{}
}

func init() {
	protocols.Register("tcpclient", NewTCPClient)
}
