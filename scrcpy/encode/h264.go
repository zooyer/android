package encode

import (
	"bytes"
	"image"
	"sync"

	"github.com/chai2010/webp"
	"github.com/pixiv/go-libjpeg/jpeg"
	"golang.org/x/image/vp8"
)

// 软件H.264编码器（Go实现）
type H264Encoder struct {
	width        int
	height       int
	fps          int
	bitrate      int
	gop          int // 关键帧间隔
	quality      int
	frameCount   int
	lastKeyFrame int
	mutex        sync.Mutex
	bufferPool   *sync.Pool
}

func NewH264Encoder(width, height, fps, bitrate int) *H264Encoder {
	// 计算GOP大小（关键帧间隔）
	gop := fps * 2 // 每2秒一个关键帧

	// 计算质量参数（1-51，越低质量越好）
	quality := 23
	if bitrate < 2000000 {
		quality = 28 // 低比特率使用更高压缩
	}

	return &H264Encoder{
		width:   width,
		height:  height,
		fps:     fps,
		bitrate: bitrate,
		gop:     gop,
		quality: quality,
		bufferPool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, width*height*3/2) // YUV420大小
			},
		},
	}
}

func (e *H264Encoder) Encode(rgbaData []byte, width, height int) ([]byte, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.frameCount++

	// 检查是否需要关键帧
	needKeyFrame := (e.frameCount-e.lastKeyFrame >= e.gop) || e.frameCount == 1

	// 转换为YUV420
	yuvBuf := e.bufferPool.Get().([]byte)
	defer e.bufferPool.Put(yuvBuf)

	e.rgbaToYUV420(rgbaData, yuvBuf, width, height)

	// 简单压缩（实际项目应该集成x264或openh264）
	// 这里使用JPEG作为示例，实际应该用H.264
	return e.encodeJPEG(yuvBuf, width, height, needKeyFrame)
}

func (e *H264Encoder) rgbaToYUV420(rgba, yuv []byte, width, height int) {
	// 简单RGB转YUV（实际应该优化）
	ySize := width * height
	uvSize := ySize / 4

	yIndex := 0
	uIndex := ySize
	vIndex := ySize + uvSize

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := (y*width + x) * 4

			r := float32(rgba[i])
			g := float32(rgba[i+1])
			b := float32(rgba[i+2])

			// YUV转换公式
			yVal := 0.299*r + 0.587*g + 0.114*b
			uVal := -0.169*r - 0.331*g + 0.5*b + 128
			vVal := 0.5*r - 0.419*g - 0.081*b + 128

			yuv[yIndex] = byte(yVal)
			yIndex++

			// 对UV进行下采样（每2x2像素采一个）
			if y%2 == 0 && x%2 == 0 {
				yuv[uIndex] = byte(uVal)
				yuv[vIndex] = byte(vVal)
				uIndex++
				vIndex++
			}
		}
	}
}

func (e *H264Encoder) encodeJPEG(yuv []byte, width, height int, keyFrame bool) ([]byte, error) {
	// 将YUV转换回RGB进行JPEG编码
	rgba := make([]byte, width*height*4)
	e.yuv420ToRGBA(yuv, rgba, width, height)

	img := &image.RGBA{
		Pix:    rgba,
		Stride: width * 4,
		Rect:   image.Rect(0, 0, width, height),
	}

	var buf bytes.Buffer
	opts := &jpeg.EncoderOptions{
		Quality: e.quality,
	}

	if err := jpeg.Encode(&buf, img, opts); err != nil {
		return nil, err
	}

	// 添加简单的帧头
	frame := e.addFrameHeader(buf.Bytes(), keyFrame)

	return frame, nil
}

func (e *H264Encoder) addFrameHeader(data []byte, keyFrame bool) []byte {
	header := make([]byte, 8)

	// 帧类型：0=关键帧，1=P帧
	if keyFrame {
		header[0] = 0
		e.lastKeyFrame = e.frameCount
	} else {
		header[0] = 1
	}

	// 时间戳（毫秒）
	timestamp := uint32(e.frameCount * 1000 / e.fps)
	header[1] = byte(timestamp >> 24)
	header[2] = byte(timestamp >> 16)
	header[3] = byte(timestamp >> 8)
	header[4] = byte(timestamp)

	// 数据长度
	length := uint32(len(data))
	header[5] = byte(length >> 16)
	header[6] = byte(length >> 8)
	header[7] = byte(length)

	return append(header, data...)
}

func (e *H264Encoder) yuv420ToRGBA(yuv, rgba []byte, width, height int) {
	// 简化的YUV转RGB
	ySize := width * height

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			yIndex := y*width + x
			uvIndex := ySize + (y/2)*(width/2) + (x / 2)

			yVal := float32(yuv[yIndex])
			uVal := float32(yuv[uvIndex]) - 128
			vVal := float32(yuv[uvIndex+ySize/4]) - 128

			// YUV转RGB公式
			r := yVal + 1.402*vVal
			g := yVal - 0.344*uVal - 0.714*vVal
			b := yVal + 1.772*uVal

			// 限制范围
			r = clamp(r, 0, 255)
			g = clamp(g, 0, 255)
			b = clamp(b, 0, 255)

			rgbaIndex := (y*width + x) * 4
			rgba[rgbaIndex] = byte(r)
			rgba[rgbaIndex+1] = byte(g)
			rgba[rgbaIndex+2] = byte(b)
			rgba[rgbaIndex+3] = 255
		}
	}
}

func clamp(value, min, max float32) float32 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func (e *H264Encoder) Close() error {
	return nil
}
