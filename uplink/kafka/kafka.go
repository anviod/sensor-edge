package kafka

// KafkaUplink 实现 uplink.Uplink 接口
// 这里只做接口和结构体定义，实际生产建议用 segmentio/kafka-go 或 confluent-kafka-go

type KafkaUplink struct {
	Topic string
	NameV string
}

func (k *KafkaUplink) Send(data []byte) error {
	// 这里应调用 kafka-go 的 Writer 写入消息，演示用
	// 实际生产环境请补充连接与错误处理
	return nil
}
func (k *KafkaUplink) Name() string { return k.NameV }
func (k *KafkaUplink) Type() string { return "kafka" }
