// Package sink 负责把采集结果推送到 NAS。
package sink

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client 推送客户端。
type Client struct {
	URL     string
	Timeout time.Duration
	hc      *http.Client
}

// New 创建客户端；timeout<=0 时默认 15s。
func New(url string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &Client{URL: url, Timeout: timeout, hc: &http.Client{Timeout: timeout}}
}

// Result 服务端响应。
type Result struct {
	OK      bool   `json:"ok"`
	Changed bool   `json:"changed"`
	FP      string `json:"fp"`
	Msg     string `json:"msg"`
}

// Push 发送一次上报。
func (c *Client) Push(ctx context.Context, app string, fields map[string]string) (*Result, error) {
	body := map[string]any{"app": app}
	for k, v := range fields {
		body[k] = v
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(data))
	}
	var r Result
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &r, nil
}