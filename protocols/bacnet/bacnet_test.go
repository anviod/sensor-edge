package bacnet

import (
	"testing"
)

func getTestConfig() map[string]interface{} {
	return map[string]interface{}{
		"device_id": "test_bacnet_1",
		"points": []interface{}{
			map[string]interface{}{
				"name":       "temp",
				"address":    "1",
				"type":       "float",
				"format":     "Float AB CD",
				"unit":       "℃",
				"init_value": 25.5,
			},
			map[string]interface{}{
				"name":       "alarm",
				"address":    "2",
				"type":       "bool",
				"format":     "BOOL",
				"unit":       "",
				"init_value": false,
			},
			map[string]interface{}{
				"name":       "count",
				"address":    "3",
				"type":       "int",
				"format":     "INT",
				"unit":       "",
				"init_value": 10,
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
	if len(model) != 3 {
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
	if len(vals) != 3 {
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
	// float
	err := c.Write("temp", 30.1)
	if err != nil {
		t.Errorf("Write float failed: %v", err)
	}
	// bool
	err = c.Write("alarm", true)
	if err != nil {
		t.Errorf("Write bool failed: %v", err)
	}
	// int
	err = c.Write("count", 99)
	if err != nil {
		t.Errorf("Write int failed: %v", err)
	}
	// 读回
	vals, _ := c.Read("test_bacnet_1")
	found := 0
	for _, v := range vals {
		if v.PointID == "temp" && v.Value != 30.1 {
			t.Errorf("Write/Read float mismatch: %v", v.Value)
		}
		if v.PointID == "alarm" && v.Value != true {
			t.Errorf("Write/Read bool mismatch: %v", v.Value)
		}
		if v.PointID == "count" && v.Value != 99 {
			t.Errorf("Write/Read int mismatch: %v", v.Value)
		}
		found++
	}
	if found != 3 {
		t.Errorf("Read back count error: %d", found)
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
