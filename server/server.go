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

type Test struct {
	A int
	B int
}

func Send(s transport.Transport, a int, b int) error {
	t := Test{a, b}
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

type SimpleServer struct {
	tr transport.ServerTransport
}

//todo 增加连接池，而不是每一个都单独建立一个连接
func connhandle(s transport.Transport) {
	for {
		err, t := Recv(s)
		if err == io.EOF {
			break
		}
		if err != nil {
			glog.Error("recv failed ", err)
			return
		}
		glog.Info(t)

		err = Send(s, 1, 2)
		if err != nil {
			glog.Error("Send failed")
		}
	}
}

func (s *SimpleServer) Serve() {
	tr := transport.ServerSocket{}
	defer tr.Close()
	err := tr.Listen("tcp", ":8888")
	if err != nil {
		panic(err)
	}

	for {
		s, err := tr.Accept()
		if err != nil {
			glog.Error("accept err:", err)
			return
		}

		//todo protocol反射 -》 消息分发-》回应
		//先打印出来
		go connhandle(s)

	}
	glog.Info("server end")
}

func (s *SimpleServer) Close() {
	s.tr.Close()
}
