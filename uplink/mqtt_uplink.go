package uplink

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTTUplink 上行MQTT协议处理
type MQTTUplink struct {
	client mqtt.Client
	topic  string
	name   string
}

func (m *MQTTUplink) Publish(data interface{}) error {
	// 发布数据到MQTT
	return nil
}

func (m *MQTTUplink) Send(data []byte) error {
	token := m.client.Publish(m.topic, 1, false, data)
	token.Wait()
	return token.Error()
}

func (m *MQTTUplink) Name() string { return m.name }
func (m *MQTTUplink) Type() string { return "mqtt" }
