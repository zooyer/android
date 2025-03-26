/**
 * @Author: zzy
 * @Email: zhangzhongyuan@didiglobal.com
 * @Description:
 * @File: main.go
 * @Package: liliang
 * @Version: 1.0.0
 * @Date: 2022/11/17 12:21
 */

package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/zooyer/miskit/dns"
)

// M310H
// 型号：新魔百和M310H
// 破解方式1：连接绑定宽带，进入IPTV后，切换成wifi连接到PC。PC抓包分析请求地址和端口和URI，并设置地址后启动hook下载文件服务
// 破解方式2：TTL开启adbd
func M310H() {
	engine := gin.Default()
	engine.StaticFile("/upload/apk/app_hjkhtvmanagerRelease.apk", "/Users/zhangzhongyuan/Downloads/dangbeimarket_4.4.0_288_yunji.apk")
	engine.Run(":18082")
}

// CM311
// 破解方式：
// 默认dns是本机，先启动本程序，然后开启网络共享（默认也会启动dns服务，先启动本程序的dns让盒子解析缓存）
// mac设置，共享 -> 互联网共享 -> 点击? -> 设置来源WIFI，共享给USB LAN -> 开启
func CM311() {
	engine := gin.Default()
	// 让盒子的dns缓存上该域名
	// update.bja.bcs.ottcn.com
	if err := dns.HookHostsByLocal("update.bja.bcs.ottcn.com"); err != nil {
		fmt.Println(err)
	}
	engine.StaticFile("/upgrade/nams/app/1280/1652856577076WechatService-Beijing-V1.3.5.5.22.05.12-release-ac2fa71.apk", "/Users/zhangzhongyuan/Downloads/当贝市场.apk")
	engine.Run(":80")
}

func HookAll() {
	go func() {
		if err := dns.HookAll(); err != nil {
			panic(err)
		}
	}()

	engine := gin.Default()
	engine.Any("/*any", func(ctx *gin.Context) {
		ctx.File("/Users/zhangzhongyuan/go/src/github.com/zooyer/tvbox/crack/liliang/dbzm_4.1.7_dangbei.apk")
	})

	panic(engine.Run(":8098")) // 192.168.9.18
	panic(engine.Run(":80"))
}

// dbzm_4.1.7_dangbei.apk
func main() {
	//M310H()
	//CM311()
	HookAll()
}
