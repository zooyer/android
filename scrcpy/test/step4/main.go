// simple_solution.go - 双缓冲解决方案
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"syscall"
)

func main() {
	fmt.Println("双缓冲帧缓冲区解决方案")

	width := 720
	height := 576
	stride := width * 4               // 2880
	frameSize := stride * height      // 1658880 (一帧)
	doubleBufferSize := frameSize * 2 // 3317760 (双缓冲)

	// 打开设备
	fd, err := syscall.Open("/dev/graphics/fb0", syscall.O_RDWR, 0)
	if err != nil {
		fmt.Printf("打开失败: %v\n", err)
		return
	}
	defer syscall.Close(fd)

	// 映射双缓冲大小
	data, err := syscall.Mmap(fd, 0, doubleBufferSize, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		fmt.Printf("映射失败: %v\n", err)
		return
	}
	defer syscall.Munmap(data)

	fmt.Printf("映射 %d 字节成功\n", len(data))

	// 尝试两个缓冲区
	fmt.Println("\n尝试两个缓冲区:")

	// 第一帧（偏移0）
	img1 := extractImage(data[0:], width, height, stride, "第一帧")
	if img1 != nil {
		saveImage(img1, "/sdcard/frame1.png")
	}

	// 第二帧（偏移frameSize）
	if len(data) >= doubleBufferSize {
		img2 := extractImage(data[frameSize:], width, height, stride, "第二帧")
		if img2 != nil {
			saveImage(img2, "/sdcard/frame2.png")
		}
	}

	// 尝试中间偏移（可能正在显示的帧）
	for offset := 0; offset < doubleBufferSize-frameSize; offset += stride {
		img := extractImage(data[offset:], width, height, stride,
			fmt.Sprintf("偏移%d", offset))
		if img != nil {
			saveImage(img, fmt.Sprintf("/sdcard/offset_%d.png", offset))
		}
	}

	fmt.Println("\n检查生成的图片，找到正确的那个")
}

func extractImage(data []byte, width, height, stride int, name string) *image.RGBA {
	if len(data) < stride*height {
		return nil
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// 检查这个数据块是否有图像特征
	empty := true
	for i := 0; i < 100 && i < len(data); i++ {
		if data[i] != 0 && data[i] != 0xFF {
			empty = false
			break
		}
	}

	if empty {
		fmt.Printf("  %s: 可能是空缓冲区\n", name)
		return nil
	}

	fmt.Printf("  %s: 处理中...\n", name)

	// 尝试多种格式
	success := false
	success = success || tryFormatBGRA(data, img.Pix, width, height, stride)
	success = success || tryFormatRGBA(data, img.Pix, width, height, stride)
	success = success || tryFormatARGB(data, img.Pix, width, height, stride)

	if success {
		fmt.Printf("  %s: ✓ 可能有效\n", name)
		return img
	}

	return nil
}

func tryFormatBGRA(src, dst []byte, w, h, s int) bool {
	// 检查是否为有效的BGRA数据
	validCount := 0
	totalCheck := w * h / 10 // 检查10%的像素

	for i := 0; i < totalCheck && i*4 < len(src); i++ {
		idx := i * 4
		// BGRA数据通常不会是全0或全FF
		if src[idx] != 0 && src[idx] != 0xFF &&
			src[idx+1] != 0 && src[idx+1] != 0xFF &&
			src[idx+2] != 0 && src[idx+2] != 0xFF {
			validCount++
		}
	}

	if validCount < totalCheck/2 {
		return false
	}

	// 转换
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

	return true
}

func tryFormatRGBA(src, dst []byte, w, h, s int) bool {
	// 简单复制
	for y := 0; y < h; y++ {
		srcRow := src[y*s:]
		dstRow := dst[y*w*4:]
		copy(dstRow, srcRow[:w*4])
	}
	return true
}

func tryFormatARGB(src, dst []byte, w, h, s int) bool {
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
	return true
}

func saveImage(img *image.RGBA, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("创建文件失败: %v\n", err)
		return
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		fmt.Printf("保存PNG失败: %v\n", err)
		return
	}

	fmt.Printf("  保存到: %s\n", filename)
}
