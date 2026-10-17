package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"android/tvbox/keyd/lib/conf"
	"android/tvbox/keyd/lib/log"
	"android/tvbox/keyd/lib/uinput"
	"android/tvbox/keyd/lib/utils"
)

// 按键事件间隔常量（单位：毫秒）
const (
	intervalClick    = time.Millisecond * 200 // 触发条件：小于（按下松开间隔）
	intervalDouble   = time.Millisecond * 200 // 触发条件：小于（两次 Click 间隔）
	intervalLong     = time.Millisecond * 300 // 触发条件：大于（按下达到该时间则触发一次）
	intervalRepeat   = time.Millisecond * 400 // 触发条件：同 Long（按间隔重复触发）
	intervalCombo    = time.Millisecond * 500 // 触发条件：小于（每次 Press 间隔）
	intervalPassword = time.Millisecond * 500 // 触发条件：小于（每次 Click 间隔）
)

var keys = []int{
	2, 3, 4, 5, 6, 7, 8, 9, 10, 11, // KEY_1 ... KEY_9、KEY_0
	14, 55, 63, 64, 65, 66, 67, // KEY_DEL, KEY_DEL, KEY_F12, KEY_F6
	102, 103, 104, // KEY_UP, KEY_DOWN, KEY_LEFT
	105, 106, 107, 108, 109, // KEY_RIGHT, KEY_ENTER, KEY_ESC
	113, 114, 115, 116, // KEY_VOLUME_MUTE, KEY_POWER,
	127, 138,
	158, 139, 140, // KEY_PLAY, KEY_STOP, KEY_PAUSE
	164, 165, 166, // KEY_REWIND, KEY_FASTFORWARD, KEY_NEXT
	167, 168, 169, // KEY_PREVIOUS, KEY_RECORD, KEY_RED
	170, 171, 172, // KEY_GREEN, KEY_YELLOW, KEY_BLUE
	183, 184, 185, // KEY_VOLUMEUP, KEY_VOLUMEDOWN, KEY_MUTE
	195,
	200, 201, 202, // KEY_CHANNELUP, KEY_CHANNELDOWN, KEY_POWER
	213, 214, 215, // KEY_HOME, KEY_MENU, KEY_INFO
	217, 218, 219, // KEY_EXIT, KEY_SETUP, KEY_TV
	224, 225, 226, // KEY_NUMERIC_0, KEY_NUMERIC_1, KEY_NUMERIC_2
	227, 228, 229, // KEY_NUMERIC_3, KEY_NUMERIC_4, KEY_NUMERIC_5
	230, 231, 232, // KEY_NUMERIC_6, KEY_NUMERIC_7, KEY_NUMERIC_8
	233,      // KEY_NUMERIC_9
	249,      // KEY_SETTINGS
	250, 353, // CMCC_MAGIC, DPAD_CENTER
	// 其他常见遥控器按键
	256, 257, 258, // KEY_AUDIO, KEY_VIDEO, KEY_TV
	259, 260, 261, // KEY_RADIO, KEY_BOOKMARK, KEY_CAPTION
}

var (
	config = flag.String("c", "keyd.yaml", "config file")

	files []*os.File
	mutex sync.Mutex
)

func initNoSighup() {
	signals := make(chan os.Signal, 128)
	signal.Notify(signals, syscall.SIGHUP)
}

func init() {
	initNoSighup()

	flag.Parse()

	if err := conf.Init(*config); err != nil {
		panic(err)
	}

	log.Init(conf.Config.Debug)

	var signals = make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signals

		mutex.Lock()
		defer mutex.Unlock()

		for _, file := range files {
			if err := uinput.Close(file); err != nil {
				log.Warn("close grab error:", err)
			}
		}

		os.Exit(0)
	}()
}

// fileExists 判断文件是否存在
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if err != nil {
		return false
	}

	return true
}

func doCommand(cmd string) {
	if cmd == "" {
		return
	}

	log.Info("exec doCommand:", cmd)

	var shell = conf.Config.Shell
	if shell == "" || !fileExists(shell) {
		shell = os.Getenv("SHELL")
	}

	// 未找到默认环境变量中的shell，选择一个已存在的shell
	if shell == "" || !fileExists(shell) {
		shell = "/system/bin/sh"
	}

	// 未找到 shell
	if !fileExists(shell) {
		log.Warn("shell not found:", shell)
		return
	}

	if err := exec.Command(shell, "-c", cmd).Run(); err != nil {
		log.Warn("run doCommand error:", err)
	}
}

// KeyState 按键状态
type KeyState struct {
	Pressed       bool
	LastPress     time.Time
	LastRelease   time.Time
	LastClickTime time.Time
	LongFired     bool
	MockKey       bool

	ClickTimer   *time.Timer
	LongTimer    *time.Timer
	RepeatTicker *time.Ticker
}

var (
	keyMutex  sync.Mutex
	keyStates = make(map[uint16]*KeyState)
)

func hookMatch(h conf.KeyHook, code uint16) bool {
	switch v := h.Keycode.(type) {
	case int:
		return uint16(v) == code
	case []interface{}:
		for _, k := range v {
			if uint16(k.(int)) == code {
				return true
			}
		}
	}

	return false
}

func keyHasHook(code uint16, hooks []conf.KeyHook) bool {
	for _, h := range hooks {
		if hookMatch(h, code) {
			return true
		}
	}
	return false
}

func actionHooked(code uint16, action conf.Action, hooks []conf.KeyHook) bool {
	for _, h := range hooks {
		if h.Action == action && hookMatch(h, code) {
			return true
		}
	}
	return false
}

func writeUI(ui *os.File, typ, code uint16, value int32) {
	log.Debug("write ui:", typ, code, value)

	var now = time.Now()

	_ = binary.Write(ui, utils.LocalEndian, uinput.InputEvent{
		Time: syscall.Timeval{
			Sec:  int32(now.Unix()),
			Usec: int32(now.Nanosecond() / 1000),
		},
		Type:  typ,
		Code:  code,
		Value: value,
	})
}

func emitKey(ui *os.File, code uint16, value int32) {
	log.Debug("emit key:", code, value)
	writeUI(ui, uinput.EvKey, code, value)
	time.Sleep(100 * time.Millisecond)
	emitSyn(ui)
}

func emitSyn(ui *os.File) {
	log.Debug("emit syn")
	writeUI(ui, uinput.EvSyn, uinput.SYNReport, 0)
}

func onClick(ui *os.File, code uint16) {
	emitKey(ui, code, 1)
	emitKey(ui, code, 0)
}

func handleKeyWithEmit(ev uinput.InputEvent, hooks []conf.KeyHook, ui *os.File) {
	if ev.Type != uinput.EvKey {
		return
	}

	keyMutex.Lock()
	defer keyMutex.Unlock()

	s, ok := keyStates[ev.Code]
	if !ok {
		s = &KeyState{}
		keyStates[ev.Code] = s
	}

	now := time.Now()

	switch ev.Value {

	case 1: // PRESS
		s.Pressed = true
		s.LastPress = now
		s.LongFired = false
		s.MockKey = false

		if s.ClickTimer != nil {
			s.ClickTimer.Stop()
			s.ClickTimer = nil
		}

		for _, h := range hooks {
			if h.Action == conf.ActionPress && hookMatch(h, ev.Code) {
				go doCommand(h.Command)
			}
		}

		s.LongTimer = time.AfterFunc(intervalLong, func() {
			keyMutex.Lock()
			defer keyMutex.Unlock()

			if !s.Pressed || s.LongFired {
				return
			}

			s.LongFired = true

			log.Info("key long:", ev.Code)

			if actionHooked(ev.Code, conf.ActionLong, hooks) {
				for _, h := range hooks {
					if h.Action == conf.ActionLong && hookMatch(h, ev.Code) {
						go doCommand(h.Command)
					}
				}
			} else {
				// 没 hook LONG → 模拟长按
				emitKey(ui, ev.Code, ev.Value)
				s.MockKey = true
			}

			s.RepeatTicker = time.NewTicker(intervalRepeat)
			go func() {
				for range s.RepeatTicker.C {
					keyMutex.Lock()
					if !s.Pressed {
						keyMutex.Unlock()
						return
					}

					keyMutex.Unlock()

					for _, h := range hooks {
						if h.Action == conf.ActionRepeat && hookMatch(h, ev.Code) {
							go doCommand(h.Command)
						}
					}
				}
			}()
		})

	case 0: // RELEASE
		s.Pressed = false
		s.LastRelease = now

		if s.LongTimer != nil {
			s.LongTimer.Stop()
		}

		if s.RepeatTicker != nil {
			s.RepeatTicker.Stop()
		}

		if s.MockKey {
			go func() {
				time.Sleep(intervalLong)
				emitKey(ui, ev.Code, ev.Value)
			}()
		}

		for _, h := range hooks {
			if h.Action == conf.ActionRelease && hookMatch(h, ev.Code) {
				go doCommand(h.Command)
			}
		}

		dur := now.Sub(s.LastPress)
		if dur <= intervalClick && !s.LongFired {
			isDouble := now.Sub(s.LastClickTime) <= intervalDouble

			if isDouble {
				log.Info("double:", ev.Code)

				if s.ClickTimer != nil {
					s.ClickTimer.Stop()
					s.ClickTimer = nil
				}

				for _, h := range hooks {
					if h.Action == conf.ActionDouble && hookMatch(h, ev.Code) {
						go doCommand(h.Command)
					}
				}

				s.LastClickTime = time.Time{}
			} else {
				if s.ClickTimer != nil {
					s.ClickTimer.Stop()
				}

				s.ClickTimer = time.AfterFunc(intervalDouble, func() {
					keyMutex.Lock()
					defer keyMutex.Unlock()

					if !s.Pressed && !s.LongFired {
						log.Info("click:", ev.Code)

						if actionHooked(ev.Code, conf.ActionClick, hooks) {
							for _, h := range hooks {
								if h.Action == conf.ActionClick && hookMatch(h, ev.Code) {
									go doCommand(h.Command)
								}
							}
						} else {
							log.Info("mock click:", ev.Code)
							onClick(ui, ev.Code)
						}
						s.LastClickTime = time.Now()
					}
				})
			}

			s.LastClickTime = now
		}
	}

	// TODO 支持COMBO、PASSWORD组合密码模式
}

func run(dev conf.Device) {
	// 打开输入设备
	input, err := uinput.Grab(dev.Input)
	if err != nil {
		log.Error("grab input error:", err)
		return
	}

	mutex.Lock()
	files = append(files, input)
	mutex.Unlock()

	// 创建虚拟设备
	ui, err := uinput.Create(fmt.Sprintf("%s.keyd", dev.Name), dev.Ident, append(keys, conf.Config.Codes...)...)
	if err != nil {
		log.Error("create uinput error:", err)
		return
	}
	defer ui.Close()

	// 确保异常退出不会锁死遥控
	defer uinput.Close(input)

	var buf = make([]byte, unsafe.Sizeof(uinput.InputEvent{}))

	for {
		for i := 0; i < 3; i++ {
			if _, err = input.Read(buf); err == nil {
				break
			}

			time.Sleep(200 * time.Millisecond)
		}

		if err != nil {
			log.Warn(dev.Name, "read error:", err)
			return
		}

		var ev = *(*uinput.InputEvent)(unsafe.Pointer(&buf[0]))

		log.Debug(dev.Name, "event:", utils.MarshalString(ev))

		// 没有 hook 的键：完全透传
		if !keyHasHook(ev.Code, dev.Hooks) {
			if err = binary.Write(ui, utils.LocalEndian, ev); err != nil {
				log.Warn(dev.Name, "write error:", err)
			}
			log.Debug(dev.Name, "passthrough key:", ev.Code, "value:", ev.Value)
			continue
		}

		handleKeyWithEmit(ev, dev.Hooks, ui)
	}
}

type Event struct {
	Path    string // /dev/input/event4
	Name    string // CMCC_Voice_Remote
	Unique  string // 0c:f3:ce:1e:35:85
	Vendor  string // 0416
	Product string // 0300
	Bustype string // 0005
	Version string // 0000
}

// 读取单个文件内容，去换行空格
func readFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(b)), nil
}

func readEvent(event string) (_ *Event, err error) {
	var base = filepath.Join("/sys/class/input", event, "device")

	var e = Event{Path: filepath.Join("/dev/input", event)}

	if e.Name, err = readFile(filepath.Join(base, "name")); err != nil {
		return
	}

	if e.Unique, err = readFile(filepath.Join(base, "uniq")); err != nil {
		return
	}

	if e.Vendor, err = readFile(filepath.Join(base, "id/vendor")); err != nil {
		return
	}

	if e.Product, err = readFile(filepath.Join(base, "id/product")); err != nil {
		return
	}

	if e.Bustype, err = readFile(filepath.Join(base, "id/bustype")); err != nil {
		return
	}

	if e.Version, err = readFile(filepath.Join(base, "id/version")); err != nil {
		return
	}

	return &e, nil
}

// TODO 重构完，待验证
func main() {
	utils.PrintBanner()

	dirs, err := os.ReadDir("/dev/input")
	if err != nil {
		log.Error("read /dev/input:", err)
	}

	var events []Event
	for _, file := range dirs {
		var event *Event
		if event, err = readEvent(file.Name()); err != nil {
			log.Warn(file.Name(), err)
			continue
		}

		log.Debug(event.Name, "event:", utils.MarshalString(event))
		events = append(events, *event)
	}

	var wg sync.WaitGroup
	wg.Add(len(conf.Config.Device))

	for _, dev := range conf.Config.Device {
		for _, event := range events {
			if dev.Name == event.Name {
				dev.Input = event.Path
				break
			}
		}

		var dev = dev
		go func() {
			defer wg.Done()
			for {
				run(dev)
				time.Sleep(1 * time.Second)
			}
		}()
	}

	wg.Wait()
}
