package network

import "encoding/binary"

const (
	MsgTypeKeyEvent uint8 = iota
	MsgTypeTouchEvent
	MsgTypeScrollEvent
	MsgTypeClipboard
	MsgTypeSetScreenPower
)

// 键盘事件结构
type KeyEvent struct {
	KeyCode uint16
	Action  uint8 // 0=按下, 1=抬起
	Meta    uint8 // 修饰键
}

// 触摸事件结构
type TouchEvent struct {
	Action    uint8 // 0=按下, 1=移动, 2=抬起
	PointerID uint8 // 手指ID
	X         uint16
	Y         uint16
	Pressure  uint16 // 压力值
}

// 滚动事件结构
type ScrollEvent struct {
	X       uint16
	Y       uint16
	HScroll int16 // 水平滚动
	VScroll int16 // 垂直滚动
}

func SerializeKeyEvent(event KeyEvent) []byte {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint16(data[0:2], event.KeyCode)
	data[2] = event.Action
	data[3] = event.Meta
	return data
}

func SerializeTouchEvent(event TouchEvent) []byte {
	data := make([]byte, 8)
	data[0] = event.Action
	data[1] = event.PointerID
	binary.LittleEndian.PutUint16(data[2:4], event.X)
	binary.LittleEndian.PutUint16(data[4:6], event.Y)
	binary.LittleEndian.PutUint16(data[6:8], event.Pressure)
	return data
}
