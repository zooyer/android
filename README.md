

##### 模拟打开应用
monkey -p com.iflytek.xiri 1

##### 获取当前活动页面包名
dumpsys activity activities | grep mResumedActivity | awk -F '/' '{print $1}' | awk '{print $NF}'

##### 查看禁用的包
pm list packages -d

##### 获取包默认 Activity
cmd package resolve-activity --brief tv.icntv.vendor

##### 启动 Activity
am start tv.icntv.vendor/.main

##### 启动当贝桌面（直接包名就行）
am start com.dangbei.tvlauncher

##### 查看遥控器设备
cat /proc/bus/input/devices

##### 获取所有输入法
ime -a