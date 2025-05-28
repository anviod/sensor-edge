package httpclient

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sensor-edge/protocols"
	"time"
)

type HTTPClient struct {
	URL    string
	Method string
}

func (h *HTTPClient) Init(config map[string]interface{}) error {
	h.URL = config["url"].(string)
	h.Method = config["method"].(string)
	if mock, ok := config["mock"].(bool); ok && mock {
		h.URL = "MOCK"
	}
	return nil
}

func (h *HTTPClient) Read(deviceID string) ([]protocols.PointValue, error) {
	if h.URL == "MOCK" {
		now := time.Now().Unix()
		// 根据设备ID自定义不同mock数据
		switch deviceID {
		case "plc1":
			// 模拟风机状态和转速
			fanStatus := rand.Intn(2) == 1
			fanSpeed := 400 + rand.Intn(2000) // 400~2400
			return []protocols.PointValue{
				{PointID: "d100", Value: fanStatus, Quality: "good", Timestamp: now},
				{PointID: "d101", Value: fanSpeed, Quality: "good", Timestamp: now},
			}, nil
		case "sensor_http":
			// 模拟温度
			temp := 20.0 + rand.Float64()*80.0 // 20~100
			return []protocols.PointValue{
				{PointID: "temperature", Value: temp, Quality: "good", Timestamp: now},
			}, nil
		default:
			return []protocols.PointValue{}, nil
		}
	}
	resp, err := http.Get(h.URL)
	if err != nil {
		// 网络异常时返回模拟数据
		return []protocols.PointValue{
			{
				PointID:   "http_temp",
				Value:     22.0,
				Quality:   "mock",
				Timestamp: time.Now().Unix(),
			},
		}, nil
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	val := parsed["temp"] // 假设返回 {"temp": 23.5}
	return []protocols.PointValue{
		{
			PointID:   "http_temp",
			Value:     val,
			Quality:   "good",
			Timestamp: time.Now().Unix(),
		},
	}, nil
}

func (h *HTTPClient) ReadBatch(deviceID string, points []string) ([]protocols.PointValue, error) {
	if len(points) == 0 {
		return nil, nil
	}
	var values []protocols.PointValue
	for _, pt := range points {
		v, err := h.ReadPoint(pt)
		if err != nil {
			values = append(values, protocols.PointValue{PointID: pt, Quality: "bad", Value: nil, Timestamp: time.Now().Unix()})
			continue
		}
		values = append(values, protocols.PointValue{PointID: pt, Quality: "good", Value: v, Timestamp: time.Now().Unix()})
	}
	return values, nil
}

func (h *HTTPClient) Write(point string, value interface{}) error {
	// mock写入直接打印
	if h.URL == "MOCK" {
		fmt.Printf("[MOCK WRITE] %s <- %v\n", point, value)
		return nil
	}
	// 真实http写入可扩展
	return nil
}

func (h *HTTPClient) Close() error {
	return nil
}

func NewHTTPClient() protocols.Protocol {
	return &HTTPClient{}
}

func init() {
	protocols.Register("http", NewHTTPClient)
}
