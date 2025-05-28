package sparkplugb

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// SparkplugBConfig 实现 uplink.Uplink 接口
// 仅演示参数结构和接口，生产建议完善缓存、重连、证书等

type SparkplugBConfig struct {
	ClientID      string
	GroupID       string
	NodeID        string
	GroupPath     bool
	OfflineCache  bool
	CacheMemMB    int
	CacheDiskMB   int
	CacheInterval int
	Host          string
	Port          int
	Username      string
	Password      string
	SSL           bool
	CAFile        string
	CertFile      string
	KeyFile       string
	KeyPassword   string
}

type SparkplugBUplink struct {
	Config      SparkplugBConfig
	Client      mqtt.Client
	NameV       string
	Queue       *OutboundQueue
	TLSReloader *TLSReloader
}

func NewSparkplugBUplink(cfg SparkplugBConfig) *SparkplugBUplink {
	var tlsCfg *tls.Config
	if cfg.SSL {
		tlsReloader := NewTLSReloader(cfg.CAFile, cfg.CertFile, cfg.KeyFile, cfg.KeyPassword)
		tlsCfg = tlsReloader.GetConfig()
	}
	opts := mqtt.NewClientOptions().
		AddBroker(fmt.Sprintf("%s://%s:%d", func() string {
			if cfg.SSL {
				return "ssl"
			} else {
				return "tcp"
			}
		}(), cfg.Host, cfg.Port)).
		SetClientID(cfg.ClientID).
		SetUsername(cfg.Username).
		SetPassword(cfg.Password).
		SetTLSConfig(tlsCfg).
		SetCleanSession(false).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second)
	client := mqtt.NewClient(opts)
	queue := NewOutboundQueue(nil) // 生产建议传入持久化实现
	return &SparkplugBUplink{Config: cfg, Client: client, NameV: "sparkplugb", Queue: queue}
}

// --- 以下为结构体模拟 Sparkplug B Payload ---
type SimPayload struct {
	Timestamp int64       `json:"timestamp"`
	Metrics   []SimMetric `json:"metrics"`
	Seq       int32       `json:"seq"`
}

type SimMetric struct {
	Name        string  `json:"name"`
	Datatype    int32   `json:"datatype"`
	DoubleValue float64 `json:"double_value,omitempty"`
	IntValue    int32   `json:"int_value,omitempty"`
	StringValue string  `json:"string_value,omitempty"`
	Timestamp   int64   `json:"timestamp"`
}

func BuildSimMetricDouble(name string, value float64) SimMetric {
	return SimMetric{
		Name:        name,
		Datatype:    8,
		DoubleValue: value,
		Timestamp:   time.Now().UnixNano() / 1e6,
	}
}

func BuildSimMetricInt(name string, value int32) SimMetric {
	return SimMetric{
		Name:      name,
		Datatype:  10,
		IntValue:  value,
		Timestamp: time.Now().UnixNano() / 1e6,
	}
}

func BuildSimMetricString(name string, value string) SimMetric {
	return SimMetric{
		Name:        name,
		Datatype:    12,
		StringValue: value,
		Timestamp:   time.Now().UnixNano() / 1e6,
	}
}

// SendSimNBIRTH 发布 NBIRTH 消息（结构体模拟，JSON编码）
func (s *SparkplugBUplink) SendSimNBIRTH(metrics []SimMetric) error {
	if token := s.Client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	payload := SimPayload{
		Timestamp: time.Now().UnixNano() / 1e6,
		Metrics:   metrics,
		Seq:       0,
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	topic := fmt.Sprintf("spBv1.0/%s/NBIRTH/%s", s.Config.GroupID, s.Config.NodeID)
	t := s.Client.Publish(topic, 1, false, buf)
	t.Wait()
	return t.Error()
}

// SendSimNDATA 发布 NDATA 消息（结构体模拟，JSON编码）
func (s *SparkplugBUplink) SendSimNDATA(metrics []SimMetric) error {
	if token := s.Client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	payload := SimPayload{
		Timestamp: time.Now().UnixNano() / 1e6,
		Metrics:   metrics,
		Seq:       0,
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	topic := fmt.Sprintf("spBv1.0/%s/NDATA/%s", s.Config.GroupID, s.Config.NodeID)
	t := s.Client.Publish(topic, 1, false, buf)
	t.Wait()
	return t.Error()
}

// SendSimDBIRTH 发布 DBIRTH 消息（结构体模拟，JSON编码）
func (s *SparkplugBUplink) SendSimDBIRTH(deviceID string, metrics []SimMetric) error {
	if token := s.Client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	payload := SimPayload{
		Timestamp: time.Now().UnixNano() / 1e6,
		Metrics:   metrics,
		Seq:       0,
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	topic := fmt.Sprintf("spBv1.0/%s/DBIRTH/%s/%s", s.Config.GroupID, s.Config.NodeID, deviceID)
	t := s.Client.Publish(topic, 1, false, buf)
	t.Wait()
	return t.Error()
}

// SendSimNDEATH 发布 NDEATH 消息（结构体模拟，JSON编码）
func (s *SparkplugBUplink) SendSimNDEATH() error {
	if token := s.Client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	payload := SimPayload{
		Timestamp: time.Now().UnixNano() / 1e6,
		Seq:       0,
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	topic := fmt.Sprintf("spBv1.0/%s/NDEATH/%s", s.Config.GroupID, s.Config.NodeID)
	t := s.Client.Publish(topic, 1, false, buf)
	t.Wait()
	return t.Error()
}

// SendSimDDEATH 发布 DDEATH 消息（结构体模拟，JSON编码）
func (s *SparkplugBUplink) SendSimDDEATH(deviceID string) error {
	if token := s.Client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	payload := SimPayload{
		Timestamp: time.Now().UnixNano() / 1e6,
		Seq:       0,
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	topic := fmt.Sprintf("spBv1.0/%s/DDEATH/%s/%s", s.Config.GroupID, s.Config.NodeID, deviceID)
	t := s.Client.Publish(topic, 1, false, buf)
	t.Wait()
	return t.Error()
}

func (s *SparkplugBUplink) Name() string { return s.NameV }
func (s *SparkplugBUplink) Type() string { return "sparkplugb" }
