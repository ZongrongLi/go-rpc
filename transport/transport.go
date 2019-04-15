/*
 * File: transport.go
 * Project: transport
 * File Created: Friday, 5th April 2019 12:00:35 am
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Friday, 5th April 2019 4:47:41 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * null lizongrong - 2019
 */
package transport

import (
	"io"
	"net"
	"time"

	"github.com/golang/glog"
)

type TransportType byte

const TCPTransport TransportType = iota

type Transport interface {
	Dial(network, addr string, option DialOption) error
	io.ReadWriteCloser
	RemoteAddr() net.Addr
	LocalAddr() net.Addr
}

type Socket struct {
	conn net.Conn
}

type DialOption struct {
	Timeout time.Duration
}

func (s *Socket) Dial(network, addr string, option DialOption) error {
	conn, err := net.DialTimeout(network, addr, option.Timeout)
	if err != nil {
		glog.Error("Dial failed: ", err)
	}
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
	glog.Info("closed")
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
	io.Closer
}

type ServerSocket struct {
	ln net.Listener
}

func (s *ServerSocket) Listen(network, addr string) error {
	ln, err := net.Listen(network, addr)
	if err != nil {
		glog.Error("Listen failed network,addr: ", network, addr, err)
	}
	s.ln = ln
	return err
}

func (s *ServerSocket) Accept() (Transport, error) {
	conn, err := s.ln.Accept()
	if err != nil {
		glog.Error("Accept failed: ", err)
	}
	return &Socket{conn: conn}, err
}

func (s *ServerSocket) Close() error {
	err := s.ln.Close()
	if err != nil {
		glog.Error("Close failed: ", err)
	}
	return err
}
