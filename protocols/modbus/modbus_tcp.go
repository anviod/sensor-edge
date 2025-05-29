package modbus

import (
	"encoding/binary"
	"fmt"
	"sort"
	"time"

	"sensor-edge/protocols"

	"github.com/goburrow/modbus"
)

type ModbusTCP struct {
	client  modbus.Client
	handler *modbus.TCPClientHandler
}

func (m *ModbusTCP) Init(config map[string]interface{}) error {
	ip := config["ip"].(string)
	var port int
	switch v := config["port"].(type) {
	case float64:
		port = int(v)
	case int:
		port = v
	case int64:
		port = int(v)
	default:
		return fmt.Errorf("invalid port type: %T", v)
	}
	var slaveId byte
	switch v := config["slave_id"].(type) {
	case float64:
		slaveId = byte(v)
	case int:
		slaveId = byte(v)
	case int64:
		slaveId = byte(v)
	default:
		return fmt.Errorf("invalid slave_id type: %T", v)
	}
	addr := fmt.Sprintf("%s:%d", ip, port)
	handler := modbus.NewTCPClientHandler(addr)
	handler.Timeout = 5 * time.Second
	handler.SlaveId = slaveId
	if err := handler.Connect(); err != nil {
		return err
	}
	m.handler = handler
	m.client = modbus.NewClient(handler)
	return nil
}

func (m *ModbusTCP) ReadCoils(address, quantity uint16) ([]bool, error) {
	results, err := m.client.ReadCoils(address, quantity)
	if err != nil {
		return nil, err
	}
	coils := make([]bool, quantity)
	for i := uint16(0); i < quantity; i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		coils[i] = (results[byteIndex]>>bitIndex)&0x01 == 0x01
	}
	return coils, nil
}

func (m *ModbusTCP) ReadDiscreteInputs(address, quantity uint16) ([]bool, error) {
	results, err := m.client.ReadDiscreteInputs(address, quantity)
	if err != nil {
		return nil, err
	}
	inputs := make([]bool, quantity)
	for i := uint16(0); i < quantity; i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		inputs[i] = (results[byteIndex]>>bitIndex)&0x01 == 0x01
	}
	return inputs, nil
}

func (m *ModbusTCP) ReadHoldingRegisters(address, quantity uint16) ([]uint16, error) {
	results, err := m.client.ReadHoldingRegisters(address, quantity)
	if err != nil {
		return nil, err
	}
	regs := make([]uint16, quantity)
	for i := 0; i < int(quantity); i++ {
		regs[i] = binary.BigEndian.Uint16(results[2*i : 2*i+2])
	}
	return regs, nil
}

func (m *ModbusTCP) ReadInputRegisters(address, quantity uint16) ([]uint16, error) {
	results, err := m.client.ReadInputRegisters(address, quantity)
	if err != nil {
		return nil, err
	}
	regs := make([]uint16, quantity)
	for i := 0; i < int(quantity); i++ {
		regs[i] = binary.BigEndian.Uint16(results[2*i : 2*i+2])
	}
	return regs, nil
}

func (m *ModbusTCP) WriteSingleCoil(address uint16, value bool) error {
	var v uint16
	if value {
		v = 0xFF00
	}
	_, err := m.client.WriteSingleCoil(address, v)
	return err
}

func (m *ModbusTCP) WriteSingleRegister(address, value uint16) error {
	_, err := m.client.WriteSingleRegister(address, value)
	return err
}

func (m *ModbusTCP) WriteMultipleCoils(address uint16, values []bool) error {
	byteCount := (len(values) + 7) / 8
	buf := make([]byte, byteCount)
	for i, b := range values {
		if b {
			buf[i/8] |= 1 << (uint(i) % 8)
		}
	}
	_, err := m.client.WriteMultipleCoils(address, uint16(len(values)), buf)
	return err
}

func (m *ModbusTCP) WriteMultipleRegisters(address uint16, values []uint16) error {
	buf := make([]byte, 2*len(values))
	for i, val := range values {
		binary.BigEndian.PutUint16(buf[2*i:], val)
	}
	_, err := m.client.WriteMultipleRegisters(address, uint16(len(values)), buf)
	return err
}

func (m *ModbusTCP) Read(deviceID string) ([]protocols.PointValue, error) {
	// 示例：读取 Holding Registers 10个寄存器，并映射为业务点
	regs, err := m.ReadHoldingRegisters(0, 10)
	if err != nil {
		return nil, err
	}
	// 这里你可以根据点位映射，将寄存器值转换为 PointValue
	var values []protocols.PointValue
	for i, r := range regs {
		values = append(values, protocols.PointValue{
			PointID:   fmt.Sprintf("reg%d", i),
			Value:     r,
			Quality:   "good",
			Timestamp: time.Now().Unix(),
		})
	}
	return values, nil
}

// ReadBatch 批量读取接口，支持自动分段连续区间优化，寄存器最大125，线圈最大64
func (m *ModbusTCP) ReadBatch(deviceID string, points []string) ([]protocols.PointValue, error) {
	if len(points) == 0 {
		return nil, nil
	}
	type addrPoint struct {
		addr uint16
		name string
	}
	var addrPoints []addrPoint
	for _, pt := range points {
		addr, err := parseAddress(pt)
		if err != nil {
			return nil, fmt.Errorf("parse address for point %s failed: %v", pt, err)
		}
		addrPoints = append(addrPoints, addrPoint{addr, pt})
	}
	sort.Slice(addrPoints, func(i, j int) bool { return addrPoints[i].addr < addrPoints[j].addr })

	var (
		results []protocols.PointValue
		n       = len(addrPoints)
		i       = 0
	)
	const maxRegs = 125
	for i < n {
		start := i
		end := i
		for end+1 < n && addrPoints[end+1].addr == addrPoints[end].addr+1 && (addrPoints[end+1].addr-addrPoints[start].addr+1) <= maxRegs {
			end++
		}
		baseAddr := addrPoints[start].addr
		quantity := addrPoints[end].addr - baseAddr + 1
		regVals, err := m.ReadHoldingRegisters(baseAddr, quantity)
		if err != nil {
			return nil, fmt.Errorf("modbus batch read failed: %v", err)
		}
		for k := start; k <= end; k++ {
			offset := addrPoints[k].addr - baseAddr
			results = append(results, protocols.PointValue{
				PointID:   addrPoints[k].name,
				Value:     regVals[offset],
				Quality:   "good",
				Timestamp: time.Now().Unix(),
			})
		}
		i = end + 1
	}
	return results, nil
}

// parseAddress 将点位字符串如 "40001" 转为寄存器编号
func parseAddress(point string) (uint16, error) {
	if len(point) < 2 {
		return 0, fmt.Errorf("invalid point address")
	}
	var base int
	switch point[0] {
	case '4':
		base = 40001
	default:
		base = 1
	}
	var addr int
	_, err := fmt.Sscanf(point, "%d", &addr)
	if err != nil {
		return 0, err
	}
	return uint16(addr - base), nil
}

func (m *ModbusTCP) Close() error {
	return m.handler.Close()
}

func NewModbusTCP() protocols.Protocol {
	return &ModbusTCP{}
}

func init() {
	protocols.Register("modbus_tcp", NewModbusTCP)
}

// Write 单点写入，支持线圈和寄存器
func (m *ModbusTCP) Write(point string, value interface{}) error {
	addr, err := parseAddress(point)
	if err != nil {
		return err
	}
	switch v := value.(type) {
	case bool:
		return m.WriteSingleCoil(addr, v)
	case uint16:
		return m.WriteSingleRegister(addr, v)
	case int:
		return m.WriteSingleRegister(addr, uint16(v))
	default:
		return fmt.Errorf("unsupported value type: %T", value)
	}
}

func (m *ModbusTCP) SetSlave(slaveId byte) {
	if m.handler != nil {
		m.handler.SlaveId = slaveId
	}
}
