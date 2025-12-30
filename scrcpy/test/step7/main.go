package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"
	"syscall"
)

func main() {
	fmt.Println("=== Mali显示驱动截图测试 ===")

	// 尝试所有fb设备
	for i := 0; i < 8; i++ {
		device := fmt.Sprintf("/dev/graphics/fb%d", i)
		fmt.Printf("\n尝试设备: %s\n", device)

		// 尝试打开
		fd, err := syscall.Open(device, syscall.O_RDONLY, 0)
		if err != nil {
			fmt.Printf("  打开失败: %v\n", err)
			continue
		}

		// 尝试读取一小部分
		buf := make([]byte, 4096)
		n, err := syscall.Read(fd, buf)
		syscall.Close(fd)

		if err != nil {
			fmt.Printf("  读取失败: %v\n", err)
			continue
		}

		fmt.Printf("  可读取 %d 字节\n", n)

		// 分析数据
		analyzeBuffer(buf[:n], device)

		// 如果看起来像图像数据，尝试截图
		if isLikelyImage(buf[:n]) {
			fmt.Printf("  ✓ 可能是图像数据，尝试截图...\n")
			tryCapture(device)
		}
	}
}

func analyzeBuffer(data []byte, device string) {
	if len(data) == 0 {
		return
	}

	// 检查特征
	zeroCount := 0
	ffCount := 0
	nonZeroFF := 0

	for i := 0; i < len(data) && i < 100; i++ {
		if data[i] == 0 {
			zeroCount++
		} else if data[i] == 0xFF {
			ffCount++
		} else {
			nonZeroFF++
		}
	}

	fmt.Printf("  前100字节: 0x00=%d, 0xFF=%d, 其他=%d\n", zeroCount, ffCount, nonZeroFF)

	// 如果是ARM指令特征
	if len(data) >= 4 {
		// 检查是否是指令 (常见的ARM指令操作码)
		opcode := (uint32(data[3]) << 24) | (uint32(data[2]) << 16) |
			(uint32(data[1]) << 8) | uint32(data[0])

		// ARM指令常见模式
		isInstruction := false
		if (opcode & 0x0F000000) == 0x0A000000 { // B指令
			isInstruction = true
		} else if (opcode & 0x0FF00000) == 0x03200000 { // 常见数据操作
			isInstruction = true
		}

		if isInstruction {
			fmt.Printf("  ⚠ 可能是ARM指令数据\n")
		}
	}

	// 保存原始数据用于分析
	filename := fmt.Sprintf("/sdcard/%s_raw.bin", strings.ReplaceAll(device, "/", "_"))
	os.WriteFile(filename, data[:min(len(data), 65536)], 0644)
	fmt.Printf("  原始数据保存到: %s\n", filename)
}

func isLikelyImage(data []byte) bool {
	if len(data) < 100 {
		return false
	}

	// 图像数据通常不是全0或全FF，且有一定的分布
	nonUniform := 0
	prev := data[0]

	for i := 1; i < 100 && i < len(data); i++ {
		if data[i] != prev {
			nonUniform++
		}
		prev = data[i]
	}

	// 如果变化足够多，可能是图像
	return nonUniform > 30
}

func tryCapture(device string) {
	// 尝试读取更多数据
	data, err := os.ReadFile(device)
	if err != nil {
		fmt.Printf("    读取失败: %v\n", err)
		return
	}

	if len(data) < 720*576*4 {
		fmt.Printf("    数据不足: %d < %d\n", len(data), 720*576*4)
		return
	}

	// 尝试不同格式和偏移
	width, height := 720, 576
	stride := 2880

	offsets := []int{0, 4096, 8192, 16384, 32768}
	formats := []struct {
		name    string
		convert func(src, dst []byte, w, h, s int)
	}{
		{"BGRA", convertBGRA},
		{"RGBA", convertRGBA},
		{"ARGB", convertARGB},
	}

	for _, offset := range offsets {
		for _, format := range formats {
			if offset+height*stride <= len(data) {
				img := image.NewRGBA(image.Rect(0, 0, width, height))
				format.convert(data[offset:], img.Pix, width, height, stride)

				filename := fmt.Sprintf("/sdcard/capture_%s_%d_%s.png",
					strings.ReplaceAll(device, "/", "_"), offset, format.name)

				if saveImage(img, filename) {
					fmt.Printf("    截图保存到: %s\n", filename)
				}
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
