package uplink

import (
	"sensor-edge/uplink/http"
	"sensor-edge/uplink/kafka"
	"sensor-edge/uplink/mqtt"
	"sensor-edge/uplink/nats"
	"sensor-edge/uplink/redis"
)

// Uplink 通用上行接口
// 支持MQTT/HTTP/Kafka/NATS等

// UplinkFactory 用于统一创建各类 uplink 实例
type UplinkFactory struct{}

func (f *UplinkFactory) NewUplink(typ string) Uplink {
	switch typ {
	case "mqtt":
		return &mqtt.MqttUplink{}
	case "http":
		return &http.HTTPClientUplink{}
	case "nats":
		return &nats.NatsUplink{}
	case "redis":
		return &redis.RedisUplink{}
	case "kafka":
		return &kafka.KafkaUplink{}
	default:
		return nil
	}
}
