/*
 * @Author: lizongrong
 * @since: 2019-04-04 17:41:17
 * @lastTime: 2019-04-04 22:53:16
 */
package transport

import (
	"io"
	"net"
)

type Transport interface {
	Dial(network, addr string) error
	io.ReadWriteCloser
	RemoteAddr() net.Addr
	LocalAddr() net.Addr
}

type Socket struct {
	conn net.Conn
}

func (s *Socket) Dial(network, addr string) error {
	conn, err := net.Dial(network, addr)
	s.conn = conn
	return err
}

func (s *Socket) Read(p []byte) (n int, err error) {
	return s.conn.Read(p)
}

func (s *Socket) Write(p []byte) (n int, err error) {
	return s.conn.Write(p)
}

func (s *Socket) Close() error {
	return s.conn.Close()
}

func (s *Socket) RemoteAddr() net.Addr {
	return s.conn.RemoteAddr()
}

func (s Socket) LocalAddr() net.Addr {
	return s.conn.LocalAddr()
}

type ServerTransport interface {
	Listen(network, addr string) error
	Accept() (Transport, error)
}

type ServerSocket struct {
	ln net.Listener
}

func (s *ServerSocket) Listen(network, addr string) error {
	ln, err := net.Listen(network, addr)
	s.ln = ln
	return err
}

func (s *ServerSocket) Accept() (Transport, error) {
	conn, err := s.ln.Accept()
	return &Socket{conn: conn}, err
}

func (s *ServerSocket) Close() error {
	return s.ln.Close()
}
