package main

import (
	"fmt"
	"sensor-edge/protocols/modbus"
	"sensor-edge/uplink"
)

func main() {
	// core.Run() // Removed because core.Run is undefined

	//WebApp()

	// Modbus 读写示例
	modbusCfg := map[string]interface{}{
		"ip":       "127.0.0.1",
		"port":     502,
		"slave_id": 1,
	}
	client := &modbus.ModbusTCP{}
	if err := client.Init(modbusCfg); err != nil {
		fmt.Println("Modbus Init error:", err)
		return
	}
	defer client.Close()

	// 读取寄存器
	values, err := client.Read("1") // 假设读取40001寄存器
	if err != nil {
		fmt.Println("Modbus Read error:", err)
	} else {
		fmt.Println("Modbus Read result:", values)
	}

	// 写入寄存器（如40001写入123）
	err = client.Write("40002", uint16(123))
	if err != nil {
		fmt.Println("Modbus Write error:", err)
	} else {
		fmt.Println("Modbus Write success")
	}

	// 上报（假设用MQTT uplink，实际可用UplinkManager）
	factory := uplink.UplinkFactory{}
	uplinkInst := factory.NewUplink("mqtt")
	if uplinkInst != nil {
		uplinkInst.Send([]byte(`{"device_id":"plc1","data":{"temp1":12.3}}`))
	}
}
