package protocols

type PointConfig struct {
	PointID   string // 点位唯一标识（如物模型名或address）
	Address   string // Modbus寄存器地址（如"40001"）
	Type      string // 数据类型（如int/float/bool）
	Unit      string // 单位
	Transform string // 转换表达式
	Format    string // 格式化类型（如 INT、Float AB CD、Double AB CD EF GH 等）
}
