package types

type DeviceConfig struct {
	ID       string                 `yaml:"id"`
	Protocol string                 `yaml:"protocol"`
	Interval int                    `yaml:"interval"`
	Config   map[string]interface{} `yaml:"config"`
}

type Config struct {
	Devices []DeviceConfig `yaml:"devices"`
}
