package input

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
	"unsafe"
)

// Linux输入事件定义
const (
	EV_KEY             = 0x01
	EV_ABS             = 0x03
	EV_SYN             = 0x00
	ABS_MT_POSITION_X  = 0x35
	ABS_MT_POSITION_Y  = 0x36
	ABS_MT_TRACKING_ID = 0x39
	ABS_MT_SLOT        = 0x2f
	SYN_REPORT         = 0x00
	BTN_TOUCH          = 0x14a
)

type input_event struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

type Injector struct {
	touchFd    *os.File
	keyboardFd *os.File
	verbose    bool
}

func NewInjector() (*Injector, error) {
	injector := &Injector{}

	// 查找触摸设备
	touchDev, err := findTouchDevice()
	if err != nil {
		return nil, fmt.Errorf("找不到触摸设备: %v", err)
	}

	injector.touchFd, err = os.OpenFile(touchDev, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("无法打开触摸设备: %v", err)
	}

	// 查找键盘设备
	keyboardDev, err := findKeyboardDevice()
	if err == nil {
		injector.keyboardFd, err = os.OpenFile(keyboardDev, os.O_RDWR, 0)
		if err != nil {
			fmt.Printf("警告: 无法打开键盘设备: %v\n", err)
		}
	}

	return injector, nil
}

func findTouchDevice() (string, error) {
	// 查找支持多点触控的设备
	for i := 0; i < 32; i++ {
		path := fmt.Sprintf("/dev/input/event%d", i)

		fd, err := os.Open(path)
		if err != nil {
			continue
		}
		defer fd.Close()

		// 检查设备能力
		var bitmask [1024]byte
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
			fd.Fd(), uintptr(0x80184506), uintptr(unsafe.Pointer(&bitmask[0])))

		if errno != 0 {
			continue
		}

		// 检查是否支持ABS_MT_POSITION_X
		if isBitSet(bitmask[:], EV_ABS) {
			// 获取绝对轴信息
			var absInfo [64]byte
			_, _, errno = syscall.Syscall(syscall.SYS_IOCTL,
				fd.Fd(), uintptr(0x80184540+ABS_MT_POSITION_X),
				uintptr(unsafe.Pointer(&absInfo[0])))

			if errno == 0 {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("未找到触摸设备")
}

func findKeyboardDevice() (string, error) {
	// 查找键盘设备
	for i := 0; i < 32; i++ {
		path := fmt.Sprintf("/dev/input/event%d", i)

		fd, err := os.Open(path)
		if err != nil {
			continue
		}
		defer fd.Close()

		// 检查设备能力
		var bitmask [1024]byte
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
			fd.Fd(), uintptr(0x80184506), uintptr(unsafe.Pointer(&bitmask[0])))

		if errno != 0 {
			continue
		}

		// 检查是否支持按键事件
		if isBitSet(bitmask[:], EV_KEY) {
			return path, nil
		}
	}

	return "", fmt.Errorf("未找到键盘设备")
}

func isBitSet(bitmask []byte, bit int) bool {
	byteIndex := bit / 8
	bitIndex := uint(bit % 8)

	if byteIndex >= len(bitmask) {
		return false
	}

	return (bitmask[byteIndex] & (1 << bitIndex)) != 0
}

func (i *Injector) InjectTouch(x, y int, action uint8) error {
	if i.touchFd == nil {
		return fmt.Errorf("触摸设备未初始化")
	}

	var events []input_event

	// 根据动作类型构造事件序列
	switch action {
	case 0: // 按下
		events = []input_event{
			{Type: EV_ABS, Code: ABS_MT_TRACKING_ID, Value: 0},
			{Type: EV_ABS, Code: ABS_MT_POSITION_X, Value: int32(x)},
			{Type: EV_ABS, Code: ABS_MT_POSITION_Y, Value: int32(y)},
			{Type: EV_ABS, Code: ABS_MT_SLOT, Value: 0},
			{Type: EV_SYN, Code: SYN_REPORT, Value: 0},
		}
	case 1: // 移动
		events = []input_event{
			{Type: EV_ABS, Code: ABS_MT_POSITION_X, Value: int32(x)},
			{Type: EV_ABS, Code: ABS_MT_POSITION_Y, Value: int32(y)},
			{Type: EV_ABS, Code: ABS_MT_SLOT, Value: 0},
			{Type: EV_SYN, Code: SYN_REPORT, Value: 0},
		}
	case 2: // 抬起
		events = []input_event{
			{Type: EV_ABS, Code: ABS_MT_TRACKING_ID, Value: -1},
			{Type: EV_SYN, Code: SYN_REPORT, Value: 0},
		}
	}

	// 写入事件
	for _, event := range events {
		event.Time = syscall.NsecToTimeval(time.Now().UnixNano())

		var buf bytes.Buffer
		binary.Write(&buf, binary.LittleEndian, event)

		if _, err := i.touchFd.Write(buf.Bytes()); err != nil {
			return fmt.Errorf("写入触摸事件失败: %v", err)
		}
	}

	return nil
}

func (i *Injector) InjectKey(keyCode uint16, pressed bool) error {
	if i.keyboardFd == nil {
		// 尝试使用虚拟键盘
		return i.injectKeyViaUinput(keyCode, pressed)
	}

	value := int32(0)
	if pressed {
		value = 1
	}

	event := input_event{
		Time:  syscall.NsecToTimeval(time.Now().UnixNano()),
		Type:  EV_KEY,
		Code:  uint16(keyCode),
		Value: value,
	}

	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, event)

	// 按键事件
	if _, err := i.keyboardFd.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("写入按键事件失败: %v", err)
	}

	// SYN_REPORT
	synEvent := input_event{
		Time:  syscall.NsecToTimeval(time.Now().UnixNano()),
		Type:  EV_SYN,
		Code:  SYN_REPORT,
		Value: 0,
	}

	buf.Reset()
	binary.Write(&buf, binary.LittleEndian, synEvent)

	if _, err := i.keyboardFd.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("写入同步事件失败: %v", err)
	}

	return nil
}

func (i *Injector) injectKeyViaUinput(keyCode uint16, pressed bool) error {
	// 备用方案：通过sendevent命令
	value := "0"
	if pressed {
		value = "1"
	}

	// 使用sendevent工具
	cmd := fmt.Sprintf("sendevent /dev/input/event1 1 %d %s", keyCode, value)
	return exec.Command("sh", "-c", cmd).Run()
}

func (i *Injector) Close() {
	if i.touchFd != nil {
		i.touchFd.Close()
	}
	if i.keyboardFd != nil {
		i.keyboardFd.Close()
	}
}
