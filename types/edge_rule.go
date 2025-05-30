package types

type AggregateRule struct {
	ID          string `yaml:"id"`
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
	DeviceID    string `yaml:"device_id"`
	Point       string `yaml:"point"`
	Method      string `yaml:"method"`
	Window      int    `yaml:"window"`
}

type AlarmRuleEdge struct {
	ID          string `yaml:"id"`
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
	DeviceID    string `yaml:"device_id"`
	Point       string `yaml:"point"`
	Condition   string `yaml:"condition"`
	Level       string `yaml:"level"`
	Message     string `yaml:"message"`
}

type LinkageRule struct {
	ID            string `yaml:"id"`
	Description   string `yaml:"description"`
	Type          string `yaml:"type"`
	SourceDevice  string `yaml:"source_device"`
	SourcePoint   string `yaml:"source_point"`
	Condition     string `yaml:"condition"`
	ActionDevice  string `yaml:"action_device"`
	ActionAddress string `yaml:"action_address"`
	ActionValue   any    `yaml:"action_value"`
}

// 滑动窗口缓存
// 用于聚合规则

type PointBuffer struct {
	Values []float64
	Size   int
}

func (b *PointBuffer) Add(v float64) {
	b.Values = append(b.Values, v)
	if len(b.Values) > b.Size {
		b.Values = b.Values[1:]
	}
}

func (b *PointBuffer) Avg() float64 {
	sum := 0.0
	for _, v := range b.Values {
		sum += v
	}
	if len(b.Values) == 0 {
		return 0
	}
	return sum / float64(len(b.Values))
}

// 新版分组式边缘规则结构体
// 支持 device_id 下聚合、报警、联动多类型规则

type DeviceEdgeRules struct {
	DeviceID  string          `yaml:"device_id"`
	Aggregate []AggregateRule `yaml:"aggregate,omitempty"`
	Alarm     []AlarmRuleEdge `yaml:"alarm,omitempty"`
	Linkage   []LinkageRule   `yaml:"linkage,omitempty"`
}
