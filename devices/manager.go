package devices

import "sensor-edge/protocols"

// Device 设备对象定义
type Device struct {
	ID       string
	Protocol protocols.Protocol
	Points   map[string]interface{}
}

// Manager 设备管理器
type Manager struct {
	Devices map[string]*Device
}

// NewManager 创建设备管理器
func NewManager() *Manager {
	return &Manager{Devices: make(map[string]*Device)}
}

// AddDevice 添加设备
func (m *Manager) AddDevice(device *Device) {
	m.Devices[device.ID] = device
}
