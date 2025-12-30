package capture

import (
	"unsafe"
)

type ColorConverter struct {
	useSIMD bool
}

func NewColorConverter() *ColorConverter {
	return &ColorConverter{
		useSIMD: detectSIMD(),
	}
}

func detectSIMD() bool {
	// 检查CPU是否支持NEON (ARM) 或 SSE (x86)
	// 这里简化处理，实际应该检查CPU特性
	return false
}

func (c *ColorConverter) Convert(src, dst []byte, width, height, stride int, format PixelFormat) {
	switch format {
	case RGB565:
		c.convertRGB565(src, dst, width, height, stride)
	case RGB888:
		c.convertRGB888(src, dst, width, height, stride)
	case BGRA8888:
		c.convertBGRA8888(src, dst, width, height, stride)
	default:
		// 默认复制
		copy(dst, src)
	}
}

func (c *ColorConverter) convertRGB565(src, dst []byte, width, height, stride int) {
	// 手动优化版本
	for y := 0; y < height; y++ {
		srcRow := src[y*stride:]
		dstRow := dst[y*width*4:]

		for x := 0; x < width; x++ {
			// 读取RGB565像素
			pixel := uint16(srcRow[x*2])<<8 | uint16(srcRow[x*2+1])

			// 提取颜色分量
			r := (pixel >> 11) & 0x1F
			g := (pixel >> 5) & 0x3F
			b := pixel & 0x1F

			// 扩展到8位
			r = (r << 3) | (r >> 2)
			g = (g << 2) | (g >> 4)
			b = (b << 3) | (b >> 2)

			// 写入RGBA8888
			dstRow[x*4] = byte(r)
			dstRow[x*4+1] = byte(g)
			dstRow[x*4+2] = byte(b)
			dstRow[x*4+3] = 255 // Alpha
		}
	}
}

func (c *ColorConverter) convertRGB888(src, dst []byte, width, height, stride int) {
	for y := 0; y < height; y++ {
		srcRow := src[y*stride:]
		dstRow := dst[y*width*4:]

		for x := 0; x < width; x++ {
			dstRow[x*4] = srcRow[x*3]     // R
			dstRow[x*4+1] = srcRow[x*3+1] // G
			dstRow[x*4+2] = srcRow[x*3+2] // B
			dstRow[x*4+3] = 255           // A
		}
	}
}

func (c *ColorConverter) convertBGRA8888(src, dst []byte, width, height, stride int) {
	// BGRA -> RGBA 转换
	for y := 0; y < height; y++ {
		srcRow := src[y*stride:]
		dstRow := dst[y*width*4:]

		// 使用unsafe进行批量转换
		srcPtr := (*[1 << 30]uint32)(unsafe.Pointer(&srcRow[0]))
		dstPtr := (*[1 << 30]uint32)(unsafe.Pointer(&dstRow[0]))

		for x := 0; x < width; x++ {
			bgra := srcPtr[x]
			// 交换R和B通道
			r := (bgra >> 16) & 0xFF
			b := bgra & 0xFF
			rgba := (bgra & 0xFF00FF00) | (b << 16) | r
			dstPtr[x] = rgba
		}
	}
}
