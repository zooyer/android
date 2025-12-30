# client.py - Python客户端示例
import socket
import struct
import cv2
import numpy as np
import threading

class ScrcpyClient:
    def __init__(self, host='localhost', port=8888):
        self.sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.sock.connect((host, port))
        self.running = False

    def start(self):
        # 接收设备信息
        info = self.sock.recv(20)
        width = struct.unpack('<H', info[2:4])[0]
        height = struct.unpack('<H', info[4:6])[0]

        print(f"连接到设备: {width}x{height}")

        self.running = True
        threading.Thread(target=self.receive_frames).start()

    def receive_frames(self):
        while self.running:
            # 读取帧头
            header = self.sock.recv(16)
            if len(header) < 16:
                break

            frame_size = struct.unpack('<I', header[0:4])[0]
            timestamp = struct.unpack('<Q', header[4:12])[0]

            # 读取帧数据
            frame_data = b''
            while len(frame_data) < frame_size:
                chunk = self.sock.recv(min(4096, frame_size - len(frame_data)))
                if not chunk:
                    break
                frame_data += chunk

            if len(frame_data) == frame_size:
                self.process_frame(frame_data)

    def process_frame(self, data):
        # 解码JPEG（根据实际编码格式调整）
        nparr = np.frombuffer(data, np.uint8)
        frame = cv2.imdecode(nparr, cv2.IMREAD_COLOR)

        if frame is not None:
            cv2.imshow('Screen', frame)
            cv2.waitKey(1)

    def send_touch(self, x, y, action):
        # 发送触摸事件
        msg = struct.pack('<BBIHH', 0x02, action, 0, x, y)
        self.sock.send(msg)

    def send_key(self, keycode, pressed):
        # 发送按键事件
        msg = struct.pack('<BBHB', 0x01, keycode, 1 if pressed else 0, 0)
        self.sock.send(msg)

    def close(self):
        self.running = False
        self.sock.close()
        cv2.destroyAllWindows()

if __name__ == '__main__':
    client = ScrcpyClient()
    client.start()