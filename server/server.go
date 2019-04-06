/*
 * File: server.go
 * Project: server
 * File Created: Friday, 5th April 2019 4:35:00 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Friday, 5th April 2019 4:48:26 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * Copyright lizongrong - 2019
 */

package server

import (
	"encoding/json"
	"io"

	"github.com/golang/glog"
	"github.com/tiancai110a/go-rpc/transport"
)

//用来传递参数的通用结构体
type Test struct {
	Seq   uint64
	A     int //发送的参数
	B     int
	Reply *int //返回的参数
}

func Send(s transport.Transport, t *Test) error {
	data, err := json.Marshal(t)
	if err != nil {
		glog.Error("Marshal failed")
		return err
	}

	_, err = s.Write(data)
	return err
}

func Recv(conn transport.Transport) (error, *Test) {
	data := make([]byte, 10000)
	n, err := conn.Read(data)
	if err == io.EOF {
		return err, nil
	}
	if err != nil {
		glog.Error("read failed", err)
		return err, nil
	}
	t := Test{}
	err = json.Unmarshal(data[:n], &t)

	if err != nil {
		glog.Error("read failed", err)
		return err, nil

	}
	return err, &t
}

type RPCServer interface {
	Serve(network string, addr string) error
	Close() error
}

type simpleServer struct {
	tr transport.ServerTransport
}

func NewSimpleServer() RPCServer {
	s := simpleServer{}
	return &s
}

//todo 增加连接池，而不是每一个都单独建立一个连接
func (s *simpleServer) connhandle(tr transport.Transport) {
	for {
		err, t := Recv(tr)
		if err == io.EOF {
			break
		}
		if err != nil {
			glog.Error("recv failed ", err)
			return
		}
		*(t.Reply) = t.A + t.B

		err = Send(tr, t)
		if err != nil {
			glog.Error("Send failed")
		}
	}
}

func (s *simpleServer) Serve(network string, addr string) error {
	tr := transport.ServerSocket{}

	defer tr.Close()
	err := tr.Listen(network, addr)
	if err != nil {
		panic(err)
	}

	for {
		con, err := tr.Accept()
		if err != nil {
			glog.Error("accept err:", err)
			return err
		}

		go s.connhandle(con)

	}
	glog.Info("server end")
	return nil
}

func (s *simpleServer) Close() error {
	return s.tr.Close()
}
