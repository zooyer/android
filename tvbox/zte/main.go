package main

import (
	"io"
	"log"
	"net"
)

func main() {
	// STB协议解析
	listener, err := net.Listen("tcp", ":10042")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}

		go func(conn net.Conn) {
			defer conn.Close()
			var buf = make([]byte, 8)
			n, err := conn.Read(buf)
			if err != nil {
				log.Println("read error:", err.Error())
				return
			}
			buf = make([]byte, 6)
			n, err = conn.Read(buf)
			if err != nil {
				log.Println("read error:", err.Error())
				return
			}
			log.Println("password:", string(buf[:n]))
			conn.Write([]byte{0x00, 0x02, 0x34, 0x33, 0x00, 0x00, 0x2E, 0xE1})
			buf = make([]byte, 8)
			n, err = conn.Read(buf)
			if err != nil {
				log.Println("read error:", err.Error())
				return
			}
			buf = make([]byte, 4)
			n, err = conn.Read(buf)
			if err != nil {
				log.Println("read error:", err.Error())
				return
			}
			conn.Write([]byte{0x10, 0x21, 0x2E, 0x30, 0x00, 0x00, 0x00, 0x04})
			conn.Write([]byte("V3.0"))
			buf = make([]byte, 96)
			n, err = io.ReadFull(conn, buf)
			if err != nil {
				log.Println("read error:", err.Error())
				return
			}
			conn.Write(buf[:n])
		}(conn)
	}
}
