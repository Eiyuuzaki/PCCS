// Package source 定义数据源：从手机本地读取某个 App 的凭据。
package source

import "context"

// Item 一次采集到的结果。
type Item struct {
	App    string            `json:"app"`
	Fields map[string]string `json:"-"`
	FP     string            `json:"fp,omitempty"`
}

// Source 可插拔的数据源。
type Source interface {
	// Name 用于日志与上报 app 字段（如 "jd"）。
	Name() string
	// Description 人类可读说明。
	Description() string
	// Path 源数据绝对路径。
	Path() string
	// Fetch 读取一次；ok=false 表示当前不可用（App 未登录等），不算错误。
	Fetch(ctx context.Context) (fields map[string]string, ok bool, err error)
}