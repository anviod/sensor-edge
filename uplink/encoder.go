package uplink

import (
	"encoding/json"
	"sensor-edge/schema"
	"time"
)

// EncodeDataReport 统一格式编码
func EncodeDataReport(deviceID string, points map[string]interface{}, alarms []schema.AlarmInfo, metrics map[string]interface{}) []byte {
	report := schema.DataReport{
		DeviceID:  deviceID,
		Timestamp: time.Now().UTC().Unix(),
		Data:      points,
		Alarm:     alarms,
		Metrics:   metrics,
	}
	buf, _ := json.Marshal(report)
	return buf
}
