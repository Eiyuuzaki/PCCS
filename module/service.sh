#!/system/bin/sh
# PCCS service（late_start，系统基本就绪后执行）
# 安全原则：自带超时、不阻塞开机流程、失败只记录不退出。

MODDIR=${0%/*}
LOG="$MODDIR/pccs.log"
STATE="/data/adb/pccs/state.json"
BIN="$MODDIR/bin/pccs"

log() { echo "$(date "+%Y-%m-%d %H:%M:%S") $*" >> "$LOG" 2>/dev/null; }

log "service.sh 进入"

# 等开机完成，最多等 180 秒（超时也继续，不卡死）
i=0
while [ "$i" -lt 90 ]; do
  if [ "$(getprop sys.boot_completed 2>/dev/null)" = "1" ]; then break; fi
  sleep 2
  i=$((i+1))
done

# 再等 20 秒，让网络栈稳定
sleep 20

# 二进制缺失则直接退出（不会影响其他模块）
if [ ! -f "$BIN" ]; then
  log "缺少 $BIN，退出"
  exit 0
fi
chmod 755 "$BIN" 2>/dev/null

# 单实例守卫：只匹配本模块的绝对路径，避免误杀
if pgrep -f "^$BIN" >/dev/null 2>&1; then
  log "已有实例在跑，跳过"
  exit 0
fi

mkdir -p /data/adb/pccs 2>/dev/null
chmod 700 /data/adb/pccs 2>/dev/null

log "启动 pccs"
nohup "$BIN" \
  -nas "http://192.168.31.208:18080/ingest" \
  -interval 1h \
  -state "$STATE" \
  >> "$LOG" 2>&1 &

log "已拉起 pid=$!"
exit 0