# PCCS — Phone Collector Cookie Sync

从手机 App 的本地存储里读取凭据，**指纹变化时**才推送到 NAS。

> 设计目标：把「人工扫码续期」变成「App 自己续、我们只管读」。

## 原理

```
京东 App 正常使用
   ↓ 服务端 Set-Cookie，App 自己续期
/data/data/com.jingdong.app.mall/app_webview/Default/Cookies   (SQLite)
   ↓ pccs 只读打开（mode=ro，SQLite 自动合并 -wal）
计算字段指纹 → 与上次比对
   ↓ 变了才推（没变零流量）
POST http://<nas>:18080/ingest  {app:"jd", pt_key, pt_pin}
   ↓
NAS 落盘 mobile_cookie.json (600) + QQ 推送
```

**关键点：我们从不登录、从不发请求给京东**，只读取 App 已经续好的结果。

## 特性

- **零输入**：不需要账号密码、不需要扫码
- **零多余流量**：指纹不变不推送（默认 1 小时检查一次）
- **只读**：`mode=ro&busy_timeout(3000)`，不锁 App 的库
- **纯 Go + 无 CGO**：`modernc.org/sqlite`，交叉编译 arm64 无痛
- **可扩展**：`internal/source` 接口化，加新 App 只需实现一个 Source

## 构建

推送到 GitHub 即自动编译（`.github/workflows/build.yml`）：

```bash
git push origin main
# → Actions 产出 dist/pccs (linux/arm64) + dist/pccs-module-vX.Y.Z.zip
```

本地构建（需 Go 1.22+）：

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath \
  -ldflags "-s -w" -o dist/pccs ./cmd/pccs
```

## 用法

```bash
# 常驻（模块里由 service.sh 启动）
pccs -nas http://192.168.31.208:18080/ingest -interval 1h

# 只跑一次（调试）
pccs -once -v

# 指定京东库路径
pccs -jd-db /data/data/com.jingdong.app.mall/app_webview/Default/Cookies
```

| 参数 | 默认 | 说明 |
|---|---|---|
| `-nas` | `http://192.168.31.208:18080/ingest` | NAS 上报地址 |
| `-interval` | `1h` | 轮询间隔 |
| `-state` | `/data/adb/pccs/state.json` | 记录上次指纹 |
| `-jd-db` | App 默认路径 | 京东 Cookie 库 |
| `-once` | false | 只跑一次 |
| `-v` | false | 详细日志 |

## 安装（KSU 模块）

1. 下载 `pccs-module-*.zip`
2. KSU 管理器 → 模块 → 从本地安装 → 重启
3. 日志：`/data/adb/modules/pccs/pccs.log`

## 服务端协议

```http
POST /ingest
Content-Type: application/json

{"app":"jd","pt_key":"...","pt_pin":"..."}

200 {"ok":true,"changed":true,"fp":"097ed606eac7aeb9"}
200 {"ok":true,"changed":false,"fp":"097ed606eac7aeb9"}   # 指纹未变
403 {"ok":false,"msg":"forbidden"}                          # 非白名单来源
```

服务端按 **IP 白名单** 鉴权（内网），见 NAS 上的 `collector_ingest.py`。

## 目录结构

```
cmd/pccs/            主程序（轮询 + 指纹比对 + 上报）
internal/source/     数据源（接口 + 京东实现 + 指纹）
internal/sink/       HTTP 上报
module/              KSU 模块脚本与 module.prop
.github/workflows/   交叉编译 + 打包 + Release
```

## License

MIT