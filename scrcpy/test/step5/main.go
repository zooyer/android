package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"syscall"
	"unsafe"
)

// Linux帧缓冲区ioctl命令
const (
	FBIOGET_VSCREENINFO = 0x4600
	FBIOGET_FSCREENINFO = 0x4601
)

// 结构体定义
type fb_bitfield struct {
	offset    uint32
	length    uint32
	msb_right uint32
}

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

func main() {
	fmt.Println("=== Android帧缓冲区截图工具 (修正版) ===")

	// 1. 打开帧缓冲区设备
	fd, err := syscall.Open("/dev/graphics/fb0", syscall.O_RDONLY, 0)
	if err != nil {
		fmt.Printf("❌ 打开设备失败: %v\n", err)
		fmt.Println("请确保:")
		fmt.Println("1. 设备已root")
		fmt.Println("2. 程序以root权限运行")
		fmt.Println("3. 设备文件存在: ls -l /dev/graphics/fb0")
		return
	}
	defer syscall.Close(fd)

	fmt.Printf("✓ 设备打开成功，文件描述符: %d\n", fd)

	// 2. 获取可变屏幕信息 (包含分辨率、偏移量等)
	var vinfo fb_var_screeninfo
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		FBIOGET_VSCREENINFO,
		uintptr(unsafe.Pointer(&vinfo)),
	)

	if errno != 0 {
		fmt.Printf("❌ 获取屏幕信息失败 (errno=%d)\n", errno)
		return
	}

	fmt.Printf("✓ 屏幕信息获取成功\n")
	fmt.Printf("  可见分辨率: %dx%d\n", vinfo.xres, vinfo.yres)
	fmt.Printf("  虚拟分辨率: %dx%d\n", vinfo.xres_virtual, vinfo.yres_virtual)
	fmt.Printf("  可见区域偏移: x=%d, y=%d\n", vinfo.xoffset, vinfo.yoffset)
	fmt.Printf("  每像素位数: %d\n", vinfo.bits_per_pixel)

	// 打印颜色位字段信息 (关键!)
	fmt.Printf("  颜色格式:\n")
	fmt.Printf("    红色: offset=%d, length=%d\n", vinfo.red.offset, vinfo.red.length)
	fmt.Printf("    绿色: offset=%d, length=%d\n", vinfo.green.offset, vinfo.green.length)
	fmt.Printf("    蓝色: offset=%d, length=%d\n", vinfo.blue.offset, vinfo.blue.length)
	fmt.Printf("    透明度: offset=%d, length=%d\n", vinfo.transp.offset, vinfo.transp.length)

	// 3. 获取固定屏幕信息 (包含行长度等)
	var finfo fb_fix_screeninfo
	_, _, errno = syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		FBIOGET_FSCREENINFO,
		uintptr(unsafe.Pointer(&finfo)),
	)

	if errno != 0 {
		fmt.Printf("❌ 获取固定信息失败 (errno=%d)\n", errno)
		return
	}

	fmt.Printf("✓ 固定信息获取成功\n")
	fmt.Printf("  行长度: %d 字节\n", finfo.line_length)
	fmt.Printf("  内存大小: %d 字节 (%.2f MB)\n", finfo.smem_len,
		float64(finfo.smem_len)/(1024*1024))
	fmt.Printf("  缓冲区类型: %d\n", finfo.type_)

	// 4. 计算关键参数
	visibleWidth := int(vinfo.xres)          // 720
	visibleHeight := int(vinfo.yres)         // 576
	virtualHeight := int(vinfo.yres_virtual) // 1152
	bpp := int(vinfo.bits_per_pixel)         // 32

	// 使用实际的行长度
	stride := int(finfo.line_length)
	if stride == 0 {
		// 如果line_length为0，则计算
		stride = int(vinfo.xres_virtual) * (bpp / 8) // 720 * 4 = 2880
	}

	// 计算整个framebuffer大小
	fbSize := stride * virtualHeight // 2880 * 1152 = 3317760

	// 计算可见区域的起始偏移 (关键修正!)
	byteOffset := int(vinfo.yoffset) * stride
	byteOffset += int(vinfo.xoffset) * (bpp / 8)

	fmt.Printf("\n📐 计算参数:\n")
	fmt.Printf("  可见区域: %dx%d\n", visibleWidth, visibleHeight)
	fmt.Printf("  虚拟高度: %d (双缓冲)\n", virtualHeight)
	fmt.Printf("  行跨距: %d 字节\n", stride)
	fmt.Printf("  Framebuffer总大小: %d 字节\n", fbSize)
	fmt.Printf("  可见数据起始偏移: %d 字节 (yoffset=%d行)\n",
		byteOffset, vinfo.yoffset)

	// 5. 映射整个framebuffer到内存
	fmt.Printf("\n🗺️  映射内存...\n")
	mmapData, err := syscall.Mmap(
		fd,
		0,
		fbSize,
		syscall.PROT_READ,
		syscall.MAP_SHARED,
	)

	if err != nil {
		fmt.Printf("❌ 内存映射失败: %v\n", err)
		return
	}
	defer syscall.Munmap(mmapData)

	fmt.Printf("✓ 成功映射 %d 字节内存\n", len(mmapData))

	// 6. 检查偏移是否有效
	if byteOffset+visibleHeight*stride > len(mmapData) {
		fmt.Printf("❌ 偏移计算错误: 偏移超出内存范围\n")
		fmt.Printf("   偏移: %d, 需要: %d, 实际: %d\n",
			byteOffset, byteOffset+visibleHeight*stride, len(mmapData))
		return
	}

	// 7. 创建RGBA图像并转换数据
	fmt.Printf("\n🎨 创建图像 (格式转换)...\n")

	// 根据透明度长度判断是否有Alpha通道
	hasAlpha := vinfo.transp.length > 0
	fmt.Printf("  Alpha通道: %v\n", hasAlpha)

	// 创建图像
	img := image.NewRGBA(image.Rect(0, 0, visibleWidth, visibleHeight))

	// 从正确偏移开始读取数据
	src := mmapData[byteOffset:]

	// 根据文章分析和你的设备信息进行格式转换
	// 你的设备: red.offset=16, green.offset=0, blue.offset=8, transp.length=0
	// 格式为: G(0-7), B(8-15), R(16-23), X(24-31)
	convertFramebufferToRGBA(src, img.Pix, visibleWidth, visibleHeight, stride, vinfo)

	// 8. 保存为PNG文件
	fmt.Printf("\n💾 保存图像...\n")
	outputPath := "/sdcard/screenshot.png"

	file, err := os.Create(outputPath)
	if err != nil {
		// 尝试备用路径
		outputPath = "/data/local/tmp/screenshot.png"
		file, err = os.Create(outputPath)
		if err != nil {
			fmt.Printf("❌ 创建文件失败: %v\n", err)
			return
		}
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		fmt.Printf("❌ 保存PNG失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 截图成功保存到: %s\n", outputPath)
	fmt.Printf("   图像尺寸: %dx%d\n", visibleWidth, visibleHeight)

	// 9. 额外保存一份原始数据用于调试
	if len(mmapData) <= 10*1024*1024 { // 不超过10MB
		debugPath := "/sdcard/framebuffer_raw.bin"
		debugFile, err := os.Create(debugPath)
		if err == nil {
			debugFile.Write(mmapData)
			debugFile.Close()
			fmt.Printf("📊 原始数据保存到: %s (用于调试)\n", debugPath)
		}
	}
}

// 转换framebuffer数据到RGBA格式
func convertFramebufferToRGBA(src, dst []byte, width, height, stride int, vinfo fb_var_screeninfo) {
	bpp := int(vinfo.bits_per_pixel)
	bytesPerPixel := bpp / 8

	// 根据颜色位字段信息确定格式
	redOffset := int(vinfo.red.offset / 8)      // 16/8 = 2
	greenOffset := int(vinfo.green.offset / 8)  // 0/8 = 0
	blueOffset := int(vinfo.blue.offset / 8)    // 8/8 = 1
	alphaOffset := int(vinfo.transp.offset / 8) // 24/8 = 3
	hasAlpha := vinfo.transp.length > 0

	fmt.Printf("  转换参数: R@%d, G@%d, B@%d", redOffset, greenOffset, blueOffset)
	if hasAlpha {
		fmt.Printf(", A@%d\n", alphaOffset)
	} else {
		fmt.Printf(", 无Alpha\n")
	}

	for y := 0; y < height; y++ {
		srcRow := src[y*stride:]
		dstRow := dst[y*width*4:]

		for x := 0; x < width; x++ {
			srcPos := x * bytesPerPixel
			dstPos := x * 4

			// 根据偏移读取各通道
			r := srcRow[srcPos+redOffset]
			g := srcRow[srcPos+greenOffset]
			b := srcRow[srcPos+blueOffset]

			// 处理Alpha通道
			var a byte = 255
			if hasAlpha && alphaOffset < bytesPerPixel {
				a = srcRow[srcPos+alphaOffset]
			}

			// 转换为标准RGBA
			dstRow[dstPos] = r   // Red
			dstRow[dstPos+1] = g // Green
			dstRow[dstPos+2] = b // Blue
			dstRow[dstPos+3] = a // Alpha
		}
	}
}

// 打印原始数据头用于调试
func debugRawData(data []byte, offset, length int) {
	fmt.Printf("\n🔍 原始数据头 %d 字节 (偏移 %d):\n", length, offset)

	for i := 0; i < length && offset+i < len(data); i += 16 {
		fmt.Printf("  %08x: ", offset+i)

		// 十六进制显示
		for j := 0; j < 16 && offset+i+j < len(data); j++ {
			fmt.Printf("%02x ", data[offset+i+j])
			if j == 7 {
				fmt.Printf(" ")
			}
		}

		// ASCII显示
		fmt.Printf(" |")
		for j := 0; j < 16 && offset+i+j < len(data); j++ {
			b := data[offset+i+j]
			if b >= 32 && b <= 126 {
				fmt.Printf("%c", b)
			} else {
				fmt.Printf(".")
			}
		}
		fmt.Printf("|\n")
	}
}
