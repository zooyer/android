package network

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type ControlMessage struct {
	Type uint8
	Data []byte
}

type Client struct {
	conn     net.Conn
	ch       chan []byte
	quit     chan struct{}
	lastPing time.Time
}

type VideoServer struct {
	port         int
	listener     net.Listener
	clients      map[*Client]bool
	clientsMutex sync.RWMutex
	controlChan  chan ControlMessage
	frameChan    chan []byte
	verbose      bool
	quit         chan struct{}
	wg           sync.WaitGroup
}

func NewVideoServer(port int, verbose bool) *VideoServer {
	return &VideoServer{
		port:        port,
		clients:     make(map[*Client]bool),
		controlChan: make(chan ControlMessage, 100),
		frameChan:   make(chan []byte, 60),
		verbose:     verbose,
		quit:        make(chan struct{}),
	}
}

func (s *VideoServer) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("监听端口失败: %v", err)
	}
	s.listener = listener

	log.Printf("视频服务器启动，监听端口 %d", s.port)

	// 启动广播协程
	go s.broadcastFrames()

	// 启动客户端清理协程
	go s.cleanupClients()

	// 接受连接
	s.wg.Add(1)
	go s.acceptConnections()

	return nil
}

func (s *VideoServer) acceptConnections() {
	defer s.wg.Done()

	for {
		select {
		case <-s.quit:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				if s.verbose {
					log.Printf("接受连接错误: %v", err)
				}
				continue
			}

			s.handleNewConnection(conn)
		}
	}
}

func (s *VideoServer) handleNewConnection(conn net.Conn) {
	client := &Client{
		conn:     conn,
		ch:       make(chan []byte, 30),
		quit:     make(chan struct{}),
		lastPing: time.Now(),
	}

	s.clientsMutex.Lock()
	s.clients[client] = true
	s.clientsMutex.Unlock()

	if s.verbose {
		log.Printf("客户端连接: %s", conn.RemoteAddr())
	}

	// 发送设备信息
	s.sendDeviceInfo(client)

	// 启动读写协程
	s.wg.Add(2)
	go s.readClientMessages(client)
	go s.writeClientMessages(client)
}

func (s *VideoServer) sendDeviceInfo(client *Client) {
	// 设备信息包
	info := make([]byte, 20)

	// 协议版本
	binary.LittleEndian.PutUint16(info[0:2], 1)

	// 屏幕尺寸（占位，实际从捕获器获取）
	binary.LittleEndian.PutUint16(info[2:4], 1920)
	binary.LittleEndian.PutUint16(info[4:6], 1080)

	// 比特率
	binary.LittleEndian.PutUint32(info[6:10], 8000000)

	// 帧率
	binary.LittleEndian.PutUint32(info[10:14], 30)

	// 时间戳
	binary.LittleEndian.PutUint64(info[12:20], uint64(time.Now().UnixNano()))

	client.ch <- info
}

func (s *VideoServer) readClientMessages(client *Client) {
	defer s.wg.Done()
	defer s.removeClient(client)

	buf := make([]byte, 1024)

	for {
		select {
		case <-client.quit:
			return
		default:
			// 设置读取超时
			client.conn.SetReadDeadline(time.Now().Add(10 * time.Second))

			n, err := client.conn.Read(buf)
			if err != nil {
				if err != io.EOF && s.verbose {
					log.Printf("读取客户端数据错误: %v", err)
				}
				return
			}

			// 处理消息
			if n > 0 {
				s.processClientMessage(client, buf[:n])
				client.lastPing = time.Now()
			}
		}
	}
}

func (s *VideoServer) processClientMessage(client *Client, data []byte) {
	if len(data) < 2 {
		return
	}

	msgType := data[0]
	msgData := data[1:]

	switch msgType {
	case 0x01: // 键盘事件
		s.controlChan <- ControlMessage{Type: MsgTypeKeyEvent, Data: msgData}
	case 0x02: // 触摸事件
		s.controlChan <- ControlMessage{Type: MsgTypeTouchEvent, Data: msgData}
	case 0x03: // 滚动事件
		s.controlChan <- ControlMessage{Type: MsgTypeScrollEvent, Data: msgData}
	case 0x04: // 剪贴板
		s.controlChan <- ControlMessage{Type: MsgTypeClipboard, Data: msgData}
	case 0x05: // 心跳
		client.lastPing = time.Now()
	case 0x06: // 屏幕电源控制
		s.controlChan <- ControlMessage{Type: MsgTypeSetScreenPower, Data: msgData}
	}
}

func (s *VideoServer) writeClientMessages(client *Client) {
	defer s.wg.Done()

	for {
		select {
		case <-client.quit:
			return
		case frame := <-client.ch:
			if err := s.sendFrame(client, frame); err != nil {
				if s.verbose {
					log.Printf("发送帧失败: %v", err)
				}
				return
			}
		}
	}
}

func (s *VideoServer) sendFrame(client *Client, frame []byte) error {
	// 添加帧头
	header := make([]byte, 16)
	binary.LittleEndian.PutUint32(header[0:4], uint32(len(frame)))
	binary.LittleEndian.PutUint64(header[4:12], uint64(time.Now().UnixNano()))

	// 发送帧头
	if _, err := client.conn.Write(header); err != nil {
		return err
	}

	// 发送帧数据
	if _, err := client.conn.Write(frame); err != nil {
		return err
	}

	return nil
}

func (s *VideoServer) broadcastFrames() {
	for {
		select {
		case <-s.quit:
			return
		case frame := <-s.frameChan:
			s.clientsMutex.RLock()
			for client := range s.clients {
				select {
				case client.ch <- frame:
					// 发送成功
				default:
					// 通道满，跳过此客户端
					if s.verbose {
						log.Println("客户端通道满，跳过帧")
					}
				}
			}
			s.clientsMutex.RUnlock()
		}
	}
}

func (s *VideoServer) cleanupClients() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.quit:
			return
		case <-ticker.C:
			s.clientsMutex.Lock()
			for client := range s.clients {
				if time.Since(client.lastPing) > 60*time.Second {
					if s.verbose {
						log.Printf("客户端超时: %s", client.conn.RemoteAddr())
					}
					s.removeClient(client)
				}
			}
			s.clientsMutex.Unlock()
		}
	}
}

func (s *VideoServer) removeClient(client *Client) {
	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	if _, exists := s.clients[client]; exists {
		delete(s.clients, client)
		close(client.quit)
		client.conn.Close()

		if s.verbose {
			log.Printf("客户端断开: %s", client.conn.RemoteAddr())
		}
	}
}

func (s *VideoServer) BroadcastFrame(frame []byte) {
	select {
	case s.frameChan <- frame:
		// 发送成功
	default:
		// 通道满，丢弃最旧的帧
		select {
		case <-s.frameChan:
			s.frameChan <- frame
		default:
		}
	}
}

func (s *VideoServer) ControlChannel() <-chan ControlMessage {
	return s.controlChan
}

func (s *VideoServer) Stop() {
	close(s.quit)

	if s.listener != nil {
		s.listener.Close()
	}

	// 关闭所有客户端
	s.clientsMutex.Lock()
	for client := range s.clients {
		close(client.quit)
		client.conn.Close()
		delete(s.clients, client)
	}
	s.clientsMutex.Unlock()

	s.wg.Wait()
	log.Println("视频服务器已停止")
}
