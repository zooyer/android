package control

import (
	"encoding/binary"
	"log"
	"os/exec"
	"sync"

	"android/scrcpy/input"
)

type Controller struct {
	injector       *input.Injector
	config         *Config
	clipboard      string
	clipboardMutex sync.RWMutex
	screenOn       bool
}

type Config struct {
	StayAwake   bool
	ForwardAll  bool
	ShowTouches bool
	Verbose     bool
}

func NewController(config *Config) *Controller {
	injector, err := input.NewInjector()
	if err != nil {
		log.Printf("警告: 输入注入器初始化失败: %v", err)
	}

	return &Controller{
		injector: injector,
		config:   config,
		screenOn: true,
	}
}

func (c *Controller) InjectKeyEvent(data []byte) {
	if c.injector == nil {
		return
	}

	if len(data) < 4 {
		return
	}

	keyCode := binary.LittleEndian.Uint16(data[0:2])
	action := data[2]

	pressed := action == 0 // 0=按下, 1=抬起

	if err := c.injector.InjectKey(keyCode, pressed); err != nil && c.config.Verbose {
		log.Printf("注入按键事件失败: %v", err)
	}
}

func (c *Controller) InjectTouchEvent(data []byte) {
	if c.injector == nil {
		return
	}

	if len(data) < 8 {
		return
	}

	action := data[0]
	x := binary.LittleEndian.Uint16(data[2:4])
	y := binary.LittleEndian.Uint16(data[4:6])

	// 转换为实际屏幕坐标
	screenX, screenY := c.scaleCoordinates(int(x), int(y))

	if err := c.injector.InjectTouch(screenX, screenY, action); err != nil && c.config.Verbose {
		log.Printf("注入触摸事件失败: %v", err)
	}
}

func (c *Controller) scaleCoordinates(x, y int) (int, int) {
	// 这里应该根据实际屏幕分辨率缩放
	// 假设客户端发送的是相对坐标（0-1000）
	// 转换为实际像素坐标
	// screenWidth, screenHeight := 1920, 1080 // 应该从捕获器获取

	// scaledX := x * screenWidth / 1000
	// scaledY := y * screenHeight / 1000

	return x, y // 临时返回原始值
}

func (c *Controller) InjectScrollEvent(data []byte) {
	// 滚动事件可以通过多次触摸事件模拟
	if len(data) < 8 {
		return
	}

	x := binary.LittleEndian.Uint16(data[0:2])
	y := binary.LittleEndian.Uint16(data[2:4])
	hScroll := int16(binary.LittleEndian.Uint16(data[4:6]))
	vScroll := int16(binary.LittleEndian.Uint16(data[6:8]))

	// 模拟滚动
	if vScroll != 0 {
		// 通过多次触摸移动模拟滚动
		c.simulateScroll(int(x), int(y), int(vScroll))
	}
}

func (c *Controller) simulateScroll(x, y, delta int) {
	if c.injector == nil {
		return
	}

	// 按下
	c.injector.InjectTouch(x, y, 0)

	// 移动（模拟滚动）
	for i := 0; i < abs(delta)/10; i++ {
		newY := y - delta/abs(delta)*10*i
		c.injector.InjectTouch(x, newY, 1)
	}

	// 抬起
	c.injector.InjectTouch(x, y-delta, 2)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (c *Controller) SetClipboard(text string) {
	c.clipboardMutex.Lock()
	c.clipboard = text
	c.clipboardMutex.Unlock()

	// 设置Android剪贴板
	c.setAndroidClipboard(text)
}

func (c *Controller) setAndroidClipboard(text string) {
	// 通过am命令设置剪贴板
	// am broadcast -a clipper.set -e text "要设置的内容"
	cmd := exec.Command("am", "broadcast", "-a", "clipper.set", "-e", "text", text)
	if err := cmd.Run(); err != nil {
		// 备用方案：使用service call
		c.setClipboardViaService(text)
	}
}

func (c *Controller) setClipboardViaService(text string) {
	// 使用Android的clipboard服务
	// service call clipboard 2 i32 1 s16 "text"
	// 这需要系统权限
}

func (c *Controller) GetClipboard() string {
	c.clipboardMutex.RLock()
	defer c.clipboardMutex.RUnlock()
	return c.clipboard
}

func (c *Controller) SetScreenPower(on bool) {
	c.screenOn = on

	if on {
		// 唤醒屏幕
		exec.Command("input", "keyevent", "26").Run() // 电源键
	} else {
		// 关闭屏幕
		exec.Command("input", "keyevent", "26").Run()
	}
}

func (c *Controller) Close() {
	if c.injector != nil {
		c.injector.Close()
	}
}
