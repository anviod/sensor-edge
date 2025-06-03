package protocols

// PointValue 定义了数据点的结构体
type PointValue struct {
	PointID   string
	Value     interface{}
	Quality   string
	Timestamp int64
}

// Protocol 是所有协议模块需实现的通用接口
type Protocol interface {
	Init(config map[string]interface{}) error
	Read(deviceID string) ([]PointValue, error)
	// 批量读取接口，增加 function 功能码参数
	ReadBatch(deviceID string, function string, points []string) ([]PointValue, error)
	Write(point string, value interface{}) error // 新增写入接口，便于联动控制
	Close() error
	Reconnect() error // 新增重连接口，便于处理连接异常
}
