// fb_demo.go - 帧缓冲区测试程序
package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"syscall"
	"unsafe"
)

// Linux帧缓冲区结构定义
const (
	FBIOGET_VSCREENINFO = 0x4600
	FBIOGET_FSCREENINFO = 0x4601
)

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
	fmt.Println("=== Android帧缓冲区测试程序 ===")
	fmt.Println("正在尝试读取/dev/graphics/fb0...")

	// 尝试打开帧缓冲区
	fd, err := os.OpenFile("/dev/graphics/fb0", os.O_RDONLY, 0)
	if err != nil {
		log.Printf("无法打开/dev/graphics/fb0: %v", err)

		// 尝试其他可能的设备
		fmt.Println("尝试备用设备...")
		devices := []string{"/dev/fb0", "/dev/graphics/fb1"}
		for _, dev := range devices {
			fd, err = os.OpenFile(dev, os.O_RDONLY, 0)
			if err == nil {
				fmt.Printf("成功打开设备: %s\n", dev)
				break
			}
			fmt.Printf("无法打开 %s: %v\n", dev, err)
		}

		if fd == nil {
			log.Fatal("所有帧缓冲区设备都无法访问，需要root权限？")
		}
	}
	defer fd.Close()

	fmt.Println("设备打开成功，获取屏幕信息...")

	// 获取可变屏幕信息
	var varInfo fb_var_screeninfo
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		fd.Fd(), FBIOGET_VSCREENINFO, uintptr(unsafe.Pointer(&varInfo)))

	if errno != 0 {
		log.Fatalf("获取屏幕信息失败 (errno=%d)", errno)
	}

	fmt.Printf("屏幕分辨率: %dx%d\n", varInfo.xres, varInfo.yres)
	fmt.Printf("每像素位数: %d\n", varInfo.bits_per_pixel)
	fmt.Printf("红色分量: offset=%d, length=%d\n", varInfo.red.offset, varInfo.red.length)
	fmt.Printf("绿色分量: offset=%d, length=%d\n", varInfo.green.offset, varInfo.green.length)
	fmt.Printf("蓝色分量: offset=%d, length=%d\n", varInfo.blue.offset, varInfo.blue.length)

	// 获取固定屏幕信息
	var fixInfo fb_fix_screeninfo
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL,
		fd.Fd(), FBIOGET_FSCREENINFO, uintptr(unsafe.Pointer(&fixInfo)))

	if errno != 0 {
		log.Fatalf("获取固定信息失败 (errno=%d)", errno)
	}

	fmt.Printf("内存大小: %d 字节\n", fixInfo.smem_len)
	fmt.Printf("行长度: %d 字节\n", fixInfo.line_length)
	fmt.Printf("设备ID: %s\n", string(fixInfo.id[:]))

	// 内存映射
	fmt.Println("正在映射内存...")
	mmapData, err := syscall.Mmap(
		int(fd.Fd()),
		0,
		int(fixInfo.smem_len),
		syscall.PROT_READ,
		syscall.MAP_SHARED,
	)

	if err != nil {
		log.Fatalf("内存映射失败: %v", err)
	}
	defer syscall.Munmap(mmapData)

	// 保存原始数据（可选，用于调试）
	if false { // 设置为true来保存原始数据
		saveRawData(mmapData, "/sdcard/framebuffer.raw")
	}

	// 创建PNG图像
	fmt.Println("正在创建PNG图像...")
	img, err := createImage(mmapData, &varInfo, &fixInfo)
	if err != nil {
		log.Fatalf("创建图像失败: %v", err)
	}

	// 保存到文件
	outputPath := "/sdcard/demo.png"
	fmt.Printf("正在保存到: %s\n", outputPath)

	file, err := os.Create(outputPath)
	if err != nil {
		log.Printf("无法创建文件: %v，尝试备用路径", err)
		outputPath = "/data/local/tmp/demo.png"
		file, err = os.Create(outputPath)
		if err != nil {
			log.Fatalf("无法创建备用文件: %v", err)
		}
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		log.Fatalf("保存PNG失败: %v", err)
	}

	fmt.Printf("✓ 图像保存成功: %s\n", outputPath)
	fmt.Println("=== 测试完成 ===")

	// 显示图像信息
	bounds := img.Bounds()
	fmt.Printf("图像尺寸: %dx%d\n", bounds.Dx(), bounds.Dy())
}

func createImage(data []byte, varInfo *fb_var_screeninfo, fixInfo *fb_fix_screeninfo) (*image.RGBA, error) {
	width := int(varInfo.xres)
	height := int(varInfo.yres)
	bpp := int(varInfo.bits_per_pixel)
	stride := int(fixInfo.line_length)

	fmt.Printf("创建图像: %dx%d, %dbpp, stride=%d\n", width, height, bpp, stride)

	// 创建RGBA图像
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// 根据像素格式处理数据
	switch bpp {
	case 16:
		convertRGB565(data, img.Pix, width, height, stride)
	case 24:
		convertRGB24(data, img.Pix, width, height, stride, varInfo)
	case 32:
		convertRGBA32(data, img.Pix, width, height, stride, varInfo)
	default:
		return nil, fmt.Errorf("不支持的像素格式: %d bpp", bpp)
	}

	return img, nil
}

func convertRGB565(src, dst []byte, width, height, stride int) {
	fmt.Println("转换 RGB565 格式...")

	bytesPerPixel := 2
	for y := 0; y < height; y++ {
		srcRow := src[y*stride:]
		dstRow := dst[y*width*4:]

		for x := 0; x < width; x++ {
			srcPos := x * bytesPerPixel

			// 读取RGB565像素 (小端字节序)
			pixel := uint16(srcRow[srcPos]) | uint16(srcRow[srcPos+1])<<8

			// 提取RGB分量 (5-6-5格式)
			r := (pixel >> 11) & 0x1F // 5位
			g := (pixel >> 5) & 0x3F  // 6位
			b := pixel & 0x1F         // 5位

			// 扩展到8位
			r = (r << 3) | (r >> 2)
			g = (g << 2) | (g >> 4)
			b = (b << 3) | (b >> 2)

			// 写入RGBA
			dstPos := x * 4
			dstRow[dstPos] = byte(r)   // R
			dstRow[dstPos+1] = byte(g) // G
			dstRow[dstPos+2] = byte(b) // B
			dstRow[dstPos+3] = 255     // A
		}
	}
}

func convertRGB24(src, dst []byte, width, height, stride int, varInfo *fb_var_screeninfo) {
	fmt.Println("转换 RGB24 格式...")

	// 确定通道顺序
	redOffset := int(varInfo.red.offset / 8)
	greenOffset := int(varInfo.green.offset / 8)
	blueOffset := int(varInfo.blue.offset / 8)

	fmt.Printf("通道偏移: R=%d, G=%d, B=%d\n", redOffset, greenOffset, blueOffset)

	for y := 0; y < height; y++ {
		srcRow := src[y*stride:]
		dstRow := dst[y*width*4:]

		for x := 0; x < width; x++ {
			srcPos := x * 3
			dstPos := x * 4

			dstRow[dstPos+redOffset] = srcRow[srcPos]     // R
			dstRow[dstPos+greenOffset] = srcRow[srcPos+1] // G
			dstRow[dstPos+blueOffset] = srcRow[srcPos+2]  // B
			dstRow[dstPos+3] = 255                        // A
		}
	}
}

func convertRGBA32(src, dst []byte, width, height, stride int, varInfo *fb_var_screeninfo) {
	fmt.Println("转换 RGBA32 格式...")

	// 确定通道顺序
	redOffset := int(varInfo.red.offset / 8)
	greenOffset := int(varInfo.green.offset / 8)
	blueOffset := int(varInfo.blue.offset / 8)
	alphaOffset := 3 // 假设alpha通道在第4个字节

	if varInfo.transp.length > 0 {
		alphaOffset = int(varInfo.transp.offset / 8)
	}

	fmt.Printf("通道偏移: R=%d, G=%d, B=%d, A=%d\n",
		redOffset, greenOffset, blueOffset, alphaOffset)

	for y := 0; y < height; y++ {
		srcRow := src[y*stride:]
		dstRow := dst[y*width*4:]

		for x := 0; x < width; x++ {
			srcPos := x * 4
			dstPos := x * 4

			// 重新排列通道到RGBA顺序
			dstRow[dstPos] = srcRow[srcPos+redOffset]     // R
			dstRow[dstPos+1] = srcRow[srcPos+greenOffset] // G
			dstRow[dstPos+2] = srcRow[srcPos+blueOffset]  // B
			dstRow[dstPos+3] = 255                        // A (或使用alpha通道)

			if varInfo.transp.length > 0 {
				dstRow[dstPos+3] = srcRow[srcPos+alphaOffset]
			}
		}
	}
}

func saveRawData(data []byte, path string) {
	fmt.Printf("保存原始数据到: %s\n", path)
	file, err := os.Create(path)
	if err != nil {
		log.Printf("无法保存原始数据: %v", err)
		return
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		log.Printf("写入原始数据失败: %v", err)
	}
}

// 简化的编译脚本
func buildScript() string {
	return `#!/bin/bash
# build_demo.sh - 构建测试程序

echo "构建Android测试程序..."

# 设置交叉编译环境
export GOOS=android
export GOARCH=arm
export GOARM=7
export CGO_ENABLED=0

# 编译
go build -o fb_demo fb_demo.go

if [ -f "fb_demo" ]; then
    echo "编译成功!"
    ls -lh fb_demo
    
    # 推送到设备
    echo "推送到Android设备..."
    adb push fb_demo /data/local/tmp/
    adb shell chmod 755 /data/local/tmp/fb_demo
    
    echo "运行测试程序 (需要root权限)..."
    adb shell su -c "/data/local/tmp/fb_demo"
    
    # 拉取结果
    echo "拉取生成的图片..."
    adb pull /sdcard/demo.png ./demo.png 2>/dev/null || adb pull /data/local/tmp/demo.png ./demo.png 2>/dev/null
    
    if [ -f "demo.png" ]; then
        echo "✓ 图片已保存到 ./demo.png"
        # 在Linux上尝试显示图片
        if command -v display >/dev/null 2>&1; then
            display demo.png &
        elif command -v xdg-open >/dev/null 2>&1; then
            xdg-open demo.png &
        fi
    fi
else
    echo "编译失败!"
    exit 1
fi
`
}

// 快速测试脚本
func testScript() string {
	return `#!/bin/bash
# quick_test.sh - 快速测试

echo "快速帧缓冲区测试..."

# 检查ADB连接
if ! adb devices | grep -q "device$"; then
    echo "错误: 没有找到连接的Android设备"
    exit 1
fi

# 检查root权限
echo "检查设备root权限..."
if adb shell su -c "echo test" | grep -q "test"; then
    echo "✓ 设备已root"
else
    echo "⚠ 设备可能未root，尝试继续..."
fi

# 检查帧缓冲区设备
echo "检查帧缓冲区设备..."
for dev in /dev/graphics/fb0 /dev/fb0 /dev/graphics/fb1; do
    if adb shell "ls $dev 2>/dev/null"; then
        echo "找到设备: $dev"
        DEVICE="$dev"
        break
    fi
done

if [ -z "$DEVICE" ]; then
    echo "未找到帧缓冲区设备"
    exit 1
fi

# 检查权限
echo "检查设备权限..."
adb shell "ls -l $DEVICE"

echo "测试完成！"
`
}
