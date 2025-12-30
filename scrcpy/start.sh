#!/bin/bash
# run.sh - 启动服务

PORT=${1:-8888}

echo "启动 scrcpy-go 服务..."

# 设置ADB端口转发
adb forward tcp:$PORT tcp:$PORT

# 在设备上启动服务
adb shell /data/local/tmp/scrcpy-go \
    --port $PORT \
    --bit-rate 8M \
    --max-fps 60 \
    --max-width 1280 \
    --stay-awake \
    --verbose &

# 等待服务启动
sleep 2

echo "服务已启动，客户端可以连接 localhost:$PORT"
echo "按 Ctrl+C 停止服务"

# 保持脚本运行
wait