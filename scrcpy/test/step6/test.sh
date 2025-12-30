#!/bin/bash
# diagnose_fb.sh

echo "=== 帧缓冲区诊断 ==="

# 1. 检查设备文件
echo "1. 设备文件检查:"
adb shell "ls -l /dev/graphics/fb* /dev/fb* 2>/dev/null"

# 2. 检查系统信息
echo -e "\n2. 系统信息:"
adb shell "cat /proc/fb 2>/dev/null || echo '无/proc/fb文件'"

# 3. 检查dmesg中的fb信息
echo -e "\n3. 内核消息中的fb信息:"
adb shell "dmesg | grep -i 'fb\|frame' | busybox tail -20"

# 4. 检查sysfs中的fb信息
echo -e "\n4. SysFS中的fb信息:"
adb shell "ls -l /sys/class/graphics/fb0/ 2>/dev/null && cat /sys/class/graphics/fb0/* 2>/dev/null"

# 5. 尝试直接读取
echo -e "\n5. 尝试读取设备:"
adb shell "dd if=/dev/graphics/fb0 of=/dev/null bs=1024 count=1 2>&1 | busybox head -5"

# 6. 检查是否有surfaceflinger正在使用
echo -e "\n6. 检查显示服务:"
adb shell "ps | grep -i surface"