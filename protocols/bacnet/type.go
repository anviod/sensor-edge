package bacnet

// BACnet对象类型

type ObjectType uint16

const (
	AnalogInput           ObjectType = 0x00 // 模拟量输入 *
	AnalogOutput          ObjectType = 0x01 // 模拟量输出 *
	AnalogValue           ObjectType = 0x02 // 模拟量值 *
	BinaryInput           ObjectType = 0x03 // 开关量输入 *
	BinaryOutput          ObjectType = 0x04 // 开关量输出 *
	BinaryValue           ObjectType = 0x05 // 开关量值 *
	Calendar              ObjectType = 0x06 // 日历
	Command               ObjectType = 0x07 // 命令
	BacnetDevice          ObjectType = 0x08 // 设备对象 *
	EventEnrollment       ObjectType = 0x09 // 事件注册
	File                  ObjectType = 0x0A // 文件
	Group                 ObjectType = 0x0B // 组
	Loop                  ObjectType = 0x0C // 回路
	MultiStateInput       ObjectType = 0x0D // 多状态输入 *
	MultiStateOutput      ObjectType = 0x0E // 多状态输出 *
	NotificationClass     ObjectType = 0x0F // 通知类
	Program               ObjectType = 0x10 // 程序
	Schedule              ObjectType = 0x11 // 日程
	Averaging             ObjectType = 0x12 // 平均值
	MultiStateValue       ObjectType = 0x13 // 多状态值 *
	Trendlog              ObjectType = 0x14 // 趋势日志
	LifeSafetyPoint       ObjectType = 0x15 // 生命安全点
	LifeSafetyZone        ObjectType = 0x16 // 生命安全区
	Accumulator           ObjectType = 0x17 // 累加器
	PulseConverter        ObjectType = 0x18 // 脉冲转换器
	EventLog              ObjectType = 0x19 // 事件日志
	GlobalGroup           ObjectType = 0x1A // 全局组
	TrendLogMultiple      ObjectType = 0x1B // 多趋势日志
	LoadControl           ObjectType = 0x1C // 负载控制
	StructuredView        ObjectType = 0x1D // 结构化视图
	AccessDoor            ObjectType = 0x1E // 门禁门
	Timer                 ObjectType = 0x1F // 定时器
	AccessCredential      ObjectType = 0x20 // 门禁凭证
	AccessPoint           ObjectType = 0x21 // 门禁点
	AccessRights          ObjectType = 0x22 // 门禁权限
	AccessUser            ObjectType = 0x23 // 门禁用户
	AccessZone            ObjectType = 0x24 // 门禁区域
	CredentialDataInput   ObjectType = 0x25 // 凭证数据输入
	NetworkSecurity       ObjectType = 0x26 // 网络安全
	BitstringValue        ObjectType = 0x27 // 位串值
	CharacterstringValue  ObjectType = 0x28 // 字符串值
	DatePatternValue      ObjectType = 0x29 // 日期模式值
	DateValue             ObjectType = 0x2a // 日期值
	DatetimePatternValue  ObjectType = 0x2b // 日期时间模式值
	DatetimeValue         ObjectType = 0x2c // 日期时间值
	IntegerValue          ObjectType = 0x2d // 整数值
	LargeAnalogValue      ObjectType = 0x2e // 大模拟量值
	OctetstringValue      ObjectType = 0x2f // 八位字节串值
	PositiveIntegerValue  ObjectType = 0x30 // 正整数值
	TimePatternValue      ObjectType = 0x31 // 时间模式值
	TimeValue             ObjectType = 0x32 // 时间值
	NotificationForwarder ObjectType = 0x33 // 通知转发器
	AlertEnrollment       ObjectType = 0x34 // 报警注册
	Channel               ObjectType = 0x35 // 通道
	LightingOutput        ObjectType = 0x36 // 灯光输出
	BinaryLightingOutput  ObjectType = 0x37 // 开关灯光输出
	NetworkPort           ObjectType = 0x38 // 网络端口
	ProprietaryMin        ObjectType = 0x80 // 厂商自定义最小值
	Proprietarymax        ObjectType = 0x3ff // 厂商自定义最大值
)

// BACnet对象唯一标识

type ObjectID struct {
	Type     ObjectType // 对象类型
	Instance uint32     // 实例号
}

// BACnet属性值类型

type PropertyValueType byte

const (
	TypeNull            PropertyValueType = 0  // 空
	TypeBoolean         PropertyValueType = 1  // 布尔
	TypeUnsignedInt     PropertyValueType = 2  // 无符号整型
	TypeSignedInt       PropertyValueType = 3  // 有符号整型
	TypeReal            PropertyValueType = 4  // 单精度浮点
	TypeDouble          PropertyValueType = 5  // 双精度浮点
	TypeOctetString     PropertyValueType = 6  // 字节串
	TypeCharacterString PropertyValueType = 7  // 字符串
	TypeBitString       PropertyValueType = 8  // 位串
	TypeEnumerated      PropertyValueType = 9  // 枚举
	TypeDate            PropertyValueType = 10 // 日期
	TypeTime            PropertyValueType = 11 // 时间
	TypeObjectID        PropertyValueType = 12 // 对象ID
)

// BACnet属性值

type PropertyValue struct {
	Type  PropertyValueType // 属性值类型
	Value interface{}       // 属性值
}
