package bacnet

import (
	"testing"
)

func TestDiscoverDevices(t *testing.T) {
	client := &BacnetClient{}
	devices, err := client.DiscoverDevices()
	if err != nil {
		t.Fatalf("DiscoverDevices failed: %v", err)
	}
	if len(devices) != 2 {
		t.Errorf("expect 2 devices, got %d", len(devices))
	}
	if devices[0].DeviceID != "2228316" || devices[1].DeviceID != "2228317" {
		t.Errorf("device id mismatch: %+v", devices)
	}
}
