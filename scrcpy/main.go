package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/scrcpy-go/capture"
	"github.com/scrcpy-go/control"
	"github.com/scrcpy-go/encode"
	"github.com/scrcpy-go/network"
	"github.com/scrcpy-go/utils"
)

type Config struct {
	Port        int
	BitRate     int
	MaxFPS      int
	MaxWidth    int
	MaxHeight   int
	StayAwake   bool
	ForwardAll  bool
	Verbose     bool
	Record      string
	ShowTouches bool
}

type Server struct {
	config     *Config
	capturer   capture.Capturer
	encoder    encode.Encoder
	server     *network.VideoServer
	controller *control.Controller
	stopChan   chan struct{}
	frameChan  chan []byte
	stats      *utils.Stats
}

func parseFlags() *Config {
	config := &Config{}

	flag.IntVar(&config.Port, "port", 8888, "服务端口")
	flag.IntVar(&config.BitRate, "bit-rate", 8000000, "比特率 (bps)")
	flag.IntVar(&config.MaxFPS, "max-fps", 60, "最大帧率")
	flag.IntVar(&config.MaxWidth, "max-width", 1920, "最大宽度")
	flag.IntVar(&config.MaxHeight, "max-height", 1080, "最大高度")
	flag.BoolVar(&config.StayAwake, "stay-awake", true, "保持屏幕唤醒")
	flag.BoolVar(&config.ForwardAll, "forward-all", true, "转发所有剪贴板")
	flag.BoolVar(&config.Verbose, "verbose", false, "详细日志")
	flag.StringVar(&config.Record, "record", "", "录制到文件")
	flag.BoolVar(&config.ShowTouches, "show-touches", false, "显示触摸点")

	flag.Parse()
	return config
}

func main() {
	fmt.Println("scrcpy-go - Android远程控制服务器")
	fmt.Printf("Go版本: %s, CPU核心数: %d\n", runtime.Version(), runtime.NumCPU())

	config := parseFlags()

	server := &Server{
		config:    config,
		stopChan:  make(chan struct{}),
		frameChan: make(chan []byte, 60),
		stats:     utils.NewStats(),
	}

	// 初始化
	if err := server.init(); err != nil {
		log.Fatalf("初始化失败: %v", err)
	}

	// 启动服务
	go server.run()

	// 信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 统计输出
	go server.printStats()

	<-sigChan
	log.Println("收到停止信号...")

	server.cleanup()
	log.Println("服务已停止")
}

func (s *Server) init() error {
	var err error

	log.Println("初始化屏幕捕获器...")
	s.capturer, err = capture.NewFramebufferCapturer()
	if err != nil {
		return fmt.Errorf("创建捕获器失败: %v", err)
	}

	width, height := s.capturer.Resolution()
	log.Printf("屏幕分辨率: %dx%d", width, height)

	// 调整尺寸
	scaledWidth, scaledHeight := s.scaleDimensions(width, height)

	log.Println("初始化编码器...")
	s.encoder = encode.NewH264Encoder(
		scaledWidth,
		scaledHeight,
		s.config.MaxFPS,
		s.config.BitRate,
	)

	log.Println("初始化网络服务器...")
	s.server = network.NewVideoServer(s.config.Port, s.config.Verbose)

	log.Println("初始化控制器...")
	s.controller = control.NewController(s.config)

	log.Println("初始化完成!")
	return nil
}

func (s *Server) scaleDimensions(width, height int) (int, int) {
	if width <= s.config.MaxWidth && height <= s.config.MaxHeight {
		return width, height
	}

	ratio := float64(width) / float64(height)

	if width > s.config.MaxWidth {
		width = s.config.MaxWidth
		height = int(float64(width) / ratio)
	}

	if height > s.config.MaxHeight {
		height = s.config.MaxHeight
		width = int(float64(height) * ratio)
	}

	// 确保是偶数（H.264要求）
	width = width & ^1
	height = height & ^1

	return width, height
}

func (s *Server) run() {
	// 启动网络服务器
	go s.server.Start()

	// 处理控制消息
	go s.handleControlMessages()

	// 主捕获循环
	frameInterval := time.Second / time.Duration(s.config.MaxFPS)
	ticker := time.NewTicker(frameInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.captureAndStream()
		}
	}
}

func (s *Server) captureAndStream() {
	start := time.Now()

	// 捕获屏幕
	rawData, width, height, err := s.capturer.Capture()
	if err != nil {
		s.stats.Increment("capture_errors")
		log.Printf("捕获失败: %v", err)
		return
	}

	s.stats.RecordDuration("capture_time", time.Since(start))
	captureTime := time.Now()

	// 编码
	encoded, err := s.encoder.Encode(rawData, width, height)
	if err != nil {
		s.stats.Increment("encode_errors")
		log.Printf("编码失败: %v", err)
		return
	}

	s.stats.RecordDuration("encode_time", time.Since(captureTime))
	encodeTime := time.Now()

	// 发送到所有客户端
	s.server.BroadcastFrame(encoded)

	s.stats.RecordDuration("send_time", time.Since(encodeTime))
	s.stats.RecordDuration("total_time", time.Since(start))
	s.stats.Increment("frames_sent")
}

func (s *Server) handleControlMessages() {
	for {
		select {
		case <-s.stopChan:
			return
		case msg := <-s.server.ControlChannel():
			s.processControlMessage(msg)
		}
	}
}

func (s *Server) processControlMessage(msg network.ControlMessage) {
	switch msg.Type {
	case network.MsgTypeKeyEvent:
		s.controller.InjectKeyEvent(msg.Data)
	case network.MsgTypeTouchEvent:
		s.controller.InjectTouchEvent(msg.Data)
	case network.MsgTypeScrollEvent:
		s.controller.InjectScrollEvent(msg.Data)
	case network.MsgTypeClipboard:
		s.controller.SetClipboard(msg.Data)
	case network.MsgTypeSetScreenPower:
		s.controller.SetScreenPower(msg.Data[0] == 1)
	}
}

func (s *Server) printStats() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.stats.Print()
		}
	}
}

func (s *Server) cleanup() {
	close(s.stopChan)

	if s.capturer != nil {
		s.capturer.Close()
	}

	if s.encoder != nil {
		s.encoder.Close()
	}

	if s.server != nil {
		s.server.Stop()
	}
}
