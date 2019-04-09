/*
 * File: sgserver.go
 * Project: server
 * File Created: Tuesday, 9th April 2019 5:09:30 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Tuesday, 9th April 2019 5:09:33 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * Copyright 2019 - 2019
 */

package server

import (
	"context"
	"errors"
	"io"
	"reflect"
	"sync"

	"github.com/golang/glog"
	"github.com/tiancai110a/go-rpc/protocol"
	"github.com/tiancai110a/go-rpc/transport"
)

type SGServer struct {
	tr         transport.ServerTransport
	serviceMap sync.Map
	option     Option
	serializer protocol.Serializer
	mutex      sync.Mutex
	protocol   protocol.Protocol
	shutdown   bool
}

func NewSGmpleServer(op *Option) (RPCServer, error) {
	s := SGServer{}
	proto := protocol.ProtocolMap[s.option.ProtocolType]
	s.protocol = proto
	if op == nil {
		s.option = DefaultOption
	} else {
		s.option = *op
	}
	var err error
	s.serializer, err = protocol.NewSerializer(s.option.SerializeType)
	if err != nil {
		//glog.Error("new serializer failed", err)
		return nil, err
	}
	return &s, nil
}

func (s *SGServer) Register(rcvr interface{}) error {

	typ := reflect.TypeOf(rcvr)
	name := typ.Name()
	srv := new(service)
	srv.name = name
	srv.rcvr = reflect.ValueOf(rcvr)
	srv.typ = typ

	methods := suitableMethods(typ, true)
	if len(methods) == 0 {
		var errorStr string

		// 如果对应的类型没有任何符合规则的方法，扫描对应的指针类型
		// 也是从net.rpc包里抄来的
		method := suitableMethods(reflect.PtrTo(srv.typ), false)
		if len(method) != 0 {
			errorStr = "rpcx.Register: type " + name + " has no exported methods of suitable type (hint: pass a pointer to value of that type)"
		} else {
			errorStr = "rpcx.Register: type " + name + " has no exported methods of suitable type"
		}
		glog.Info(errorStr)
		return errors.New(errorStr)
	}

	srv.methods = methods

	glog.Info("service name", srv.name)
	if _, duplicate := s.serviceMap.LoadOrStore(name, srv); duplicate {
		return nil
	}
	return nil
}

func (s *SGServer) serveTransport(tr transport.Transport) {
	for {
		if s.shutdown {
			tr.Close()
			break
		}
		requestMsg, err := s.protocol.DecodeMessage(tr, s.serializer)
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
		ctx := context.Background()

		s.doHandleRequest(ctx, requestMsg, responseMsg, tr)
	}
}

func (s *SGServer) doHandleRequest(ctx context.Context, requestMsg *protocol.Message, responseMsg *protocol.Message, tr transport.Transport) {

	sname := requestMsg.ServiceName
	mname := requestMsg.MethodName

	srvInterface, ok := s.serviceMap.Load(sname)
	if !ok {
		glog.Error("can not find service")
		s.writeErrorResponse(responseMsg, tr, "can not find service")
		return
	}
	srv, ok := srvInterface.(*service)
	if !ok {
		glog.Error("not *service type")
		s.writeErrorResponse(responseMsg, tr, "not *service type")
		return
	}

	glog.Infof("%s.%s is called", sname, mname)

	argv, err := reflecttionArgs(srv, mname)
	if err != nil {
		glog.Error("reflecttionArgs failed:", err)
		s.writeErrorResponse(responseMsg, tr, err.Error())
		return
	}
	err = s.serializer.Unmarshal(requestMsg.Data, &argv)
	if err != nil {
		glog.Error("Unmarshal args failed: ", err)
		s.writeErrorResponse(responseMsg, tr, err.Error())
		return
	}

	//调用方法
	replyv, err := reflectionCall(ctx, srv, mname, argv)
	if err != nil {
		glog.Error("reflectionCall failed: ", err)
		s.writeErrorResponse(responseMsg, tr, err.Error())
		return
	}

	responseData, err := s.serializer.Marshal(replyv)
	if err != nil {
		glog.Error("serializer failed: ", err)
		s.writeErrorResponse(responseMsg, tr, err.Error())
		return
	}

	responseMsg.StatusCode = protocol.StatusOK
	responseMsg.Data = responseData

	_, err = tr.Write(s.protocol.EncodeMessage(responseMsg, s.serializer))
	if err != nil {
		glog.Error("trasport failed: ", err)
		s.writeErrorResponse(responseMsg, tr, err.Error())
		return
	}
}

func (s *SGServer) Serve(network string, addr string) error {
	if s.shutdown {
		return nil
	}
	tr := transport.ServerSocket{}
	defer tr.Close()
	err := tr.Listen(network, addr)
	if err != nil {
		panic(err)
	}

	for {

		if s.shutdown {
			break
		}
		con, err := tr.Accept()
		if s.shutdown {
			return nil
		}
		if err != nil {
			glog.Error("accept err:", err)
			return err
		}

		go s.serveTransport(con)

	}
	glog.Info("server end")
	return nil
}

func (s *SGServer) writeErrorResponse(responseMsg *protocol.Message, w io.Writer, err string) {
	proto := protocol.ProtocolMap[s.option.ProtocolType]
	responseMsg.Error = err
	glog.Error(responseMsg.Error)
	responseMsg.StatusCode = protocol.StatusError
	responseMsg.Data = responseMsg.Data[:0]
	_, _ = w.Write(proto.EncodeMessage(responseMsg, s.serializer))
}

func (s *SGServer) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.shutdown = true

	err := s.tr.Close()

	s.serviceMap.Range(func(key, value interface{}) bool {
		s.serviceMap.Delete(key)
		return true
	})
	return err
}
