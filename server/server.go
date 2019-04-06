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
	"log"

	"github.com/golang/glog"
	"github.com/tiancai110a/go-rpc/protocol"
	"github.com/tiancai110a/go-rpc/transport"
)

//用来传递参数的通用结构体
type Test struct {
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

type RPCServer interface {
	Serve(network string, addr string) error
	Close() error
}

type simpleServer struct {
	tr     transport.ServerTransport
	option Option
}

func NewSimpleServer() RPCServer {
	s := simpleServer{}
	s.option = DefaultOption
	return &s
}
func (s *simpleServer) writeErrorResponse(responseMsg *protocol.Message, w io.Writer, err string) {
	proto := protocol.ProtocolMap[s.option.ProtocolType]
	responseMsg.Error = err
	log.Println(responseMsg.Error)
	responseMsg.StatusCode = protocol.StatusError
	responseMsg.Data = responseMsg.Data[:0]
	_, _ = w.Write(proto.EncodeMessage(responseMsg))
}

//todo 增加连接池，而不是每一个都单独建立一个连接
func (s *simpleServer) connhandle(tr transport.Transport) {
	for {
		var t Test
		proto := protocol.ProtocolMap[s.option.ProtocolType]
		requestMsg, err := proto.DecodeMessage(tr)
		if err != nil {
			break
		}

		err = json.Unmarshal(requestMsg.Data, &t)

		if err != nil {
			glog.Error("read failed: ", err)
			continue
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			glog.Error("recv failed ", err)
			return
		}

		//执行了一些计算和服务
		*(t.Reply) = t.A + t.B

		responseMsg := requestMsg.Clone()
		responseMsg.MessageType = protocol.MessageTypeResponse

		responseData, err := json.Marshal(t)
		if err != nil {
			s.writeErrorResponse(responseMsg, tr, err.Error())
			return
		}

		responseMsg.StatusCode = protocol.StatusOK
		responseMsg.Data = responseData

		_, err = tr.Write(proto.EncodeMessage(responseMsg))
		if err != nil {
			log.Println(err)
			return
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
