#!/usr/bin/env python3
"""Mock NDC server untuk test lokal."""
import socket
import sys

HOST = '127.0.0.1'
PORT = 12355


class MockNDCServer:
    def __init__(self, host=HOST, port=PORT):
        self.host = host
        self.port = port
        self.sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        self.sock.bind((self.host, self.port))
        self.sock.listen(1)

    def run(self):
        print(f'[server] listening on {self.host}:{self.port}')
        conn, addr = self.sock.accept()
        print(f'[server] client connected from {addr}')
        with conn:
            while True:
                data = conn.recv(4096)
                if not data:
                    print('[server] client disconnected')
                    break
                print(f'[server] raw hex  : {data.hex()}')
                print(f'[server] as string: {data!r}')
                # Echo balik supaya simulator menerima response
                conn.sendall(b'\x02reply\x03')


if __name__ == '__main__':
    host = sys.argv[1] if len(sys.argv) > 1 else HOST
    port = int(sys.argv[2]) if len(sys.argv) > 2 else PORT
    MockNDCServer(host, port).run()
