package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type UplinkConfig struct {
	Type     string            `yaml:"type"`
	Name     string            `yaml:"name"`
	Enable   bool              `yaml:"enable"`
	Broker   string            `yaml:"broker"`
	ClientID string            `yaml:"client_id"`
	Username string            `yaml:"username"`
	Password string            `yaml:"password"`
	Topic    string            `yaml:"topic"`
	URL      string            `yaml:"url"`
	Method   string            `yaml:"method"`
	Headers  map[string]string `yaml:"headers"`
	Brokers  []string          `yaml:"brokers"`
	Server   string            `yaml:"server"`
	Subject  string            `yaml:"subject"`
}

// LoadUplinkConfigs loads the uplink configurations from the specified file.
// The default file is configs/uplinks.yaml
func LoadUplinkConfigs(file string) ([]UplinkConfig, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var cfgs []UplinkConfig
	err = yaml.Unmarshal(data, &cfgs)
	return cfgs, err
}
