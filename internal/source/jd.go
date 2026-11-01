package source

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

const defaultJDPath = "/data/data/com.jingdong.app.mall/app_webview/Default/Cookies"

// JD 读取京东 App WebView 的 Cookie 库，取 pt_key / pt_pin。
type JD struct {
	path string
}

// NewJD 创建京东数据源；path 为空则用默认路径。
func NewJD(path string) *JD {
	if path == "" {
		path = defaultJDPath
	}
	return &JD{path: path}
}

func (j *JD) Name() string        { return "jd" }
func (j *JD) Description() string { return "京东 App WebView Cookie (pt_key/pt_pin)" }
func (j *JD) Path() string        { return j.path }

// Fetch 以只读方式打开 Cookie 库。
// mode=ro 让 SQLite 自行处理 -wal/-shm（无需手动合并），
// busy_timeout 避免 App 正在写时立刻报 database is locked。
func (j *JD) Fetch(ctx context.Context) (map[string]string, bool, error) {
	dsn := fmt.Sprintf("file:%s?mode=ro&_pragma=busy_timeout(3000)", j.path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, false, fmt.Errorf("open: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	if err := db.PingContext(ctx); err != nil {
		return nil, false, fmt.Errorf("ping: %w", err)
	}

	q := `SELECT name, value FROM cookies WHERE host_key = ? AND name IN (?, ?)`
	rows, err := db.QueryContext(ctx, q, ".jd.com", "pt_key", "pt_pin")
	if err != nil {
		return nil, false, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	fields := map[string]string{}
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, false, fmt.Errorf("scan: %w", err)
		}
		fields[name] = value
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("rows: %w", err)
	}

	if fields["pt_key"] == "" || fields["pt_pin"] == "" {
		// 库存在但未登录，属"当前不可用"，不是错误。
		return nil, false, nil
	}
	return fields, true, nil
}