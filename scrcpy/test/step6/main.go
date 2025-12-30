package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
)

func main() {
	fmt.Println("=== 直接内存搜索截图方案 ===")

	// 直接读取整个可能的帧缓冲区（假设最大为10MB）
	const MAX_SIZE = 10 * 1024 * 1024

	// 1. 直接读取设备文件（最简单直接的方法）
	data, err := os.ReadFile("/dev/graphics/fb0")
	if err != nil {
		fmt.Printf("直接读取失败: %v\n", err)
		return
	}

	if len(data) == 0 {
		fmt.Println("读取到0字节数据")
		return
	}

	fmt.Printf("读取到 %d 字节数据 (%.2f MB)\n", len(data), float64(len(data))/(1024*1024))

	// 2. 分析数据特征，寻找图像
	width, height := 720, 576
	stride := 2880 // 720 * 4

	// 尝试在不同偏移处查找图像
	imageFound := false

	// 从多个可能偏移开始尝试
	offsets := []int{
		0,
		4096,    // 4KB偏移
		8192,    // 8KB
		16384,   // 16KB
		32768,   // 32KB
		65536,   // 64KB
		131072,  // 128KB
		262144,  // 256KB
		524288,  // 512KB
		1048576, // 1MB
	}

	for i, offset := range offsets {
		if offset+height*stride > len(data) {
			continue
		}

		fmt.Printf("\n尝试偏移 %d (%d): ", i+1, offset)

		// 检查这个区域是否有图像特征
		if isLikelyImageData(data[offset:offset+1000], stride) {
			fmt.Printf("✓ 发现图像特征\n")

			// 提取并保存图像
			if saveImageFromOffset(data, offset, width, height, stride, i) {
				imageFound = true
				break
			}
		} else {
			fmt.Printf("✗ 无图像特征\n")
		}
	}

	if !imageFound {
		fmt.Println("\n❌ 未找到明显的图像数据")
		fmt.Println("\n建议尝试：")
		fmt.Println("1. 检查是否需要激活显示：echo 1 > /sys/class/graphics/fb0/activate")
		fmt.Println("2. 尝试其他设备：/dev/fb0, /dev/graphics/fb1")
		fmt.Println("3. 使用备用方案：adb shell screencap -p /sdcard/screen.png")
	}
}

func isLikelyImageData(data []byte, stride int) bool {
	if len(data) < 100 {
		return false
	}

	// 检查不是全0或全FF
	allZero := true
	allFF := true
	hasPattern := false

	for i := 0; i < 100; i++ {
		if data[i] != 0 {
			allZero = false
		}
		if data[i] != 0xFF {
			allFF = false
		}
		// 检查是否有重复模式（图像行之间可能有重复）
		if i >= stride && data[i] == data[i-stride] {
			hasPattern = true
		}
	}

	// 图像数据通常不是全0或全FF，而且可能有模式
	return !allZero && !allFF && hasPattern
}

func saveImageFromOffset(data []byte, offset, width, height, stride, index int) bool {
	if offset+height*stride > len(data) {
		return false
	}

	// 创建图像
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// 尝试多种格式
	formats := []string{"BGRA", "RGBA", "ARGB", "ABGR"}

	for _, format := range formats {
		fmt.Printf("  尝试%s格式: ", format)

		switch format {
		case "BGRA":
			convertBGRA(data[offset:], img.Pix, width, height, stride)
		case "RGBA":
			convertRGBA(data[offset:], img.Pix, width, height, stride)
		case "ARGB":
			convertARGB(data[offset:], img.Pix, width, height, stride)
		case "ABGR":
			convertABGR(data[offset:], img.Pix, width, height, stride)
		}

		filename := fmt.Sprintf("/sdcard/test_%d_%s.png", index, format)
		if saveImage(img, filename) {
			fmt.Printf("✓ 保存到 %s\n", filename)
			return true
		}
	}

	return false
}

// 格式转换函数
func convertBGRA(src, dst []byte, w, h, s int) {
	for y := 0; y < h; y++ {
		srcRow := src[y*s:]
		dstRow := dst[y*w*4:]

		for x := 0; x < w; x++ {
			b := srcRow[x*4]
			g := srcRow[x*4+1]
			r := srcRow[x*4+2]
			a := srcRow[x*4+3]

			dstRow[x*4] = r
			dstRow[x*4+1] = g
			dstRow[x*4+2] = b
			dstRow[x*4+3] = a
		}
	}
}

func convertRGBA(src, dst []byte, w, h, s int) {
	for y := 0; y < h; y++ {
		srcRow := src[y*s:]
		dstRow := dst[y*w*4:]
		copy(dstRow, srcRow[:w*4])
	}
}

func convertARGB(src, dst []byte, w, h, s int) {
	for y := 0; y < h; y++ {
		srcRow := src[y*s:]
		dstRow := dst[y*w*4:]

		for x := 0; x < w; x++ {
			a := srcRow[x*4]
			r := srcRow[x*4+1]
			g := srcRow[x*4+2]
			b := srcRow[x*4+3]

			dstRow[x*4] = r
			dstRow[x*4+1] = g
			dstRow[x*4+2] = b
			dstRow[x*4+3] = a
		}
	}
}

func convertABGR(src, dst []byte, w, h, s int) {
	for y := 0; y < h; y++ {
		srcRow := src[y*s:]
		dstRow := dst[y*w*4:]

		for x := 0; x < w; x++ {
			a := srcRow[x*4]
			b := srcRow[x*4+1]
			g := srcRow[x*4+2]
			r := srcRow[x*4+3]

			dstRow[x*4] = r
			dstRow[x*4+1] = g
			dstRow[x*4+2] = b
			dstRow[x*4+3] = a
		}
	}
}

func saveImage(img *image.RGBA, filename string) bool {
	file, err := os.Create(filename)
	if err != nil {
		return false
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		return false
	}

	return true
}
