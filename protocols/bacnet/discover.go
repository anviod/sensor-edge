package bacnet

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type DeviceInfo struct {
	DeviceID  uint32
	VendorID  uint16
	ModelName string
	IP        string
}

type Device struct {
	ID        string
	IP        string
	DeviceID  uint32
	VendorID  uint16
	Online    bool
	FailCount int
	Points    []string // 可扩展为 config.PointItem
}

var devMap = map[string]*Device{}
var mu sync.RWMutex

// ObjectIdentifier BACnet对象标识
// 可根据实际协议栈结构调整
// 这里只做示例
//
//	type ObjectIdentifier struct {
//		ObjectType string
//		Instance   uint32
//	}
type ObjectIdentifier struct {
	ObjectType string
	Instance   uint32
}

type PointModel struct {
	Name       string `yaml:"name"`
	Address    string `yaml:"address"`
	ObjectType string `yaml:"object_type"`
	Instance   uint32 `yaml:"instance"`
	Property   string `yaml:"property"`
	Type       string `yaml:"type"`
	Unit       string `yaml:"unit,omitempty"`
	Writable   bool   `yaml:"writable"`
}

// ListenIAM 监听 UDP 47808 接收 I-Am 报文并自动注册设备
func ListenIAM(port int, handler func(info *DeviceInfo)) error {
	addr := net.UDPAddr{Port: port, IP: net.IPv4zero}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	buf := make([]byte, 2048)
	for {
		n, remote, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		info, err := parseIAM(buf[:n])
		if err == nil {
			info.IP = remote.IP.String()
			handler(info)
		}
	}
}

// parseIAM 解析 I-Am 报文，提取 DeviceID/VendorID
func parseIAM(data []byte) (*DeviceInfo, error) {
	if len(data) < 14 {
		return nil, fmt.Errorf("invalid length")
	}
	// 简单判断：APDU type 0x10，service 0x00（I-Am）
	if data[1] != 0x10 && data[4] != 0xC4 {
		return nil, fmt.Errorf("not I-Am")
	}
	apdu := data[6:]
	if apdu[0] != 0x10 || apdu[1] != 0x00 {
		return nil, fmt.Errorf("not unconfirmed I-Am")
	}
	deviceID := binary.BigEndian.Uint32(apdu[2:6])
	vendorID := binary.BigEndian.Uint16(apdu[6:8])
	return &DeviceInfo{
		DeviceID:  deviceID,
		VendorID:  vendorID,
		ModelName: "Unknown",
	}, nil
}

// ParseIAm 解析 I-Am 报文，提取 DeviceInfo
func ParseIAm(buf []byte, srcIP net.IP) (*DeviceInfo, error) {
	// 这里只做简单模拟，实际应解析NPDU/APDU结构
	if len(buf) < 14 {
		return nil, fmt.Errorf("I-Am too short")
	}
	// 假设 buf[1] == 0x10 表示 UnconfirmedRequest，buf[4] == 0xC4 表示 I-Am
	if buf[1] != 0x10 || buf[4] != 0xC4 {
		return nil, fmt.Errorf("not I-Am")
	}
	// 模拟解析 DeviceID、VendorID
	deviceID := uint32(buf[8])<<24 | uint32(buf[9])<<16 | uint32(buf[10])<<8 | uint32(buf[11])
	vendorID := uint16(buf[12])<<8 | uint16(buf[13])
	return &DeviceInfo{
		IP:        srcIP.String(),
		DeviceID:  deviceID,
		VendorID:  vendorID,
		ModelName: "Unknown",
	}, nil
}

// RegisterDevice 自动注册新设备并启动采集
func RegisterDevice(info *DeviceInfo) {
	mu.Lock()
	defer mu.Unlock()
	devKey := fmt.Sprintf("bacnet_%d", info.DeviceID)
	if _, exists := devMap[devKey]; exists {
		return
	}
	dev := &Device{
		ID:       devKey,
		IP:       info.IP,
		DeviceID: info.DeviceID,
		VendorID: info.VendorID,
		Online:   true,
	}
	devMap[devKey] = dev
	fmt.Printf("[DISCOVER] New device %s at %s (vendor %d)\n", dev.ID, dev.IP, dev.VendorID)
	go startPolling(dev)
}

// RegisterAndStartPolling 自动注册设备并启动点位模板生成和采集任务
func RegisterAndStartPolling(info *DeviceInfo) {
	key := fmt.Sprintf("bacnet_%d", info.DeviceID)
	mu.Lock()
	if _, ok := devMap[key]; ok {
		mu.Unlock()
		return
	}
	dev := &Device{
		ID:       key,
		IP:       info.IP,
		DeviceID: info.DeviceID,
		VendorID: info.VendorID,
		Online:   true,
	}
	devMap[key] = dev
	mu.Unlock()
	fmt.Printf("[REGISTER] Device %s (%s)\n", key, dev.IP)
	// 自动生成点位模板
	go AutoGeneratePoints(dev)
	// 启动采集任务（可根据实际点位结构完善）
	go startPolling(dev)
}

// startPolling 启动简单采集任务
func startPolling(dev *Device) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := pollOnce(dev); err != nil {
				dev.FailCount++
				if dev.FailCount >= 3 {
					dev.Online = false
					fmt.Printf("[OFFLINE] %s\n", dev.ID)
					return
				}
			} else {
				dev.FailCount = 0
				dev.Online = true
			}
		}
	}
}

func pollOnce(dev *Device) error {
	// TODO: 替换为真实读点
	fmt.Printf("[POLL] Device %s (%s) is alive\n", dev.ID, dev.IP)
	return nil
}

// ReadObjectList 读取objectList属性，返回对象列表（本地模拟/占位实现，无 go-bacnet 依赖）
func ReadObjectList(dev *Device) ([]ObjectIdentifier, error) {
	// 这里直接返回模拟数据，或后续对接你自己的 BACnetClient 实现
	return []ObjectIdentifier{
		{"analogInput", 0},
		{"analogInput", 1},
		{"binaryInput", 2},
	}, nil
}

// AutoGeneratePoints 自动生成点位模板并写入yaml
func AutoGeneratePoints(dev *Device) {
	objectList, err := ReadObjectList(dev)
	if err != nil {
		fmt.Printf("[POINT-GEN] Read objectList failed: %v\n", err)
		return
	}
	points := make([]PointModel, 0)
	for _, obj := range objectList {
		name := fmt.Sprintf("%s_%d", obj.ObjectType, obj.Instance)
		address := fmt.Sprintf("%s:%d", obj.ObjectType, obj.Instance)
		prop := "presentValue"
		pType := GuessValueType(obj.ObjectType)
		points = append(points, PointModel{
			Name:       name,
			Address:    address,
			ObjectType: obj.ObjectType,
			Instance:   obj.Instance,
			Property:   prop,
			Type:       pType,
			Writable:   IsWritable(obj.ObjectType),
		})
	}
	err = WritePointsToFile(dev.ID, points)
	if err != nil {
		fmt.Printf("[POINT-GEN] Write failed: %v\n", err)
	}
}

// WritePointsToFile 写入点位模板到yaml
func WritePointsToFile(devID string, points []PointModel) error {
	filePath := fmt.Sprintf("bacnet_points_%s.yaml", devID)
	data := map[string]interface{}{
		"device_id": devID,
		"protocol":  "bacnet",
		"functions": map[string][]PointModel{},
	}
	funcMap := data["functions"].(map[string][]PointModel)
	for _, pt := range points {
		fn := pt.ObjectType
		funcMap[fn] = append(funcMap[fn], pt)
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, out, 0644)
}

func GuessValueType(objType string) string {
	switch objType {
	case "analogInput", "analogValue":
		return "float"
	case "binaryInput", "binaryValue":
		return "bool"
	default:
		return "float"
	}
}

func IsWritable(objType string) bool {
	switch objType {
	case "analogValue", "binaryValue", "multiStateValue":
		return true
	default:
		return false
	}
}
