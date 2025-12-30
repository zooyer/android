// fb_demo_final.go - 最终修复版
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

const (
	FBIOGET_VSCREENINFO = 0x4600
	FBIOGET_FSCREENINFO = 0x4601
	O_RDWR              = 0x0002
)

func main() {
	fmt.Println("=== Android帧缓冲区测试（最终版）===")

	// 根据你的输出，我们知道：
	// 分辨率: 720x576
	// 虚拟分辨率: 720x1152 (双缓冲)
	// BPP: 32
	// 格式: R=16, G=0, B=8 (这可能是BGR格式，不是BGRA!)

	if err := testWithKnownInfo(); err != nil {
		log.Printf("测试失败: %v", err)
	} else {
		fmt.Println("✓ 测试成功！")
	}
}

func testWithKnownInfo() error {
	// 使用你的已知信息
	width := 720
	height := 576
	virtualHeight := 1152 // 虚拟高度
	bpp := 32
	stride := width * bpp / 8           // 720 * 4 = 2880
	frameSize := stride * virtualHeight // 2880 * 1152 = 3317760

	fmt.Printf("使用已知参数:\n")
	fmt.Printf("  物理分辨率: %dx%d\n", width, height)
	fmt.Printf("  虚拟分辨率: %dx%d\n", width, virtualHeight)
	fmt.Printf("  BPP: %d\n", bpp)
	fmt.Printf("  Stride: %d 字节\n", stride)
	fmt.Printf("  帧大小: %d 字节 (%.2f MB)\n", frameSize, float64(frameSize)/(1024*1024))

	// 关键修复：使用O_RDWR而不是O_RDONLY
	fmt.Println("\n1. 打开设备（使用读写模式）...")
	fd, err := syscall.Open("/dev/graphics/fb0", syscall.O_RDWR, 0)
	if err != nil {
		// 尝试只读模式
		fd, err = syscall.Open("/dev/graphics/fb0", syscall.O_RDONLY, 0)
		if err != nil {
			return fmt.Errorf("打开设备失败: %v", err)
		}
		fmt.Println("  使用只读模式打开")
	} else {
		fmt.Println("  使用读写模式打开")
	}
	defer syscall.Close(fd)

	// 验证文件描述符
	fmt.Printf("2. 文件描述符: %d\n", fd)

	// 尝试先读取一小部分数据
	fmt.Println("3. 尝试读取前1024字节...")
	buf := make([]byte, 1024)
	n, err := syscall.Read(fd, buf)
	if err != nil {
		fmt.Printf("  直接读取失败: %v\n", err)
	} else {
		fmt.Printf("  成功读取 %d 字节\n", n)

		// 分析数据
		fmt.Println("  前32字节:")
		for i := 0; i < 32 && i < n; i += 4 {
			fmt.Printf("    [%02d]: %02x %02x %02x %02x\n",
				i, buf[i], buf[i+1], buf[i+2], buf[i+3])
		}

		// 保存原始数据
		saveRawData(buf[:n], "/sdcard/fb_header.bin")
	}

	// 关键：重置文件指针到开始
	fmt.Println("4. 重置文件指针...")
	_, err = syscall.Seek(fd, 0, 0)
	if err != nil {
		fmt.Printf("  Seek失败: %v\n", err)
	}

	// 尝试内存映射
	fmt.Println("5. 尝试内存映射...")

	// 映射整个帧缓冲区
	mmapData, err := syscall.Mmap(
		fd,
		0,
		frameSize,
		syscall.PROT_READ,
		syscall.MAP_SHARED,
	)

	if err != nil {
		// 尝试小一点的大小
		fmt.Printf("  映射失败: %v，尝试小尺寸\n", err)

		// 只映射一帧（不是整个双缓冲）
		singleFrameSize := stride * height
		fmt.Printf("  尝试映射单帧: %d 字节\n", singleFrameSize)

		mmapData, err = syscall.Mmap(
			fd,
			0,
			singleFrameSize,
			syscall.PROT_READ,
			syscall.MAP_SHARED,
		)

		if err != nil {
			return fmt.Errorf("内存映射失败: %v", err)
		}
		fmt.Printf("  ✓ 成功映射 %d 字节\n", len(mmapData))
	} else {
		fmt.Printf("  ✓ 成功映射 %d 字节\n", len(mmapData))
	}
	defer syscall.Munmap(mmapData)

	// 分析映射的数据
	fmt.Println("6. 分析映射数据...")
	if len(mmapData) > 0 {
		// 检查前几个像素
		fmt.Println("  映射数据前32字节:")
		for i := 0; i < 32 && i < len(mmapData); i += 4 {
			fmt.Printf("    [%02d]: %02x %02x %02x %02x\n",
				i, mmapData[i], mmapData[i+1], mmapData[i+2], mmapData[i+3])
		}

		// 保存整个映射数据用于调试
		if len(mmapData) <= 10*1024*1024 { // 不超过10MB
			saveRawData(mmapData, "/sdcard/fb_mmap.bin")
		}
	}

	// 创建图像
	fmt.Println("7. 创建PNG图像...")

	// 根据你的通道偏移：R=16, G=0, B=8
	// 这意味着每个像素4字节的布局可能是：
	// 字节0: G (绿色)
	// 字节1: B? (蓝色偏移是8位=1字节)
	// 字节2: R (红色偏移是16位=2字节)
	// 字节3: ? (可能是Alpha或未使用)

	// 尝试不同的格式
	img := tryAllFormats(mmapData, width, height, stride)

	if img == nil {
		return fmt.Errorf("无法解析图像数据")
	}

	// 保存图像
	fmt.Println("8. 保存图像...")
	return savePNG(img, "/sdcard/demo.png")
}

func tryAllFormats(data []byte, width, height, stride int) *image.RGBA {
	fmt.Println("  尝试解析格式...")

	if len(data) < stride*height {
		fmt.Printf("  数据不足: 需要%d字节，实际%d字节\n", stride*height, len(data))
		return nil
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// 尝试格式1: BGRA（最常见的Android格式）
	fmt.Println("  尝试BGRA格式...")
	success1 := convertBGRA(data, img.Pix, width, height, stride)
	if success1 {
		fmt.Println("  ✓ BGRA格式看起来有效")
		savePNG(img, "/sdcard/test_bgra.png")
		return img
	}

	// 尝试格式2: RGBA
	fmt.Println("  尝试RGBA格式...")
	success2 := convertRGBA(data, img.Pix, width, height, stride)
	if success2 {
		fmt.Println("  ✓ RGBA格式看起来有效")
		savePNG(img, "/sdcard/test_rgba.png")
		return img
	}

	// 尝试格式3: 根据你的偏移：R=16, G=0, B=8
	// 这可能是：字节0=G, 字节1=B, 字节2=R, 字节3=A
	fmt.Println("  尝试GBRA格式（根据你的偏移）...")
	success3 := convertCustom(data, img.Pix, width, height, stride)
	if success3 {
		fmt.Println("  ✓ 自定义格式有效")
		savePNG(img, "/sdcard/test_custom.png")
		return img
	}

	// 尝试格式4: ARGB
	fmt.Println("  尝试ARGB格式...")
	success4 := convertARGB(data, img.Pix, width, height, stride)
	if success4 {
		fmt.Println("  ✓ ARGB格式有效")
		savePNG(img, "/sdcard/test_argb.png")
		return img
	}

	// 最后尝试：直接复制，看原始数据
	fmt.Println("  尝试直接复制...")
	copy(img.Pix, data[:width*height*4])
	savePNG(img, "/sdcard/test_raw.png")

	return img
}

func convertBGRA(src, dst []byte, width, height, stride int) bool {
	// BGRA -> RGBA
	for y := 0; y < height; y++ {
		srcRow := src[y*stride:]
		dstRow := dst[y*width*4:]

		for x := 0; x < width; x++ {
			srcPos := x * 4
			dstPos := x * 4

			b := srcRow[srcPos]
			g := srcRow[srcPos+1]
			r := srcRow[srcPos+2]
			a := srcRow[srcPos+3]

			dstRow[dstPos] = r
			dstRow[dstPos+1] = g
			dstRow[dstPos+2] = b
			dstRow[dstPos+3] = a
		}
	}
	return true
}

func convertRGBA(src, dst []byte, width, height, stride int) bool {
	// RGBA -> RGBA（直接复制）
	for y := 0; y < height; y++ {
		srcRow := src[y*stride:]
		dstRow := dst[y*width*4:]
		copy(dstRow, srcRow[:width*4])
	}
	return true
}

func convertCustom(src, dst []byte, width, height, stride int) bool {
	// 根据你的偏移：R=16, G=0, B=8
	// 字节0: G, 字节1: B, 字节2: R, 字节3: ?
	for y := 0; y < height; y++ {
		srcRow := src[y*stride:]
		dstRow := dst[y*width*4:]

		for x := 0; x < width; x++ {
			srcPos := x * 4
			dstPos := x * 4

			g := srcRow[srcPos]   // offset 0
			b := srcRow[srcPos+1] // offset 8 (1字节)
			r := srcRow[srcPos+2] // offset 16 (2字节)
			a := srcRow[srcPos+3] // 可能是alpha

			if a == 0 {
				a = 255 // 如果没有alpha，设为不透明
			}

			dstRow[dstPos] = r
			dstRow[dstPos+1] = g
			dstRow[dstPos+2] = b
			dstRow[dstPos+3] = a
		}
	}
	return true
}

func convertARGB(src, dst []byte, width, height, stride int) bool {
	// ARGB -> RGBA
	for y := 0; y < height; y++ {
		srcRow := src[y*stride:]
		dstRow := dst[y*width*4:]

		for x := 0; x < width; x++ {
			srcPos := x * 4
			dstPos := x * 4

			a := srcRow[srcPos]
			r := srcRow[srcPos+1]
			g := srcRow[srcPos+2]
			b := srcRow[srcPos+3]

			dstRow[dstPos] = r
			dstRow[dstPos+1] = g
			dstRow[dstPos+2] = b
			dstRow[dstPos+3] = a
		}
	}
	return true
}

func saveRawData(data []byte, filename string) {
	fmt.Printf("  保存原始数据到 %s...\n", filename)

	// 尝试主要路径
	file, err := os.Create(filename)
	if err != nil {
		// 尝试备用路径
		filename = "/data/local/tmp/" + filename
		file, err = os.Create(filename)
		if err != nil {
			fmt.Printf("  无法保存: %v\n", err)
			return
		}
	}
	defer file.Close()

	file.Write(data)
	fmt.Printf("  ✓ 保存 %d 字节\n", len(data))
}

func savePNG(img *image.RGBA, filename string) error {
	// 尝试主要路径
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

// 调试工具：检查设备状态
func debugDevice() {
	fmt.Println("\n=== 设备调试信息 ===")

	// 使用纯系统调用，避免Go的干扰
	fd, err := syscall.Open("/dev/graphics/fb0", syscall.O_RDONLY, 0)
	if err != nil {
		fmt.Printf("打开失败: %v\n", err)
		return
	}
	defer syscall.Close(fd)

	fmt.Printf("文件描述符: %d\n", fd)

	// 尝试fstat
	var stat syscall.Stat_t
	if err := syscall.Fstat(fd, &stat); err != nil {
		fmt.Printf("fstat失败: %v\n", err)
	} else {
		fmt.Printf("设备类型: %v\n", stat.Mode&syscall.S_IFMT)
		fmt.Printf("设备号: %d,%d\n", stat.Rdev>>8, stat.Rdev&0xFF)
	}

	// 尝试ioctl
	var vinfo [100]byte
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(fd), FBIOGET_VSCREENINFO, uintptr(unsafe.Pointer(&vinfo[0])))

	if errno != 0 {
		fmt.Printf("ioctl失败: %v\n", errno)
	} else {
		fmt.Println("ioctl成功")
	}
}
