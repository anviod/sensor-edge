package config

import (
	"encoding/json"
	"os"
	"sensor-edge/types"

	"gopkg.in/yaml.v3"
)

func LoadDevicesFromYAML(file string) ([]types.DeviceConfigWithMeta, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var devs []types.DeviceConfigWithMeta
	err = yaml.Unmarshal(data, &devs)
	return devs, err
}

func LoadDevicesFromJSON(file string) ([]types.DeviceConfigWithMeta, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var devs []types.DeviceConfigWithMeta
	err = json.Unmarshal(data, &devs)
	return devs, err
}

// configs/devices.yaml
