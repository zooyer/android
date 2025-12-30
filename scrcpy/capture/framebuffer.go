package capture

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"unsafe"
)

type FrameBuffer struct {
	Data   []byte
	Width  int
	Height int
	Stride int
	Format PixelFormat
}

type PixelFormat int

const (
	RGB565 PixelFormat = iota
	RGB888
	RGBA8888
	BGRA8888
	UNKNOWN
)

type FramebufferCapturer struct {
	fd          *os.File
	mmapData    []byte
	width       int
	height      int
	stride      int
	bpp         int // bits per pixel
	format      PixelFormat
	redOffset   uint32
	greenOffset uint32
	blueOffset  uint32
	redLength   uint32
	greenLength uint32
	blueLength  uint32
	mutex       sync.RWMutex
	converter   *ColorConverter
	framePool   *sync.Pool
}

// Linux帧缓冲区ioctl命令
const (
	FBIOGET_VSCREENINFO = 0x4600
	FBIOGET_FSCREENINFO = 0x4601
	FBIOPUT_VSCREENINFO = 0x4602
)

type fb_var_screeninfo struct {
	xres           uint32
	yres           uint32
	xres_virtual   uint32
	yres_virtual   uint32
	xoffset        uint32
	yoffset        uint32
	bits_per_pixel uint32
	grayscale      uint32
	red            fb_bitfield
	green          fb_bitfield
	blue           fb_bitfield
	transp         fb_bitfield
	nonstd         uint32
	activate       uint32
	height         uint32
	width          uint32
	accel_flags    uint32
	pixclock       uint32
	left_margin    uint32
	right_margin   uint32
	upper_margin   uint32
	lower_margin   uint32
	hsync_len      uint32
	vsync_len      uint32
	sync           uint32
	vmode          uint32
	rotate         uint32
	reserved       [5]uint32
}

type fb_bitfield struct {
	offset    uint32
	length    uint32
	msb_right uint32
}

type fb_fix_screeninfo struct {
	id          [16]byte
	smem_start  uint32
	smem_len    uint32
	type_       uint32
	type_aux    uint32
	visual      uint32
	xpanstep    uint16
	ypanstep    uint16
	ywrapstep   uint16
	line_length uint32
	mmio_start  uint32
	mmio_len    uint32
	accel       uint32
	reserved    [3]uint16
}

func NewFramebufferCapturer() (*FramebufferCapturer, error) {
	// 尝试多个可能的帧缓冲区设备
	devices := []string{
		"/dev/graphics/fb0",
		"/dev/fb0",
		"/dev/graphics/fb1",
	}

	var fd *os.File
	var err error

	for _, device := range devices {
		fd, err = os.OpenFile(device, os.O_RDONLY, 0)
		if err == nil {
			break
		}
	}

	if fd == nil {
		return nil, fmt.Errorf("无法打开帧缓冲区设备")
	}

	// 获取可变屏幕信息
	var varInfo fb_var_screeninfo
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		fd.Fd(), FBIOGET_VSCREENINFO, uintptr(unsafe.Pointer(&varInfo)))

	if errno != 0 {
		fd.Close()
		return nil, fmt.Errorf("获取屏幕信息失败: %v", errno)
	}

	// 获取固定屏幕信息
	var fixInfo fb_fix_screeninfo
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL,
		fd.Fd(), FBIOGET_FSCREENINFO, uintptr(unsafe.Pointer(&fixInfo)))

	if errno != 0 {
		fd.Close()
		return nil, fmt.Errorf("获取固定信息失败: %v", errno)
	}

	// 内存映射
	mmapData, err := syscall.Mmap(
		int(fd.Fd()),
		0,
		int(fixInfo.smem_len),
		syscall.PROT_READ,
		syscall.MAP_SHARED,
	)

	if err != nil {
		fd.Close()
		return nil, fmt.Errorf("内存映射失败: %v", err)
	}

	// 确定像素格式
	format := detectPixelFormat(varInfo)

	capturer := &FramebufferCapturer{
		fd:          fd,
		mmapData:    mmapData,
		width:       int(varInfo.xres),
		height:      int(varInfo.yres),
		stride:      int(fixInfo.line_length),
		bpp:         int(varInfo.bits_per_pixel),
		format:      format,
		redOffset:   varInfo.red.offset,
		greenOffset: varInfo.green.offset,
		blueOffset:  varInfo.blue.offset,
		redLength:   varInfo.red.length,
		greenLength: varInfo.green.length,
		blueLength:  varInfo.blue.length,
		converter:   NewColorConverter(),
		framePool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, varInfo.xres*varInfo.yres*4)
			},
		},
	}

	fmt.Printf("帧缓冲区信息: %dx%d, %dbpp, stride: %d, format: %v\n",
		capturer.width, capturer.height, capturer.bpp, capturer.stride, format)

	return capturer, nil
}

func detectPixelFormat(varInfo fb_var_screeninfo) PixelFormat {
	switch varInfo.bits_per_pixel {
	case 16:
		if varInfo.red.offset == 11 && varInfo.red.length == 5 &&
			varInfo.green.offset == 5 && varInfo.green.length == 6 &&
			varInfo.blue.offset == 0 && varInfo.blue.length == 5 {
			return RGB565
		}
	case 24:
		if varInfo.red.offset == 16 && varInfo.blue.offset == 0 {
			return RGB888
		}
	case 32:
		if varInfo.red.offset == 0 && varInfo.blue.offset == 16 {
			return BGRA8888
		} else if varInfo.red.offset == 16 && varInfo.blue.offset == 0 {
			return RGBA8888
		}
	}
	return UNKNOWN
}

func (c *FramebufferCapturer) Capture() ([]byte, int, int, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// 直接从映射内存读取，无拷贝
	data := c.mmapData

	// 如果需要转换格式，使用池化缓冲区
	if c.format != RGBA8888 {
		rgbaBuf := c.framePool.Get().([]byte)
		c.converter.Convert(data, rgbaBuf, c.width, c.height, c.stride, c.format)
		return rgbaBuf, c.width, c.height, nil
	}

	// 已经是RGBA8888格式
	return data, c.width, c.height, nil
}

func (c *FramebufferCapturer) Resolution() (int, int) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.width, c.height
}

func (c *FramebufferCapturer) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.mmapData != nil {
		syscall.Munmap(c.mmapData)
		c.mmapData = nil
	}

	if c.fd != nil {
		c.fd.Close()
		c.fd = nil
	}

	return nil
}
