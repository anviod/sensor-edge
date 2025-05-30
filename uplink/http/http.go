package http

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HttpUplink struct {
	URL     string
	Method  string
	Headers map[string]string
}

func NewHttpUplink(config map[string]interface{}) (*HttpUplink, error) {
	url, ok := config["url"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid url")
	}

	method := "POST"
	if m, ok := config["method"].(string); ok {
		method = m
	}

	headers := make(map[string]string)
	if h, ok := config["headers"].(map[string]interface{}); ok {
		for k, v := range h {
			headers[k] = fmt.Sprintf("%v", v)
		}
	}

	return &HttpUplink{
		URL:     url,
		Method:  method,
		Headers: headers,
	}, nil
}

func (c *HttpUplink) Send(data []byte) error {
	req, err := http.NewRequest(c.Method, c.URL, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http send failed with code %d: %s", resp.StatusCode, body)
	}

	return nil
}

func (c *HttpUplink) Name() string {
	return "http"
}
func (c *HttpUplink) Type() string {
	return "POST"
}
