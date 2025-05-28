package nats

// NatsUplink 实现 uplink.Uplink 接口
// ...具体实现略...

type NatsUplink struct{}

func (n *NatsUplink) Send(data []byte) error { return nil }
func (n *NatsUplink) Name() string           { return "nats" }
func (n *NatsUplink) Type() string           { return "NATS" }
