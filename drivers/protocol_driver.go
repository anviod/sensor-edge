package drivers

// ProtocolDriver 是所有协议驱动需实现的统一接口
// Init: 初始化驱动，Read/Write: 读写点位，Close: 释放资源
// 可扩展心跳、重连、诊断等方法

type ProtocolDriver interface {
	Init(config map[string]interface{}) error
	Read(deviceID string) (map[string]interface{}, error)
	Write(point string, value interface{}) error
	Close() error
}
