package modbus

import (
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
	"time"

	"sensor-edge/protocols"

	"github.com/goburrow/modbus"
)

type ModbusTCP struct {
	client  modbus.Client
	handler *modbus.TCPClientHandler
}

func (m *ModbusTCP) Init(config map[string]interface{}) error {
	ip, ok := config["ip"].(string)
	if !ok {
		return fmt.Errorf("invalid ip address")
	}
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
	regs, err := m.client.ReadHoldingRegisters(0, 10)
	if err != nil {
		return nil, err
	}
	values := make([]protocols.PointValue, len(regs)/2)
	for i := 0; i < len(regs)/2; i++ {
		val := binary.BigEndian.Uint16(regs[i*2 : i*2+2])
		values[i] = protocols.PointValue{
			PointID:   fmt.Sprintf("reg%d", i),
			Value:     val,
			Quality:   "good",
			Timestamp: time.Now().Unix(),
		}
	}
	return values, nil
}

// ReadBatch 实现接口要求的方法：接受功能码和点位地址
func (m *ModbusTCP) ReadBatch(deviceID string, function string, points []string) ([]protocols.PointValue, error) {
	// 仅支持03/04/01/02功能码，默认03
	if function == "" {
		function = "03"
	}
	pointConfigs := make([]protocols.PointConfig, len(points))
	for i, pt := range points {
		pointConfigs[i] = protocols.PointConfig{
			PointID: pt,
			Address: pt,
			Format:  "", // 默认无格式解析
		}
	}
	// 按功能码分流
	switch function {
	case "03":
		return m.ReadBatchWithFormat(deviceID, pointConfigs)
	case "04":
		return m.readInputRegistersBatch(deviceID, pointConfigs)
	case "01":
		return m.readCoilsBatch(deviceID, pointConfigs)
	case "02":
		return m.readDiscreteInputsBatch(deviceID, pointConfigs)
	default:
		return m.ReadBatchWithFormat(deviceID, pointConfigs)
	}
}

// 批量读取保持寄存器（功能码03），支持格式化
func (m *ModbusTCP) ReadBatchWithFormat(deviceID string, points []protocols.PointConfig) ([]protocols.PointValue, error) {
	if len(points) == 0 {
		return nil, nil
	}
	type addrPoint struct {
		addr   uint16
		name   string
		format string
	}
	var addrPoints []addrPoint
	for _, pt := range points {
		addr, err := parseAddress(pt.Address)
		if err != nil {
			return nil, fmt.Errorf("parse address for point %s failed: %v", pt.Address, err)
		}
		addrPoints = append(addrPoints, addrPoint{addr, pt.PointID, pt.Format})
	}
	sort.Slice(addrPoints, func(i, j int) bool { return addrPoints[i].addr < addrPoints[j].addr })

	getRegCount := func(format string) int {
		f := strings.ToUpper(format)
		if strings.HasPrefix(f, "FLOAT") || strings.HasPrefix(f, "LONG") {
			return 2
		}
		if strings.HasPrefix(f, "DOUBLE") {
			return 4
		}
		return 1
	}

	var results []protocols.PointValue
	n := len(addrPoints)
	i := 0
	const maxRegs = 125
	for i < n {
		start := i
		end := i
		maxAddr := addrPoints[start].addr + uint16(getRegCount(addrPoints[start].format)) - 1
		for j := i + 1; j < n; j++ {
			need := getRegCount(addrPoints[j].format)
			if addrPoints[j].addr <= maxAddr+1 && (addrPoints[j].addr+uint16(need)-addrPoints[start].addr) < maxRegs {
				if addrPoints[j].addr+uint16(need)-1 > maxAddr {
					maxAddr = addrPoints[j].addr + uint16(need) - 1
				}
				end = j
			} else {
				break
			}
		}
		baseAddr := addrPoints[start].addr
		quantity := maxAddr - baseAddr + 1
		regVals, err := m.client.ReadHoldingRegisters(baseAddr, quantity)
		if err != nil {
			return nil, fmt.Errorf("modbus batch read failed: %v", err)
		}
		for k := start; k <= end; k++ {
			offset := addrPoints[k].addr - baseAddr
			regCount := getRegCount(addrPoints[k].format)
			if int(offset)+regCount > len(regVals)/2 {
				results = append(results, protocols.PointValue{
					PointID:   addrPoints[k].name,
					Value:     nil,
					Quality:   "bad",
					Timestamp: time.Now().Unix(),
				})
				continue
			}
			vals := make([]uint16, regCount)
			for r := 0; r < regCount; r++ {
				vals[r] = binary.BigEndian.Uint16(regVals[(int(offset)+r)*2 : (int(offset)+r)*2+2])
			}
			var val interface{} = vals
			if regCount == 1 {
				val = vals[0]
			}
			results = append(results, protocols.PointValue{
				PointID:   addrPoints[k].name,
				Value:     val,
				Quality:   "good",
				Timestamp: time.Now().Unix(),
			})
		}
		i = end + 1
	}
	return results, nil
}

// 新增：批量读取输入寄存器（功能码04）
func (m *ModbusTCP) readInputRegistersBatch(deviceID string, points []protocols.PointConfig) ([]protocols.PointValue, error) {
	// 逻辑与 ReadBatchWithFormat 类似，只是调用 ReadInputRegisters
	if len(points) == 0 {
		return nil, nil
	}
	type addrPoint struct {
		addr   uint16
		name   string
		format string
	}
	var addrPoints []addrPoint
	for _, pt := range points {
		addr, err := parseAddress(pt.Address)
		if err != nil {
			return nil, fmt.Errorf("parse address for point %s failed: %v", pt.Address, err)
		}
		addrPoints = append(addrPoints, addrPoint{addr, pt.PointID, pt.Format})
	}
	sort.Slice(addrPoints, func(i, j int) bool { return addrPoints[i].addr < addrPoints[j].addr })

	getRegCount := func(format string) int {
		f := strings.ToUpper(format)
		if strings.HasPrefix(f, "FLOAT") || strings.HasPrefix(f, "LONG") {
			return 2
		}
		if strings.HasPrefix(f, "DOUBLE") {
			return 4
		}
		return 1
	}

	var results []protocols.PointValue
	n := len(addrPoints)
	i := 0
	const maxRegs = 125
	for i < n {
		start := i
		end := i
		maxAddr := addrPoints[start].addr + uint16(getRegCount(addrPoints[start].format)) - 1
		for j := i + 1; j < n; j++ {
			need := getRegCount(addrPoints[j].format)
			if addrPoints[j].addr <= maxAddr+1 && (addrPoints[j].addr+uint16(need)-addrPoints[start].addr) < maxRegs {
				if addrPoints[j].addr+uint16(need)-1 > maxAddr {
					maxAddr = addrPoints[j].addr + uint16(need) - 1
				}
				end = j
			} else {
				break
			}
		}
		baseAddr := addrPoints[start].addr
		quantity := maxAddr - baseAddr + 1
		regVals, err := m.client.ReadInputRegisters(baseAddr, quantity)
		if err != nil {
			return nil, fmt.Errorf("modbus input batch read failed: %v", err)
		}
		for k := start; k <= end; k++ {
			offset := addrPoints[k].addr - baseAddr
			regCount := getRegCount(addrPoints[k].format)
			if int(offset)+regCount > len(regVals)/2 {
				results = append(results, protocols.PointValue{
					PointID:   addrPoints[k].name,
					Value:     nil,
					Quality:   "bad",
					Timestamp: time.Now().Unix(),
				})
				continue
			}
			vals := make([]uint16, regCount)
			for r := 0; r < regCount; r++ {
				vals[r] = binary.BigEndian.Uint16(regVals[(int(offset)+r)*2 : (int(offset)+r)*2+2])
			}
			var val interface{} = vals
			if regCount == 1 {
				val = vals[0]
			}
			results = append(results, protocols.PointValue{
				PointID:   addrPoints[k].name,
				Value:     val,
				Quality:   "good",
				Timestamp: time.Now().Unix(),
			})
		}
		i = end + 1
	}
	return results, nil
}

// 新增：批量读取线圈（功能码01）
func (m *ModbusTCP) readCoilsBatch(deviceID string, points []protocols.PointConfig) ([]protocols.PointValue, error) {
	if len(points) == 0 {
		return nil, nil
	}
	type addrPoint struct {
		addr uint16
		name string
	}
	var addrPoints []addrPoint
	for _, pt := range points {
		addr, err := parseAddress(pt.Address)
		if err != nil {
			return nil, fmt.Errorf("parse address for point %s failed: %v", pt.Address, err)
		}
		addrPoints = append(addrPoints, addrPoint{addr, pt.PointID})
	}
	sort.Slice(addrPoints, func(i, j int) bool { return addrPoints[i].addr < addrPoints[j].addr })

	var results []protocols.PointValue
	n := len(addrPoints)
	i := 0
	const maxCoils = 2000
	for i < n {
		start := i
		end := i
		maxAddr := addrPoints[start].addr
		for j := i + 1; j < n; j++ {
			if addrPoints[j].addr == maxAddr+1 && (addrPoints[j].addr-addrPoints[start].addr) < maxCoils {
				maxAddr = addrPoints[j].addr
				end = j
			} else {
				break
			}
		}
		baseAddr := addrPoints[start].addr
		quantity := maxAddr - baseAddr + 1
		coils, err := m.client.ReadCoils(baseAddr, quantity)
		if err != nil {
			return nil, fmt.Errorf("modbus coils batch read failed: %v", err)
		}
		for k := start; k <= end; k++ {
			offset := addrPoints[k].addr - baseAddr
			val := (coils[offset/8]>>(offset%8))&0x01 == 0x01
			results = append(results, protocols.PointValue{
				PointID:   addrPoints[k].name,
				Value:     val,
				Quality:   "good",
				Timestamp: time.Now().Unix(),
			})
		}
		i = end + 1
	}
	return results, nil
}

// 新增：批量读取离散输入（功能码02）
func (m *ModbusTCP) readDiscreteInputsBatch(deviceID string, points []protocols.PointConfig) ([]protocols.PointValue, error) {
	if len(points) == 0 {
		return nil, nil
	}
	type addrPoint struct {
		addr uint16
		name string
	}
	var addrPoints []addrPoint
	for _, pt := range points {
		addr, err := parseAddress(pt.Address)
		if err != nil {
			return nil, fmt.Errorf("parse address for point %s failed: %v", pt.Address, err)
		}
		addrPoints = append(addrPoints, addrPoint{addr, pt.PointID})
	}
	sort.Slice(addrPoints, func(i, j int) bool { return addrPoints[i].addr < addrPoints[j].addr })

	var results []protocols.PointValue
	n := len(addrPoints)
	i := 0
	const maxInputs = 2000
	for i < n {
		start := i
		end := i
		maxAddr := addrPoints[start].addr
		for j := i + 1; j < n; j++ {
			if addrPoints[j].addr == maxAddr+1 && (addrPoints[j].addr-addrPoints[start].addr) < maxInputs {
				maxAddr = addrPoints[j].addr
				end = j
			} else {
				break
			}
		}
		baseAddr := addrPoints[start].addr
		quantity := maxAddr - baseAddr + 1
		inputs, err := m.client.ReadDiscreteInputs(baseAddr, quantity)
		if err != nil {
			return nil, fmt.Errorf("modbus discrete inputs batch read failed: %v", err)
		}
		for k := start; k <= end; k++ {
			offset := addrPoints[k].addr - baseAddr
			val := (inputs[offset/8]>>(offset%8))&0x01 == 0x01
			results = append(results, protocols.PointValue{
				PointID:   addrPoints[k].name,
				Value:     val,
				Quality:   "good",
				Timestamp: time.Now().Unix(),
			})
		}
		i = end + 1
	}
	return results, nil
}

func (m *ModbusTCP) Write(point string, value interface{}) error {
	addr, err := parseAddress(point)
	if err != nil {
		return err
	}
	switch v := value.(type) {
	case bool:
		var val uint16 = 0x0000
		if v {
			val = 0xFF00
		}
		_, err := m.client.WriteSingleCoil(addr, val)
		return err
	case uint16:
		_, err := m.client.WriteSingleRegister(addr, v)
		return err
	case int:
		_, err := m.client.WriteSingleRegister(addr, uint16(v))
		return err
	default:
		return fmt.Errorf("unsupported value type: %T", value)
	}
}

func (m *ModbusTCP) Close() error {
	if m.handler != nil {
		return m.handler.Close()
	}
	return nil
}

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

func NewModbusTCP() protocols.Protocol {
	return &ModbusTCP{}
}

func init() {
	protocols.Register("modbus_tcp", NewModbusTCP)
}

func (m *ModbusTCP) SetSlave(slaveId byte) {
	if m.handler != nil {
		m.handler.SlaveId = slaveId
	}
}
