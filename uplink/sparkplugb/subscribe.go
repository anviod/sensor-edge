package sparkplugb

import (
	"encoding/json"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type SouthSubConfig struct {
	Topic    string
	Template map[string]string // 点位映射模板
}

type SouthSubscriber struct {
	Client   mqtt.Client
	Config   SouthSubConfig
	UplinkFn func([]byte)
}

func (s *SouthSubscriber) Start() {
	s.Client.Subscribe(s.Config.Topic, 1, func(_ mqtt.Client, msg mqtt.Message) {
		var payload map[string]interface{}
		json.Unmarshal(msg.Payload(), &payload)
		filtered := applyMapping(payload, s.Config.Template)
		buf, _ := json.Marshal(filtered)
		s.UplinkFn(buf)
	})
}

func applyMapping(payload map[string]interface{}, tpl map[string]string) map[string]interface{} {
	res := make(map[string]interface{})
	for k, v := range tpl {
		if val, ok := payload[k]; ok {
			res[v] = val
		}
	}
	return res
}
