package uplink

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sensor-edge/config"

	kafkauplink "sensor-edge/uplink/kafka"
	natsuplink "sensor-edge/uplink/nats"
	redisuplink "sensor-edge/uplink/redis"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type UplinkManager struct {
	uplinks []Uplink
}

func NewUplinkManager(uplinks []Uplink) *UplinkManager {
	return &UplinkManager{uplinks: uplinks}
}

func NewUplinkManagerFromConfig(cfgs []config.UplinkConfig) *UplinkManager {
	uplinks := []Uplink{}
	for _, c := range cfgs {
		if !c.Enable {
			continue
		}
		switch c.Type {
		case "mqtt":
			opts := mqtt.NewClientOptions().AddBroker(c.Broker).SetClientID(c.ClientID)
			if c.Username != "" {
				opts.SetUsername(c.Username)
			}
			if c.Password != "" {
				opts.SetPassword(c.Password)
			}
			client := mqtt.NewClient(opts)
			if token := client.Connect(); token.Wait() && token.Error() != nil {
				fmt.Println("[Uplink] MQTT connect error:", token.Error())
				continue
			}
			uplinks = append(uplinks, &MQTTUplink{client: client, topic: c.Topic, name: c.Name})
		case "http":
			uplinks = append(uplinks, &HttpUplink{url: c.URL, method: c.Method, headers: c.Headers, name: c.Name})
		case "kafka":
			uplinks = append(uplinks, &kafkauplink.KafkaUplink{Topic: c.Topic, NameV: c.Name})
		case "nats":
			uplinks = append(uplinks, &natsuplink.NatsUplink{}) // 生产应传入连接参数
		case "redis":
			uplinks = append(uplinks, &redisuplink.RedisUplink{}) // 生产应传入连接参数
			// 可扩展其他协议
		}
	}
	return &UplinkManager{uplinks: uplinks}
}

func (m *UplinkManager) SendToAll(payload []byte)  error{
	data := make(map[string]interface{})
	for _, up := range m.uplinks {
		if err := up.Send(payload); err != nil {
			log.Printf("[UplinkError] %s: %v", up.Name(), err)
		}
		// 本地持久化上报日志
		f, _ := os.OpenFile("uplink.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		defer f.Close()
		json.Unmarshal(payload, &data) // 假设payload是map[string]interface{}类型
		json.NewEncoder(f).Encode(map[string]any{"uplink": up.Name(), "type": up.Type(), "data": data})
	}
	return nil
}
