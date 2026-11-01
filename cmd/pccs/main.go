// pccs — Phone Collector Cookie Sync
// 从手机 App 本地存储读取凭据，指纹变化时推送到 NAS。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Eiyuuzaki/PCCS/internal/sink"
	"github.com/Eiyuuzaki/PCCS/internal/source"
)

const version = "0.1.0"

type state struct {
	LastFP map[string]string `json:"last_fp"`
}

func main() {
	var (
		nasURL  = flag.String("nas", "http://192.168.31.208:18080/ingest", "NAS 上报地址")
		interval = flag.Duration("interval", time.Hour, "轮询间隔")
		stateP  = flag.String("state", "/data/adb/pccs/state.json", "状态文件（记录上次指纹）")
		jdPath  = flag.String("jd-db", "", "京东 Cookie 库路径（默认 App 默认路径）")
		once    = flag.Bool("once", false, "只跑一次后退出")
		verbose = flag.Bool("v", false, "详细日志")
		doVer   = flag.Bool("version", false, "打印版本")
	)
	flag.Parse()

	if *doVer {
		fmt.Println("pccs", version)
		return
	}

	log.SetFlags(log.LstdFlags)
	srcs := []source.Source{source.NewJD(*jdPath)}
	cl := sink.New(*nasURL, 15*time.Second)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("pccs %s 启动 | 间隔=%s | NAS=%s", version, *interval, *nasURL)
	for _, s := range srcs {
		log.Printf("  数据源 %-4s %s", s.Name(), s.Description())
		log.Printf("    路径 %s", s.Path())
	}

	// 启动即检查一次，之后按间隔轮询。
	runOnce := func() {
		for _, s := range srcs {
			cycle(ctx, s, cl, *stateP, *verbose)
		}
	}
	runOnce()
	if *once {
		return
	}

	t := time.NewTicker(*interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Printf("收到退出信号，结束")
			return
		case <-t.C:
			runOnce()
		}
	}
}

// cycle 单次采集 + 对比 + 上报。
func cycle(ctx context.Context, s source.Source, cl *sink.Client, statePath string, verbose bool) {
	fields, ok, err := s.Fetch(ctx)
	if err != nil {
		log.Printf("[%s] 读取失败: %v", s.Name(), err)
		// 上报失败不做重试风暴，等下一轮。
		return
	}
	if !ok {
		log.Printf("[%s] 暂无可用数据（App 未登录？），跳过", s.Name())
		return
	}

	fp := source.Fingerprint(fields)
	st := loadState(statePath)
	if st.LastFP[s.Name()] == fp {
		if verbose {
			log.Printf("[%s] 指纹未变 (%s)，跳过", s.Name(), fp)
		}
		return
	}

	log.Printf("[%s] 指纹变化 %s -> %s，上报中…", s.Name(), orNone(st.LastFP[s.Name()]), fp)
	res, err := cl.Push(ctx, s.Name(), fields)
	if err != nil {
		log.Printf("[%s] 上报失败: %v（下轮重试）", s.Name(), err)
		return
	}
	log.Printf("[%s] 上报成功 changed=%v fp=%s", s.Name(), res.Changed, res.FP)

	st.LastFP[s.Name()] = fp
	saveState(statePath, st)
}

func orNone(s string) string {
	if s == "" {
		return "(无)"
	}
	return s
}

func loadState(p string) state {
	st := state{LastFP: map[string]string{}}
	b, err := os.ReadFile(p)
	if err != nil {
		return st
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return state{LastFP: map[string]string{}}
	}
	if st.LastFP == nil {
		st.LastFP = map[string]string{}
	}
	return st
}

func saveState(p string, st state) {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		log.Printf("建目录失败: %v", err)
		return
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return
	}
	if err := os.WriteFile(p, b, 0o600); err != nil {
		log.Printf("写状态失败: %v", err)
	}
}