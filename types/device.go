package types

type DeviceMeta struct {
	ID           string `yaml:"id"`
	Name         string `yaml:"name"`
	Description  string `yaml:"description"`
	Protocol     string `yaml:"protocol"`
	ProtocolName string `yaml:"protocol_name"`
	Interval     string `yaml:"interval"`
	EnablePing   bool   `yaml:"enable_ping"`
	IP           string `yaml:"ip"`
	Port         int    `yaml:"port"`
	SlaveID      int    `yaml:"slave_id"` // 新增字段
}

type DeviceConfigWithMeta struct {
	DeviceMeta `yaml:",inline" json:",inline"`
	Config     map[string]interface{} `yaml:"config" json:"config"`
}
