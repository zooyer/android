package uinput

import (
	"encoding/binary"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

const (
	EvSyn = 0x00 // 事件同步（同步事件处理）
	EvKey = 0x01 // 键盘事件（键盘按下释放操作）
	EvRel = 0x02 // 相对事件（相对上次事件变化，如鼠标滚轮滚动）
	EvAbs = 0x03 // 绝对事件（如触摸点坐标）
	EvMsc = 0x04 // 杂项事件（特殊事件，如设备状态改变）
	EvSw  = 0x05 // 开关事件（设备开启和关闭）
	EvLed = 0x11 // LED事件（LED灯开关状态）
	EvSnd = 0x12 // 声音事件（如音量条件）
	EvRep = 0x14 // 重复事件（如按键重复）
	EvFF  = 0x15 // 力反馈事件（如震动手柄）

	SYNReport = 0

	MSCSerial   = 0x00
	MSCPulseLed = 0x01
	MSCGesture  = 0x02
	MSCRaw      = 0x03
	MSCScan     = 0x04 // 扫描码，按键的原始扫描码

	UISetEvBit  = 0x40045564
	UISetKeyBit = 0x40045565
	UIDevCreate = 0x5501

	EvIocGrab = 0x40044590
)

// InputEvent 必须与内核完全一致
type InputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

type Ident struct {
	Bus     uint16
	Vendor  uint16
	Product uint16
	Version uint16
}

// UserDev 结构大小必须正确
type UserDev struct {
	Name         [80]byte
	Ident        Ident
	FfEffectsMax uint32
	AbsMax       [64]int32
	AbsMin       [64]int32
	AbsFuzz      [64]int32
	AbsFlat      [64]int32
}

// Create 创建虚拟输入设备
func Create(name string, ident Ident, keys ...int) (file *os.File, err error) {
	if file, err = os.OpenFile("/dev/uinput", os.O_WRONLY|syscall.O_NONBLOCK, 0); err != nil {
		return
	}

	defer func() {
		if err != nil {
			_ = file.Close()
		}
	}()

	var fd = int(file.Fd())

	// 支持事件类型
	if err = unix.IoctlSetInt(fd, UISetEvBit, EvKey); err != nil {
		return
	}

	if err = unix.IoctlSetInt(fd, UISetEvBit, EvSyn); err != nil {
		return
	}

	for _, key := range keys {
		if err = unix.IoctlSetInt(fd, UISetKeyBit, key); err != nil {
			return
		}
	}

	var dev UserDev
	copy(dev.Name[:], name)
	dev.Ident = ident

	if err = binary.Write(file, binary.LittleEndian, &dev); err != nil {
		return
	}

	if err = unix.IoctlSetInt(fd, UIDevCreate, 0); err != nil {
		return
	}

	return
}

// Grab 抓取设备
func Grab(path string) (file *os.File, err error) {
	if file, err = os.Open(path); err != nil {
		return
	}

	if err = unix.IoctlSetInt(int(file.Fd()), EvIocGrab, 1); err != nil {
		return
	}

	return
}

// Close 关闭设备
func Close(file *os.File) (err error) {
	if err = unix.IoctlSetInt(int(file.Fd()), EvIocGrab, 0); err != nil {
		return
	}

	return file.Close()
}
