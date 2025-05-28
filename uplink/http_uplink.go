package uplink

import (
	"bytes"
	"net/http"
)

type HttpUplink struct {
	url     string
	method  string
	headers map[string]string
	name    string
}

func (h *HttpUplink) Send(data []byte) error {
	req, _ := http.NewRequest(h.method, h.url, bytes.NewBuffer(data))
	for k, v := range h.headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (h *HttpUplink) Name() string { return h.name }
func (h *HttpUplink) Type() string { return "http" }
