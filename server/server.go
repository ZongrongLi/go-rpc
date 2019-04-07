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
	"reflect"
	"sync"

	"github.com/golang/glog"
	"github.com/tiancai110a/go-rpc/protocol"
	"github.com/tiancai110a/go-rpc/transport"
)

type RPCServer interface {
	Serve(network string, addr string) error
	Register(rcvr interface{}, arg interface{}, reply interface{})
	Close() error
}

type methodType struct {
	method    reflect.Method
	ArgType   reflect.Type
	ReplyType reflect.Type
}

type service struct {
	name      string
	typ       reflect.Type
	rcvr      reflect.Value
	methodADD *methodType //先实现固定的方法名
}

//用来传递参数的通用结构体
type TestRequest struct {
	A int //发送的参数
	B int
}

type TestResponse struct {
	Reply int //返回的参数
}

type simpleServer struct {
	tr         transport.ServerTransport
	serviceMap sync.Map
	option     Option
}

func NewSimpleServer() RPCServer {
	s := simpleServer{}
	s.option = DefaultOption
	return &s
}

func (s *simpleServer) Register(rcvr interface{}, arg interface{}, reply interface{}) {

	typ := reflect.TypeOf(rcvr)
	name := typ.Name()
	srv := new(service)
	srv.name = name
	srv.rcvr = reflect.ValueOf(rcvr)
	srv.typ = typ

	srv.methodADD = new(methodType)
	srv.methodADD.ArgType = reflect.TypeOf(arg)
	srv.methodADD.ReplyType = reflect.TypeOf(reply)

	glog.Info("service name", srv.name)
	if _, duplicate := s.serviceMap.LoadOrStore(name, srv); duplicate {
		return
	}
	return
}
func newValue(t reflect.Type) interface{} {
	if t.Kind() == reflect.Ptr {
		return reflect.New(t.Elem()).Interface()
	} else {
		return reflect.New(t).Interface()
	}
}

//todo 增加连接池，而不是每一个都单独建立一个连接
func (s *simpleServer) connhandle(tr transport.Transport) {
	for {
		proto := protocol.ProtocolMap[s.option.ProtocolType]
		requestMsg, err := proto.DecodeMessage(tr)
		if err != nil {
			break
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			glog.Error("recv failed ", err)
			return
		}

		responseMsg := requestMsg.Clone()
		responseMsg.MessageType = protocol.MessageTypeResponse
		sname := requestMsg.ServiceName
		mname := requestMsg.MethodName

		srvInterface, ok := s.serviceMap.Load(sname)
		if !ok {
			s.writeErrorResponse(responseMsg, tr, "can not find service")
			return
		}

		srv, ok := srvInterface.(*service)
		if !ok {
			s.writeErrorResponse(responseMsg, tr, "not *service type")
			return

		}

		argv := newValue(srv.methodADD.ArgType)
		err = json.Unmarshal(requestMsg.Data, &argv)

		if err != nil {
			glog.Error("read failed: ", err)
			continue
		}

		//执行函数
		tmps := newValue(srv.typ)
		replyv := newValue(srv.methodADD.ReplyType)
		_ = reflect.ValueOf(tmps).Method(0).Call([]reflect.Value{
			reflect.ValueOf(argv),
			reflect.ValueOf(replyv)})

		glog.Infof("%s.%s is called", sname, mname)

		responseData, err := json.Marshal(replyv)
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

func (s *simpleServer) writeErrorResponse(responseMsg *protocol.Message, w io.Writer, err string) {
	proto := protocol.ProtocolMap[s.option.ProtocolType]
	responseMsg.Error = err
	log.Println(responseMsg.Error)
	responseMsg.StatusCode = protocol.StatusError
	responseMsg.Data = responseMsg.Data[:0]
	_, _ = w.Write(proto.EncodeMessage(responseMsg))
}

func (s *simpleServer) Close() error {
	return s.tr.Close()
}
