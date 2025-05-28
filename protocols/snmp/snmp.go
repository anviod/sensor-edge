package snmp

import (
	"sensor-edge/protocols"
	"time"

	"github.com/gosnmp/gosnmp"
)

type SNMPClient struct {
	client *gosnmp.GoSNMP
	oids   []string
}

func (s *SNMPClient) Init(config map[string]interface{}) error {
	s.client = &gosnmp.GoSNMP{
		Target:    config["ip"].(string),
		Port:      uint16(config["port"].(float64)),
		Version:   gosnmp.Version2c,
		Community: config["community"].(string),
		Timeout:   2 * time.Second,
		Retries:   2,
	}

	s.oids = []string{config["oid"].(string)}
	return s.client.Connect()
}

func (s *SNMPClient) Read(deviceID string) ([]protocols.PointValue, error) {
	result, err := s.client.Get(s.oids)
	if err != nil {
		return nil, err
	}

	var val interface{}
	for _, variable := range result.Variables {
		switch variable.Type {
		case gosnmp.Integer:
			val = variable.Value.(int)
		case gosnmp.OctetString:
			val = string(variable.Value.([]byte))
		default:
			val = variable.Value
		}
	}

	return []protocols.PointValue{
		{
			PointID:   "snmp_metric",
			Value:     val,
			Quality:   "good",
			Timestamp: time.Now().Unix(),
		},
	}, nil
}

// ReadBatch 批量读取接口，返回指定数据点
func (s *SNMPClient) ReadBatch(deviceID string, points []string) ([]protocols.PointValue, error) {
	if len(points) == 0 {
		return nil, nil
	}
	var values []protocols.PointValue
	for _, pt := range points {
		// 这里只做示例，实际应根据点位OID组装
		result, err := s.client.Get([]string{pt})
		if err != nil || len(result.Variables) == 0 {
			values = append(values, protocols.PointValue{PointID: pt, Quality: "bad", Value: nil, Timestamp: time.Now().Unix()})
			continue
		}
		val := result.Variables[0].Value
		values = append(values, protocols.PointValue{PointID: pt, Quality: "good", Value: val, Timestamp: time.Now().Unix()})
	}
	return values, nil
}

func (s *SNMPClient) Close() error {
	return s.client.Conn.Close()
}

func (s *SNMPClient) Write(point string, value interface{}) error {
	return nil // SNMP 通常不支持写操作，或需要特定配置
	// 可根据需要实现 SNMP SET 操作
}

func NewSNMPClient() protocols.Protocol {
	return &SNMPClient{}
}

func init() {
	protocols.Register("snmp", NewSNMPClient)
}
