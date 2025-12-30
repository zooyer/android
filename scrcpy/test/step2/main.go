// fb_demo_fixed.go - 修复版帧缓冲区测试
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"syscall"
	"unsafe"
)

const (
	FBIOGET_VSCREENINFO = 0x4600
	FBIOGET_FSCREENINFO = 0x4601
	FBIOPUT_VSCREENINFO = 0x4602
)

func main() {
	fmt.Println("=== Android帧缓冲区测试程序（修复版）===")

	// 尝试多种方法
	testMethods := []func() error{
		testMethod1, // 标准方法
		testMethod2, // 备用方法
		testMethod3, // 简单读取
	}

	for i, method := range testMethods {
		fmt.Printf("\n[尝试方法 %d]\n", i+1)
		if err := method(); err == nil {
			fmt.Println("✓ 测试成功！")
			return
		} else {
			fmt.Printf("⚠ 方法 %d 失败: %v\n", i+1, err)
		}
	}

	fmt.Println("\n❌ 所有方法都失败")
	fmt.Println("\n建议:")
	fmt.Println("1. 确保设备已root")
	fmt.Println("2. 尝试其他设备: /dev/fb0, /dev/graphics/fb1等")
	fmt.Println("3. 使用screencap命令: adb shell screencap -p /sdcard/screen.png")
}

type VarInfo struct {
	xres, yres       uint32
	xres_virtual     uint32
	yres_virtual     uint32
	xoffset, yoffset uint32
	bits_per_pixel   uint32
	grayscale        uint32
	red, green, blue struct {
		offset, length uint32
	}
	transp struct {
		offset, length uint32
	}
}

type FixInfo struct {
	id          [16]byte
	smem_start  uint64
	smem_len    uint64
	line_length uint32
}

// 方法1: 标准帧缓冲区访问
func testMethod1() error {
	fmt.Println("打开设备 /dev/graphics/fb0...")
	fd, err := os.OpenFile("/dev/graphics/fb0", os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("无法打开设备: %v", err)
	}
	defer fd.Close()

	// 获取可变信息
	var varInfo VarInfo

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		fd.Fd(), FBIOGET_VSCREENINFO, uintptr(unsafe.Pointer(&varInfo)))

	if errno != 0 {
		return fmt.Errorf("ioctl FBIOGET_VSCREENINFO 失败: %v", errno)
	}

	fmt.Printf("✓ 屏幕信息:\n")
	fmt.Printf("  分辨率: %dx%d\n", varInfo.xres, varInfo.yres)
	fmt.Printf("  虚拟分辨率: %dx%d\n", varInfo.xres_virtual, varInfo.yres_virtual)
	fmt.Printf("  偏移: %dx%d\n", varInfo.xoffset, varInfo.yoffset)
	fmt.Printf("  BPP: %d\n", varInfo.bits_per_pixel)
	fmt.Printf("  红色: offset=%d, length=%d\n", varInfo.red.offset, varInfo.red.length)
	fmt.Printf("  绿色: offset=%d, length=%d\n", varInfo.green.offset, varInfo.green.length)
	fmt.Printf("  蓝色: offset=%d, length=%d\n", varInfo.blue.offset, varInfo.blue.length)

	// 方法1A: 尝试获取固定信息
	fmt.Println("\n尝试获取固定信息...")
	var fixInfo FixInfo

	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL,
		fd.Fd(), FBIOGET_FSCREENINFO, uintptr(unsafe.Pointer(&fixInfo)))

	if errno == 0 {
		fmt.Printf("✓ 固定信息:\n")
		fmt.Printf("  内存大小: %d\n", fixInfo.smem_len)
		fmt.Printf("  行长度: %d\n", fixInfo.line_length)

		if fixInfo.smem_len > 0 {
			return tryMemoryMap(fd, varInfo, fixInfo)
		}
	}

	// 方法1B: 计算内存大小
	fmt.Println("通过计算获取内存大小...")
	lineLength := varInfo.xres_virtual * varInfo.bits_per_pixel / 8
	frameSize := lineLength * varInfo.yres_virtual

	fmt.Printf("计算的内存大小: %d 字节\n", frameSize)

	// 尝试映射计算的大小
	return tryMapSize(fd, int64(frameSize), varInfo)
}

// 方法2: 尝试其他设备
func testMethod2() error {
	devices := []string{
		"/dev/fb0",
		"/dev/graphics/fb1",
		"/dev/graphics/fb2",
		"/dev/graphics/fb3",
	}

	for _, device := range devices {
		fmt.Printf("尝试设备: %s\n", device)

		fd, err := os.OpenFile(device, os.O_RDONLY, 0)
		if err != nil {
			fmt.Printf("  无法打开: %v\n", err)
			continue
		}

		// 尝试读取前几个字节
		buf := make([]byte, 1024)
		n, err := fd.Read(buf)
		fd.Close()

		if err == nil && n > 0 {
			fmt.Printf("✓ 可以读取 %d 字节\n", n)

			// 保存原始数据
			if err := os.WriteFile("/sdcard/fb_data.bin", buf[:n], 0644); err == nil {
				fmt.Println("  数据已保存到 /sdcard/fb_data.bin")
			}

			// 尝试解析为图像
			if img := tryParseImage(buf, n); img != nil {
				saveImage(img, "/sdcard/screenshot.png")
				return nil
			}
		}
	}

	return fmt.Errorf("所有设备都无法读取")
}

// 方法3: 简单直接读取
func testMethod3() error {
	fmt.Println("尝试直接读取设备...")

	// 直接读取整个设备（可能很大，所以限制大小）
	const maxSize = 10 * 1024 * 1024 // 10MB

	data, err := os.ReadFile("/dev/graphics/fb0")
	if err != nil {
		return fmt.Errorf("直接读取失败: %v", err)
	}

	fmt.Printf("读取到 %d 字节数据\n", len(data))

	// 保存前1MB用于分析
	size := len(data)
	if size > 1024*1024 {
		size = 1024 * 1024
	}

	filename := "/sdcard/fb_dump.bin"
	if err := os.WriteFile(filename, data[:size], 0644); err != nil {
		// 尝试备用路径
		filename = "/data/local/tmp/fb_dump.bin"
		os.WriteFile(filename, data[:size], 0644)
	}

	fmt.Printf("数据已保存到 %s\n", filename)

	// 分析数据
	analyzeData(data)

	return nil
}

type a struct {
	xres, yres       uint32
	xres_virtual     uint32
	yres_virtual     uint32
	xoffset, yoffset uint32
	bits_per_pixel   uint32
	grayscale        uint32
	red, green, blue struct {
		offset, length uint32
	}
	transp struct {
		offset, length uint32
	}
}

func tryMemoryMap(fd *os.File, varInfo VarInfo, fixInfo FixInfo) error {
	fmt.Println("尝试内存映射...")

	// 使用固定信息中的大小
	size := int(fixInfo.smem_len)
	if size <= 0 {
		return fmt.Errorf("无效的内存大小: %d", size)
	}

	mmapData, err := syscall.Mmap(
		int(fd.Fd()),
		0,
		size,
		syscall.PROT_READ,
		syscall.MAP_SHARED,
	)

	if err != nil {
		return fmt.Errorf("内存映射失败: %v", err)
	}
	defer syscall.Munmap(mmapData)

	fmt.Printf("✓ 成功映射 %d 字节\n", len(mmapData))

	// 创建图像
	img := createRGBAImage(mmapData, int(varInfo.xres), int(varInfo.yres),
		int(varInfo.bits_per_pixel), int(fixInfo.line_length))

	// 保存图像
	return saveImage(img, "/sdcard/demo.png")
}

func tryMapSize(fd *os.File, size int64, varInfo VarInfo) error {
	if size <= 0 || size > 100*1024*1024 {
		size = 1920 * 1080 * 4 // 假设FHD RGBA
	}

	fmt.Printf("尝试映射 %d 字节...\n", size)

	mmapData, err := syscall.Mmap(
		int(fd.Fd()),
		0,
		int(size),
		syscall.PROT_READ,
		syscall.MAP_SHARED,
	)

	if err != nil {
		return fmt.Errorf("映射失败: %v", err)
	}
	defer syscall.Munmap(mmapData)

	fmt.Printf("✓ 映射成功，读取到 %d 字节\n", len(mmapData))

	// 尝试解析
	if len(mmapData) >= int(varInfo.xres*varInfo.yres*4) {
		img := createRGBAImage(mmapData,
			int(varInfo.xres),
			int(varInfo.yres),
			int(varInfo.bits_per_pixel),
			int(varInfo.xres)*4)

		return saveImage(img, "/sdcard/demo.png")
	}

	return fmt.Errorf("数据不足")
}

func createRGBAImage(data []byte, width, height, bpp, stride int) *image.RGBA {
	if stride <= 0 {
		stride = width * bpp / 8
	}

	fmt.Printf("创建图像: %dx%d, bpp=%d, stride=%d\n", width, height, bpp, stride)

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	switch bpp {
	case 32:
		// 根据你的输出，这是ARGB或BGRA格式（offset: R=16, G=8, B=0）
		convert32Bit(data, img.Pix, width, height, stride, 2, 1, 0) // BGR格式
	case 16:
		convertRGB565(data, img.Pix, width, height, stride)
	case 24:
		convertRGB24(data, img.Pix, width, height, stride)
	default:
		fmt.Printf("⚠ 不支持的BPP: %d，尝试作为原始数据处理\n", bpp)
		// 简单复制
		copy(img.Pix, data[:width*height*4])
	}

	return img
}

func convert32Bit(src, dst []byte, width, height, stride int, rIdx, gIdx, bIdx int) {
	for y := 0; y < height; y++ {
		srcRow := src[y*stride:]
		dstRow := dst[y*width*4:]

		for x := 0; x < width; x++ {
			srcPos := x * 4
			dstPos := x * 4

			// 根据你的offset: R=16(字节2), G=8(字节1), B=0(字节0)
			// 所以这是BGRA格式
			dstRow[dstPos] = srcRow[srcPos+2]   // R
			dstRow[dstPos+1] = srcRow[srcPos+1] // G
			dstRow[dstPos+2] = srcRow[srcPos]   // B
			dstRow[dstPos+3] = 255              // A
		}
	}
}

func convertRGB565(src, dst []byte, width, height, stride int) {
	for y := 0; y < height; y++ {
		srcRow := src[y*stride:]
		dstRow := dst[y*width*4:]

		for x := 0; x < width; x++ {
			idx := x * 2
			pixel := uint16(srcRow[idx]) | uint16(srcRow[idx+1])<<8

			r := (pixel >> 11) & 0x1F
			g := (pixel >> 5) & 0x3F
			b := pixel & 0x1F

			dstRow[x*4] = byte((r << 3) | (r >> 2))
			dstRow[x*4+1] = byte((g << 2) | (g >> 4))
			dstRow[x*4+2] = byte((b << 3) | (b >> 2))
			dstRow[x*4+3] = 255
		}
	}
}

func convertRGB24(src, dst []byte, width, height, stride int) {
	for y := 0; y < height; y++ {
		srcRow := src[y*stride:]
		dstRow := dst[y*width*4:]

		for x := 0; x < width; x++ {
			idx := x * 3
			dstRow[x*4] = srcRow[idx+2]   // R
			dstRow[x*4+1] = srcRow[idx+1] // G
			dstRow[x*4+2] = srcRow[idx]   // B
			dstRow[x*4+3] = 255           // A
		}
	}
}

func tryParseImage(data []byte, size int) *image.RGBA {
	if size < 100 {
		return nil
	}

	// 尝试检测格式
	fmt.Printf("分析数据格式...\n")
	fmt.Printf("前16字节: ")
	for i := 0; i < 16 && i < size; i++ {
		fmt.Printf("%02x ", data[i])
	}
	fmt.Println()

	// 尝试作为原始RGBA处理（720x576x4 = 1658880字节）
	expectedSize := 720 * 576 * 4
	if size >= expectedSize {
		fmt.Printf("数据大小匹配720x576 RGBA格式\n")
		return createRGBAImage(data, 720, 576, 32, 720*4)
	}

	return nil
}

func analyzeData(data []byte) {
	fmt.Println("\n数据分析:")
	fmt.Printf("总大小: %d 字节 (%.2f MB)\n", len(data), float64(len(data))/(1024*1024))

	if len(data) >= 100 {
		fmt.Println("前100字节十六进制:")
		for i := 0; i < 100; i += 16 {
			fmt.Printf("%04x: ", i)
			for j := 0; j < 16 && i+j < len(data); j++ {
				fmt.Printf("%02x ", data[i+j])
			}
			fmt.Println()
		}

		// 检查可能的图像数据模式
		fmt.Println("\n检查常见图像模式:")

		// 检查是否可能是BGRA
		hasAlpha := false
		for i := 3; i < 100 && i < len(data); i += 4 {
			if data[i] != 0 && data[i] != 255 {
				hasAlpha = true
				break
			}
		}

		fmt.Printf("Alpha通道: %v\n", hasAlpha)

		// 检查是否可能是有意义的数据（不是全0或全FF）
		allZero := true
		allFF := true
		for i := 0; i < 100 && i < len(data); i++ {
			if data[i] != 0 {
				allZero = false
			}
			if data[i] != 0xFF {
				allFF = false
			}
		}

		fmt.Printf("全0数据: %v\n", allZero)
		fmt.Printf("全FF数据: %v\n", allFF)
	}
}

func saveImage(img *image.RGBA, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		// 尝试备用路径
		filename = "/data/local/tmp/demo.png"
		file, err = os.Create(filename)
		if err != nil {
			return fmt.Errorf("创建文件失败: %v", err)
		}
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("编码PNG失败: %v", err)
	}

	fmt.Printf("✓ 图像保存到: %s\n", filename)
	return nil
}

// 快速测试脚本
func getQuickTestScript() string {
	return `#!/bin/bash
# quick_test_fb.sh - 快速帧缓冲区测试

echo "快速帧缓冲区测试..."

# 1. 编译程序
echo "1. 编译测试程序..."
GOOS=android GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -o fb_test_fixed fb_demo_fixed.go

# 2. 推送到设备
echo "2. 推送到设备..."
adb push fb_test_fixed /data/local/tmp/
adb shell chmod 755 /data/local/tmp/fb_test_fixed

# 3. 运行测试
echo "3. 运行测试..."
adb shell su -c "/data/local/tmp/fb_test_fixed 2>&1"

# 4. 拉取结果
echo "4. 检查结果..."
if adb shell "ls /sdcard/demo.png 2>/dev/null"; then
    echo "找到截图，正在拉取..."
    adb pull /sdcard/demo.png ./screenshot_$(date +%Y%m%d_%H%M%S).png
fi

if adb shell "ls /sdcard/fb_dump.bin 2>/dev/null"; then
    echo "找到原始数据，正在拉取..."
    adb pull /sdcard/fb_dump.bin ./fb_dump_$(date +%Y%m%d_%H%M%S).bin
fi

echo "测试完成！"
`
}
