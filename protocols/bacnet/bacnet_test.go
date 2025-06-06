package bacnet

import (
	"testing"
)

func getTestConfig() map[string]interface{} {
	return map[string]interface{}{
		"device_id": "test_bacnet_1",
		"points": []interface{}{
			map[string]interface{}{
				"name":                "temp",
				"address":             "1",
				"type":                "float",
				"format":              "Float AB CD",
				"unit":                "℃",
				"init_value":          25.5,
				"property_value_type": "REAL",
				"writable":            true,
			},
			map[string]interface{}{
				"name":                "alarm",
				"address":             "2",
				"type":                "bool",
				"format":              "BOOL",
				"unit":                "",
				"init_value":          false,
				"property_value_type": "BOOLEAN",
				"writable":            true,
			},
			map[string]interface{}{
				"name":                "count",
				"address":             "3",
				"type":                "int",
				"format":              "INT",
				"unit":                "",
				"init_value":          10,
				"property_value_type": "INTEGER",
				"writable":            true,
			},
			map[string]interface{}{
				"name":                "enum",
				"address":             "4",
				"type":                "int",
				"format":              "INT",
				"unit":                "",
				"init_value":          1,
				"property_value_type": "ENUMERATED",
				"writable":            true,
			},
			map[string]interface{}{
				"name":                "str",
				"address":             "5",
				"type":                "string",
				"format":              "CHAR",
				"unit":                "",
				"init_value":          "abc",
				"property_value_type": "CHARACTERSTRING",
				"writable":            true,
			},
			map[string]interface{}{
				"name":                "octet",
				"address":             "6",
				"type":                "bytes",
				"format":              "OCTET",
				"unit":                "",
				"init_value":          []byte{1, 2, 3},
				"property_value_type": "OCTETSTRING",
				"writable":            true,
			},
			map[string]interface{}{
				"name":                "bitstr",
				"address":             "7",
				"type":                "bits",
				"format":              "BITSTR",
				"unit":                "",
				"init_value":          []bool{true, false},
				"property_value_type": "BITSTRING",
				"writable":            true,
			},
			map[string]interface{}{
				"name":                "objid",
				"address":             "8",
				"type":                "objectid",
				"format":              "OBJID",
				"unit":                "",
				"init_value":          ObjectID{Type: AnalogInput, Instance: 1},
				"property_value_type": "OBJECTID",
				"writable":            true,
			},
		},
	}
}

func TestInitAndGetPointModel(t *testing.T) {
	c := &BacnetClient{}
	err := c.Init(getTestConfig())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	model := c.GetPointModel()
	if len(model) != 8 {
		t.Errorf("point model size error: %d", len(model))
	}
}

func TestRead(t *testing.T) {
	c := &BacnetClient{}
	_ = c.Init(getTestConfig())
	vals, err := c.Read("test_bacnet_1")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if len(vals) != 8 {
		t.Errorf("Read point count error: %d", len(vals))
	}
}

func TestReadBatch(t *testing.T) {
	c := &BacnetClient{}
	_ = c.Init(getTestConfig())
	vals, err := c.ReadBatch("test_bacnet_1", "analogInput", []string{"temp", "alarm"})
	if err != nil {
		t.Fatalf("ReadBatch failed: %v", err)
	}
	if len(vals) != 2 {
		t.Errorf("ReadBatch point count error: %d", len(vals))
	}
}

func TestWriteAndReadBack(t *testing.T) {
	c := &BacnetClient{}
	_ = c.Init(getTestConfig())
	_ = c.Write("temp", 30.5)
	_ = c.Write("alarm", true)
	_ = c.Write("count", 99)
	_ = c.Write("enum", 2)
	_ = c.Write("str", "hello")
	_ = c.Write("octet", []byte{9, 8, 7})
	_ = c.Write("bitstr", []bool{false, true})
	_ = c.Write("objid", ObjectID{Type: AnalogInput, Instance: 2})
	vals, err := c.Read("test_bacnet_1")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if len(vals) != 8 {
		t.Errorf("Read back count error: %d", len(vals))
	}
}

func TestWriteTypeError(t *testing.T) {
	c := &BacnetClient{}
	_ = c.Init(getTestConfig())
	if err := c.Write("temp", true); err == nil {
		t.Error("Write type error not detected (float)")
	}
	if err := c.Write("alarm", 123); err == nil {
		t.Error("Write type error not detected (bool)")
	}
	if err := c.Write("count", false); err == nil {
		t.Error("Write type error not detected (int)")
	}
}

func TestWriteNotFound(t *testing.T) {
	c := &BacnetClient{}
	_ = c.Init(getTestConfig())
	if err := c.Write("not_exist", 1); err == nil {
		t.Error("Write not found error not detected")
	}
}

func TestCloseAndReconnect(t *testing.T) {
	c := &BacnetClient{}
	_ = c.Init(getTestConfig())
	_ = c.Close()
	if _, err := c.Read("test_bacnet_1"); err == nil {
		t.Error("Read after close should fail")
	}
	_ = c.Reconnect()
	if _, err := c.Read("test_bacnet_1"); err != nil {
		t.Error("Read after reconnect should succeed")
	}
}

func TestWriteAllTypes(t *testing.T) {
	c := &BacnetClient{}
	_ = c.Init(getTestConfig())
	// REAL
	if err := c.Write("temp", 12.34); err != nil {
		t.Errorf("Write REAL failed: %v", err)
	}
	// BOOLEAN
	if err := c.Write("alarm", true); err != nil {
		t.Errorf("Write BOOLEAN failed: %v", err)
	}
	// INTEGER
	if err := c.Write("count", 123); err != nil {
		t.Errorf("Write INTEGER failed: %v", err)
	}
	// ENUMERATED
	if err := c.Write("enum", 2); err != nil {
		t.Errorf("Write ENUMERATED failed: %v", err)
	}
	// CHARACTERSTRING
	if err := c.Write("str", "hello"); err != nil {
		t.Errorf("Write CHARACTERSTRING failed: %v", err)
	}
	// OCTETSTRING
	if err := c.Write("octet", []byte{4, 5, 6}); err != nil {
		t.Errorf("Write OCTETSTRING failed: %v", err)
	}
	// BITSTRING
	if err := c.Write("bitstr", []bool{false, true}); err != nil {
		t.Errorf("Write BITSTRING failed: %v", err)
	}
	// OBJECTID
	if err := c.Write("objid", ObjectID{Type: AnalogInput, Instance: 2}); err != nil {
		t.Errorf("Write OBJECTID failed: %v", err)
	}
}

func TestWriteTypeErrorAllTypes(t *testing.T) {
	c := &BacnetClient{}
	_ = c.Init(getTestConfig())
	// REAL
	if err := c.Write("temp", int(123)); err == nil {
		t.Error("Write REAL type error not detected")
	}
	// BOOLEAN
	if err := c.Write("alarm", 123); err == nil {
		t.Error("Write BOOLEAN type error not detected")
	}
	// INTEGER
	if err := c.Write("count", false); err == nil {
		t.Error("Write INTEGER type error not detected")
	}
	// ENUMERATED
	if err := c.Write("enum", "notint"); err == nil {
		t.Error("Write ENUMERATED type error not detected")
	}
	// CHARACTERSTRING
	if err := c.Write("str", 123); err == nil {
		t.Error("Write CHARACTERSTRING type error not detected")
	}
	// OCTETSTRING
	if err := c.Write("octet", "notbytes"); err == nil {
		t.Error("Write OCTETSTRING type error not detected")
	}
	// BITSTRING
	if err := c.Write("bitstr", "notbits"); err == nil {
		t.Error("Write BITSTRING type error not detected")
	}
	// OBJECTID
	if err := c.Write("objid", 123); err == nil {
		t.Error("Write OBJECTID type error not detected")
	}
}
