package schema

// AlarmInfo 报警信息结构体
type AlarmInfo struct {
	Name    string `json:"name"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

// DataReport 上行数据报文结构体
type DataReport struct {
	DeviceID  string                 `json:"device_id"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Alarm     []AlarmInfo            `json:"alarm,omitempty"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
}
