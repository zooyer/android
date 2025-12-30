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

type BitField struct {
	Offset   uint32
	Length   uint32
	MsbRight uint32
}

type VarScreenInfo struct {
	XRes         uint32
	YRes         uint32
	XResVirtual  uint32
	YResVirtual  uint32
	XOffset      uint32
	YOffset      uint32
	BitsPerPixel uint32
	Grayscale    uint32
	Red          BitField
	Green        BitField
	Blue         BitField
	Transp       BitField
	Nonstd       uint32
	Activate     uint32
	Height       uint32
	Width        uint32
	AccelFlags   uint32
	PixClock     uint32
	LeftMargin   uint32
	RightMargin  uint32
	UpperMargin  uint32
	LowerMargin  uint32
	hsyncLen     uint32
	vsyncLen     uint32
	Sync         uint32
	VMode        uint32
	Rotate       uint32
	Reserved     [5]uint32
}

type FixScreenInfo struct {
	id         [16]byte
	SMemStart  uint32
	SMemLen    uint32
	Type       uint32
	TypeAux    uint32
	visual     uint32
	XPanStep   uint16
	YPanStep   uint16
	YWrapStep  uint16
	LineLength uint32
	MmioStart  uint32
	MmioLen    uint32
	Accel      uint32
	Reserved   [3]uint16
}

// 按 bitfield 提取并归一化到 0-255
func extract(src uint32, bf BitField) byte {
	if bf.Length == 0 {
		return 0
	}
	mask := (uint32(1) << bf.Length) - 1
	v := (src >> bf.Offset) & mask
	return byte(v * 255 / mask)
}

// framebuffer → RGBA（通吃 16 / 24 / 32 bpp）
func convertFramebufferToRGBA(src, dst []byte, width, height, stride int, info VarScreenInfo) {
	bpp := int(info.BitsPerPixel)
	bytesPerPixel := bpp / 8

	fmt.Printf("🎨 转换 framebuffer: %dbpp\n", bpp)
	fmt.Printf("    R: off=%d len=%d\n", info.Red.Offset, info.Red.Length)
	fmt.Printf("    G: off=%d len=%d\n", info.Green.Offset, info.Green.Length)
	fmt.Printf("    B: off=%d len=%d\n", info.Blue.Offset, info.Blue.Length)
	fmt.Printf("    A: off=%d len=%d\n", info.Transp.Offset, info.Transp.Length)

	for y := 0; y < height; y++ {
		srcRow := src[y*stride:]
		dstRow := dst[y*width*4:]

		for x := 0; x < width; x++ {
			srcPos := x * bytesPerPixel
			dstPos := x * 4

			var pixel uint32

			switch bytesPerPixel {
			case 2: // RGB565 / ARGB1555
				pixel = uint32(*(*uint16)(unsafe.Pointer(&srcRow[srcPos])))
			case 3: // RGB888
				pixel = uint32(srcRow[srcPos]) |
					uint32(srcRow[srcPos+1])<<8 |
					uint32(srcRow[srcPos+2])<<16
			case 4: // XRGB8888 / ARGB8888
				pixel = *(*uint32)(unsafe.Pointer(&srcRow[srcPos]))
			default:
				continue
			}

			r := extract(pixel, info.Red)
			g := extract(pixel, info.Green)
			b := extract(pixel, info.Blue)

			a := byte(255)
			if info.Transp.Length > 0 {
				a = extract(pixel, info.Transp)
			}

			dstRow[dstPos+0] = r
			dstRow[dstPos+1] = g
			dstRow[dstPos+2] = b
			dstRow[dstPos+3] = a
		}
	}
}

func main() {
	// 1. 打开帧缓冲区设备
	fb, err := os.OpenFile("/dev/graphics/fb0", os.O_RDONLY, 0)
	if err != nil {
		panic(err)
	}
	defer fb.Close()

	fmt.Printf("✓ 设备打开成功，文件描述符: %d\n", fb.Fd())

	var varInfo VarScreenInfo

	// 2. 获取可变屏幕信息 (包含分辨率、偏移量等)
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		fb.Fd(),
		FBIOGET_VSCREENINFO,
		uintptr(unsafe.Pointer(&varInfo)),
	)

	if errno != 0 {
		fmt.Printf("❌ 获取屏幕信息失败 (errno=%d)\n", errno)
		return
	}

	fmt.Printf("✓ 屏幕信息获取成功\n")
	fmt.Printf("  可见分辨率: %dx%d\n", varInfo.XRes, varInfo.YRes)
	fmt.Printf("  虚拟分辨率: %dx%d\n", varInfo.XResVirtual, varInfo.YResVirtual)
	fmt.Printf("  可见区域偏移: x=%d, y=%d\n", varInfo.XOffset, varInfo.YOffset)
	fmt.Printf("  每像素位数: %d\n", varInfo.BitsPerPixel)

	// 打印颜色位字段信息 (关键!)
	fmt.Printf("  颜色格式:\n")
	fmt.Printf("    红色: offset=%d, length=%d\n", varInfo.Red.Offset, varInfo.Red.Length)
	fmt.Printf("    绿色: offset=%d, length=%d\n", varInfo.Green.Offset, varInfo.Green.Length)
	fmt.Printf("    蓝色: offset=%d, length=%d\n", varInfo.Blue.Offset, varInfo.Blue.Length)
	fmt.Printf("    透明度: offset=%d, length=%d\n", varInfo.Transp.Offset, varInfo.Transp.Length)

	// 3. 获取固定屏幕信息 (包含行长度等)
	var fixInfo FixScreenInfo
	_, _, errno = syscall.Syscall(
		syscall.SYS_IOCTL,
		fb.Fd(),
		FBIOGET_FSCREENINFO,
		uintptr(unsafe.Pointer(&fixInfo)),
	)

	if errno != 0 {
		fmt.Printf("❌ 获取固定信息失败 (errno=%d)\n", errno)
		return
	}

	fmt.Printf("✓ 固定信息获取成功\n")
	fmt.Printf("  行长度: %d 字节\n", fixInfo.LineLength)
	fmt.Printf("  内存大小: %d 字节 (%.2f MB)\n", fixInfo.SMemLen, float64(fixInfo.SMemLen)/(1024*1024))
	fmt.Printf("  缓冲区类型: %d\n", fixInfo.Type)

	// 4. 计算关键参数
	visibleWidth := int(varInfo.XRes)         // 720
	visibleHeight := int(varInfo.YRes)        // 576
	virtualHeight := int(varInfo.YResVirtual) // 1152
	bpp := int(varInfo.BitsPerPixel)          // 32

	// 使用实际的行长度
	stride := int(fixInfo.LineLength)
	if stride == 0 {
		// 如果line_length为0，则计算
		stride = int(varInfo.XResVirtual) * (bpp / 8) // 720 * 4 = 2880
	}

	// 计算整个 framebuffer 大小
	fbSize := stride * virtualHeight // 2880 * 1152 = 3317760

	// 计算可见区域的起始偏移 (关键修正!)
	byteOffset := int(varInfo.YOffset) * stride
	byteOffset += int(varInfo.XOffset) * (bpp / 8)

	fmt.Printf("\n📐 计算参数:\n")
	fmt.Printf("  可见区域: %dx%d\n", visibleWidth, visibleHeight)
	fmt.Printf("  虚拟高度: %d (双缓冲)\n", virtualHeight)
	fmt.Printf("  行跨距: %d 字节\n", stride)
	fmt.Printf("  Framebuffer总大小: %d 字节\n", fbSize)
	fmt.Printf("  可见数据起始偏移: %d 字节 (yoffset=%d行)\n", byteOffset, varInfo.YOffset)

	// 5. 映射整个framebuffer到内存
	fmt.Printf("\n🗺️  映射内存...\n")
	mmapData, err := syscall.Mmap(
		int(fb.Fd()),
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
		fmt.Printf("   偏移: %d, 需要: %d, 实际: %d\n", byteOffset, byteOffset+visibleHeight*stride, len(mmapData))
		return
	}

	// 7. 创建RGBA图像并转换数据
	fmt.Printf("\n🎨 创建图像 (格式转换)...\n")

	// 根据透明度长度判断是否有Alpha通道
	hasAlpha := varInfo.Transp.Length > 0
	fmt.Printf("  Alpha通道: %v\n", hasAlpha)

	// 创建图像
	img := image.NewRGBA(image.Rect(0, 0, visibleWidth, visibleHeight))

	// 从正确偏移开始读取数据
	src := mmapData[byteOffset:]

	// 根据文章分析和你的设备信息进行格式转换
	// 你的设备: red.offset=16, green.offset=0, blue.offset=8, transp.length=0
	// 格式为: G(0-7), B(8-15), R(16-23), X(24-31)
	convertFramebufferToRGBA(src, img.Pix, visibleWidth, visibleHeight, stride, varInfo)

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

	if err = png.Encode(file, img); err != nil {
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
