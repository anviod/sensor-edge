package core

import (
	"fmt"
	"net"
	"sensor-edge/protocols"
	"sensor-edge/types"
	"time"
)

type Device struct {
	Meta    types.DeviceMeta
	Client  protocols.Protocol
	Healthy bool
}

type DeviceManager struct {
	Devices map[string]*Device
}

func NewDeviceManager() *DeviceManager {
	return &DeviceManager{
		Devices: make(map[string]*Device),
	}
}

func (m *DeviceManager) Register(dev types.DeviceConfigWithMeta) error {
	client, err := protocols.Create(dev.Protocol)
	if err != nil {
		return fmt.Errorf("unsupported protocol: %s", dev.Protocol)
	}

	if err := client.Init(dev.Config); err != nil {
		return fmt.Errorf("init error: %v", err)
	}

	m.Devices[dev.ID] = &Device{
		Meta:    dev.DeviceMeta,
		Client:  client,
		Healthy: true,
	}

	// 启动心跳检测协程
	if dev.EnablePing {
		go m.startHealthMonitor(dev.ID, dev.Config)
	}

	// 本地联动写入演示：注册时可直接写入一个初始值
	if dev.Protocol == "http" && dev.ID == "plc2" {
		_ = client.Write("d200", 0) // 假设支持Write
	}

	return nil
}

func (m *DeviceManager) startHealthMonitor(id string, config map[string]interface{}) {
	for {
		ip := config["ip"].(string)
		conn, err := net.DialTimeout("tcp", ip+":80", 2*time.Second)
		if err != nil {
			fmt.Printf("[WARN] Device %s offline\n", id)
			m.Devices[id].Healthy = false
		} else {
			conn.Close()
			m.Devices[id].Healthy = true
		}
		time.Sleep(10 * time.Second)
	}
}
