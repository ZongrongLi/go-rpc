/*
 * File: client.go
 * Project: client
 * File Created: Friday, 5th April 2019 5:50:17 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Friday, 5th April 2019 5:50:27 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * Copyright lizongrong - 2019
 */
package client

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

//Recv 暂时用来处理response的handler
func Recv(conn transport.Transport) (error, *Test) {
	data := make([]byte, 10000)
	n, err := conn.Read(data)

	if err != nil {
		return err, nil
	}
	t := Test{}
	err = json.Unmarshal(data[:n], &t)

	if err != nil {
		glog.Error("read failed: ", err)
		return err, nil

	}
	return err, &t
}

//RPCClient
type RPCClient interface {
	Call(a int, b int) error
	Close() error
}

//SimpleClient
type SimpleClient struct {
	rwc io.ReadWriteCloser
}

func (c *SimpleClient) input(s transport.Transport) {
	var err error
	for err == nil {
		var t *Test
		err, t = Recv(s)
		if err == io.EOF || err == io.ErrClosedPipe {
			break
		}
		if err != nil {
			glog.Error("read failed")
			break
		}
		glog.Info(t)

	}

}

//Connect 创建连接
func (c *SimpleClient) Connect(network string, addr string) error {

	tr := transport.Socket{}
	err := tr.Dial(network, addr)
	if err != nil {
		glog.Error("Connect err:", err)
		return err
	}
	c.rwc = &tr

	go c.input(&tr)
	return nil
}

//Close 关闭连接
func (c *SimpleClient) Close() error {
	err := c.rwc.Close()
	if err != nil {
		glog.Info("socket already clsosed")
	}
	return err
}

//Call call是调用rpc的入口，pack打包request，send负责序列化和发送
func (c *SimpleClient) Call(a int, b int) error {
	c.pack(a, b)
	return nil
}

func (c *SimpleClient) pack(a int, b int) error {
	t := Test{a, b}
	err := c.send(t)

	if err != nil {
		glog.Error("send failed", err)
		return err
	}
	return nil

}

func (c *SimpleClient) send(t Test) error {
	data, err := json.Marshal(t)

	if err != nil {
		glog.Error("Marshal failed", err)
		return err
	}

	_, err = c.rwc.Write(data)
	if err != nil {
		glog.Error("Write failed", err)
		return err
	}
	return err

}
