package utils

import (
	"encoding/binary"
	"fmt"
	"math"
)

// FormatType 枚举
const (
	FormatInt            = "INT"
	FormatUInt           = "UINT"
	FormatLongABCD       = "Long AB CD"
	FormatLongCDAB       = "Long CD AB"
	FormatLongBADC       = "Long BA DC"
	FormatLongDCBA       = "Long DC BA"
	FormatFloatABCD      = "Float AB CD"
	FormatFloatCDAB      = "Float CD AB"
	FormatFloatBADC      = "Float BA DC"
	FormatFloatDCBA      = "Float DC BA"
	FormatDoubleABCDEFGH = "Double AB CD EF GH"
	FormatDoubleGHEFCDAB = "Double GH EF CD AB"
	FormatDoubleBADCFEHG = "Double BA DC FE HG"
	FormatDoubleHGFEDCBA = "Double HG FE DC BA"
)

// ParseFormat 解析原始字节为对应格式的数值
func ParseFormat(format string, raw []byte) (interface{}, error) {
	switch format {
	case FormatInt:
		if len(raw) < 2 {
			return nil, fmt.Errorf("INT需2字节")
		}
		return int16(binary.BigEndian.Uint16(raw)), nil
	case FormatUInt:
		if len(raw) < 2 {
			return nil, fmt.Errorf("UINT需2字节")
		}
		return binary.BigEndian.Uint16(raw), nil
	case FormatLongABCD:
		if len(raw) < 4 {
			return nil, fmt.Errorf("Long AB CD需4字节")
		}
		return int32(binary.BigEndian.Uint32(raw)), nil
	case FormatLongCDAB:
		if len(raw) < 4 {
			return nil, fmt.Errorf("Long CD AB需4字节")
		}
		return int32(binary.LittleEndian.Uint32([]byte{raw[2], raw[3], raw[0], raw[1]})), nil
	case FormatLongBADC:
		if len(raw) < 4 {
			return nil, fmt.Errorf("Long BA DC需4字节")
		}
		return int32(binary.BigEndian.Uint32([]byte{raw[1], raw[0], raw[3], raw[2]})), nil
	case FormatLongDCBA:
		if len(raw) < 4 {
			return nil, fmt.Errorf("Long DC BA需4字节")
		}
		return int32(binary.LittleEndian.Uint32(raw)), nil
	case FormatFloatABCD:
		if len(raw) < 4 {
			return nil, fmt.Errorf("Float AB CD需4字节")
		}
		bits := binary.BigEndian.Uint32(raw)
		return math.Float32frombits(bits), nil
	case FormatFloatCDAB:
		if len(raw) < 4 {
			return nil, fmt.Errorf("Float CD AB需4字节")
		}
		bits := binary.BigEndian.Uint32([]byte{raw[2], raw[3], raw[0], raw[1]})
		return math.Float32frombits(bits), nil
	case FormatFloatBADC:
		if len(raw) < 4 {
			return nil, fmt.Errorf("Float BA DC需4字节")
		}
		bits := binary.BigEndian.Uint32([]byte{raw[1], raw[0], raw[3], raw[2]})
		return math.Float32frombits(bits), nil
	case FormatFloatDCBA:
		if len(raw) < 4 {
			return nil, fmt.Errorf("Float DC BA需4字节")
		}
		bits := binary.LittleEndian.Uint32(raw)
		return math.Float32frombits(bits), nil
	case FormatDoubleABCDEFGH:
		if len(raw) < 8 {
			return nil, fmt.Errorf("Double AB CD EF GH需8字节")
		}
		bits := binary.BigEndian.Uint64(raw)
		return math.Float64frombits(bits), nil
	case FormatDoubleGHEFCDAB:
		if len(raw) < 8 {
			return nil, fmt.Errorf("Double GH EF CD AB需8字节")
		}
		bits := binary.BigEndian.Uint64([]byte{raw[6], raw[7], raw[4], raw[5], raw[2], raw[3], raw[0], raw[1]})
		return math.Float64frombits(bits), nil
	case FormatDoubleBADCFEHG:
		if len(raw) < 8 {
			return nil, fmt.Errorf("Double BA DC FE HG需8字节")
		}
		bits := binary.BigEndian.Uint64([]byte{raw[1], raw[0], raw[3], raw[2], raw[5], raw[4], raw[7], raw[6]})
		return math.Float64frombits(bits), nil
	case FormatDoubleHGFEDCBA:
		if len(raw) < 8 {
			return nil, fmt.Errorf("Double HG FE DC BA需8字节")
		}
		bits := binary.LittleEndian.Uint64(raw)
		return math.Float64frombits(bits), nil
	default:
		return nil, fmt.Errorf("未知format: %s", format)
	}
}
