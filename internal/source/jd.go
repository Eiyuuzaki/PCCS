package source

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

const defaultJDPath = "/data/data/com.jingdong.app.mall/app_webview/Default/Cookies"

// JD 读取京东 App WebView 的 Cookie 库。
//
// 2026-11-09 修复「跛脚身份」：旧版只取 pt_key/pt_pin 两条，把 App 的整套
// 设备指纹 cookie（shshshfp*、unionwsws/eid、sdtoken、unpl、mba_*、qid_*、
// pre_*、__jd* 等 44 条）全部丢弃。需要 h5st+eid 的域（plus_business /
// SecKill2020 / activities_platform）据此把我们会话判为高危，一发写请求就
// 触发风控；不需要指纹的域（wegame-hub/jx_h5_babel）则从不出事。
// 现改为整域收取，去留（合并/清理）交给 NAS 侧 jd_sync_cookie.py 决定。
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

func (j *JD) Name() string { return "jd" }
func (j *JD) Description() string {
	return "京东 App WebView Cookie（jd.com 全域，含设备指纹）"
}
func (j *JD) Path() string { return j.path }

// Fetch 以只读方式打开 Cookie 库，取 jd.com 及其子域的全部 cookie。
// mode=ro 让 SQLite 自行处理 -wal/-shm（无需手动合并），
// busy_timeout 避免 App 正在写时立刻报 database is locked。
//
// 同名不同值的跨域 cookie（如 pt_key 同时挂在 .jd.com 与 .healthjd.com）：
// 首个出现的用裸 name，冲突的用 "name@host" 消歧——NAS 侧按同样规则还原。
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

	q := `SELECT host_key, name, value FROM cookies
		WHERE host_key = 'jd.com' OR host_key LIKE '%.jd.com'`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, false, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	fields := map[string]string{}
	for rows.Next() {
		var host, name, value string
		if err := rows.Scan(&host, &name, &value); err != nil {
			return nil, false, fmt.Errorf("scan: %w", err)
		}
		if name == "" {
			continue
		}
		if old, dup := fields[name]; dup && old != value {
			fields[name+"@"+host] = value
		} else {
			fields[name] = value
		}
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("rows: %w", err)
	}

	if fields["pt_key"] == "" {
		// 库存在但未登录，属"当前不可用"，不是错误。
		return nil, false, nil
	}
	return fields, true, nil
}
