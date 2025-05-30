package mqtt

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTTUplink 上行MQTT协议处理
type MqttUplink struct {
	client mqtt.Client
	topic  string
	name   string
}

func (m *MqttUplink) Send(data []byte) error {
	token := m.client.Publish(m.topic, 1, false, data)
	token.Wait()
	return token.Error()
}

func (m *MqttUplink) Name() string { return m.name }
func (m *MqttUplink) Type() string { return "mqtt" }

// NewMqttUplink 构造函数，便于外部包初始化
func NewMqttUplink(client mqtt.Client, topic, name string) *MqttUplink {
	return &MqttUplink{
		client: client,
		topic:  topic,
		name:   name,
	}
}
