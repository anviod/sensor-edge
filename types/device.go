package types

type DeviceMeta struct {
	ID           string `yaml:"id" json:"id"`
	Name         string `yaml:"name" json:"name"`
	Description  string `yaml:"description" json:"description"`
	Protocol     string `yaml:"protocol" json:"protocol"`
	ProtocolName string `yaml:"protocol_name" json:"protocol_name"`
	Interval     int    `yaml:"interval" json:"interval"`
	EnablePing   bool   `yaml:"enable_ping" json:"enable_ping"`
}

type DeviceConfigWithMeta struct {
	DeviceMeta `yaml:",inline" json:",inline"`
	Config     map[string]interface{} `yaml:"config" json:"config"`
}
