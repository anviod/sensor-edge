package http

import (
	"bytes"

	"fmt"
	"io/ioutil"
	"net/http"
	"time"

)

type HTTPClientUplink struct {
	url     string
	method  string
	headers map[string]string
}

func NewHTTPClientUplink(config map[string]interface{}) (*HTTPClientUplink, error) {
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

	return &HTTPClientUplink{
		url:     url,
		method:  method,
		headers: headers,
	}, nil
}

func (c *HTTPClientUplink) Send(data []byte) error {
	req, err := http.NewRequest(c.method, c.url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	for k, v := range c.headers {
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
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("http send failed with code %d: %s", resp.StatusCode, body)
	}

	return nil
}

func (c *HTTPClientUplink) Name() string {
	return "http"
}
func (c *HTTPClientUplink) Type() string {
	return "POST"
}