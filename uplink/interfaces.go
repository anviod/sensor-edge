package uplink

// Uplink 通用上行接口
// Send: 发送数据，Name/Type: 通道标识
// 可扩展支持重试、批量、异步、状态监控等

type Uplink interface {
	Send(data []byte) error
	Name() string
	Type() string
}

// Formatter 上报字段映射/模板接口
// Format: 按模板渲染数据

type Formatter interface {
	Format(data map[string]interface{}) ([]byte, error)
}
