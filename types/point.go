package types

type AlarmRule struct {
	Enable    bool   `yaml:"enable"`
	Condition string `yaml:"condition"` // 表达式：value > 50
	Level     string `yaml:"level"`     // info/warning/critical
	Message   string `yaml:"message"`
}

type PointMapping struct {
	Address   string    `yaml:"address"`   // 原始点位
	Name      string    `yaml:"name"`      // 物模型字段名
	Type      string    `yaml:"type"`      // bool/int/float/string
	Unit      string    `yaml:"unit"`      // 单位
	Transform string    `yaml:"transform"` // 转换表达式
	Format    string    `yaml:"format"`    // 格式化类型（如 INT、Long AB CD 等）
	Alarm     AlarmRule `yaml:"alarm"`
}

type DevicePointSet struct {
	DeviceID     string         `yaml:"device_id"`
	Protocol     string         `yaml:"protocol"`
	ProtocolName string         `yaml:"protocol_name"`
	Points       []PointMapping `yaml:"points"`
}
