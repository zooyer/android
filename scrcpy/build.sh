#!/bin/bash
# build.sh - 构建Android版本

set -e

echo "构建 scrcpy-go for Android..."

# 设置环境变量
export GOOS=android
export GOARCH=arm
export GOARM=7
export CGO_ENABLED=0  # 纯Go模式

# 清理
echo "清理旧文件..."
rm -f scrcpy-go

# 编译
echo "编译中..."
go build -ldflags="-s -w" -o scrcpy-go main.go

# 检查文件
if [ -f "scrcpy-go" ]; then
    echo "构建成功!"
    ls -lh scrcpy-go
else
    echo "构建失败!"
    exit 1
fi

# 推送到设备
echo "推送到Android设备..."
adb push scrcpy-go /data/local/tmp/
adb shell chmod 755 /data/local/tmp/scrcpy-go

echo "完成! 使用以下命令运行:"
echo "adb shell /data/local/tmp/scrcpy-go --port 8888"