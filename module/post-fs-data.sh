#!/system/bin/sh
# PCCS post-fs-data
# 安全原则：此阶段极早，任何失败都必须 exit 0，绝不阻断开机。
# 不做：挂载、覆盖系统目录、长时间阻塞、依赖 date/mkdir 成功。

MODDIR=${0%/*}

# 只做一件事：确保模块目录可写标记存在（失败也无所谓）
mkdir -p "$MODDIR" 2>/dev/null

# 显式成功退出，避免 KSU/Magisk 把非零返回当成故障
exit 0