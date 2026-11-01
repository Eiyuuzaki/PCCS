#!/system/bin/sh
# PCCS uninstall：只停自己的进程，只删自己的运行目录

MODDIR=${0%/*}
BIN="$MODDIR/bin/pccs"

# 精确匹配本模块二进制，避免误杀
[ -n "$BIN" ] && pkill -f "^$BIN" 2>/dev/null

# 只删本程序的状态目录
rm -rf /data/adb/pccs 2>/dev/null

exit 0