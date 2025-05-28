package models

// Device 设备元数据结构体
// 包含ID、名称、描述、协议、采集周期、心跳等

type Device struct {
	ID          string
	Name        string
	Description string
	Protocol    string
	Interval    int
	EnablePing  bool
	Config      map[string]interface{}
}

// Point 点位物模型结构体
// 包含原始地址、物模型名、类型、单位、转换、报警等

type Point struct {
	Address   string
	Name      string
	Type      string
	Unit      string
	Transform string
	Alarm     AlarmRule
}

// AlarmRule 报警规则结构体

type AlarmRule struct {
	Enable    bool
	Condition string
	Level     string
	Message   string
}

// UplinkConfig 上行通道配置结构体

type UplinkConfig struct {
	Type     string
	Name     string
	Enable   bool
	Broker   string
	ClientID string
	Username string
	Password string
	Topic    string
	URL      string
	Method   string
	Headers  map[string]string
	Brokers  []string
	Server   string
	Subject  string
}
