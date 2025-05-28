package mqtt

// MqttUplink 实现 uplink.Uplink 接口
// ...具体实现略...

type MqttUplink struct{}

func (m *MqttUplink) Send(data []byte) error { return nil }
func (m *MqttUplink) Name() string           { return "mqtt" }
func (m *MqttUplink) Type() string           { return "MQTT" }
