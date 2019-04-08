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
	"context"
	"encoding/json"
	"io"
	"reflect"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/golang/glog"
	"github.com/tiancai110a/go-rpc/protocol"
	"github.com/tiancai110a/go-rpc/transport"
)

type RPCServer interface {
	Serve(network string, addr string) error
	Register(rcvr interface{})
	Close() error
}

type methodType struct {
	method    reflect.Method
	ArgType   reflect.Type
	ReplyType reflect.Type
}

type service struct {
	name    string
	typ     reflect.Type
	rcvr    reflect.Value
	methods map[string]*methodType
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
	mutex      sync.Mutex
}

func NewSimpleServer() RPCServer {
	s := simpleServer{}
	s.option = DefaultOption
	return &s
}

func (s *simpleServer) Register(rcvr interface{}) {

	typ := reflect.TypeOf(rcvr)
	name := typ.Name()
	srv := new(service)
	srv.name = name
	srv.rcvr = reflect.ValueOf(rcvr)
	srv.typ = typ

	//TODO 找不到的时候要控制一下
	methods := suitableMethods(typ, true)
	srv.methods = methods

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

		mtype, ok := srv.methods[mname]
		if !ok {
			s.writeErrorResponse(responseMsg, tr, "can not find method")
			return
		}
		argv := newValue(mtype.ArgType)
		replyv := newValue(mtype.ReplyType)

		err = json.Unmarshal(requestMsg.Data, &argv)
		if err != nil {
			glog.Error("read failed: ", err)
			continue
		}

		ctx := context.Background()
		//执行函数
		var returns []reflect.Value
		if mtype.ArgType.Kind() != reflect.Ptr {
			returns = mtype.method.Func.Call([]reflect.Value{srv.rcvr,
				reflect.ValueOf(ctx),
				reflect.ValueOf(argv).Elem(),
				reflect.ValueOf(replyv)})
		} else {
			returns = mtype.method.Func.Call([]reflect.Value{srv.rcvr,
				reflect.ValueOf(ctx),
				reflect.ValueOf(argv),
				reflect.ValueOf(replyv)})
		}
		if len(returns) > 0 && returns[0].Interface() != nil {
			err = returns[0].Interface().(error)
			s.writeErrorResponse(responseMsg, tr, err.Error())
			return
		}

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
			glog.Error(err)
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
	glog.Error(responseMsg.Error)
	responseMsg.StatusCode = protocol.StatusError
	responseMsg.Data = responseMsg.Data[:0]
	_, _ = w.Write(proto.EncodeMessage(responseMsg))
}

func (s *simpleServer) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	err := s.tr.Close()

	s.serviceMap.Range(func(key, value interface{}) bool {
		s.serviceMap.Delete(key)
		return true
	})
	return err
}

// Is this type exported or a builtin?
func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

// Is this an exported - upper case - name?
func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

// Precompute the reflect type for error. Can't use error directly
// because Typeof takes an empty interface value. This is annoying.
var typeOfError = reflect.TypeOf((*error)(nil)).Elem()
var typeOfContext = reflect.TypeOf((*context.Context)(nil)).Elem()

//过滤符合规则的方法，从net.rpc包抄的
func suitableMethods(typ reflect.Type, reportErr bool) map[string]*methodType {
	methods := make(map[string]*methodType)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mtype := method.Type
		mname := method.Name

		// 方法必须是可导出的
		if method.PkgPath != "" {
			continue
		}
		// 需要有四个参数: receiver, Context, args, *reply.
		if mtype.NumIn() != 4 {
			if reportErr {
				glog.Error("method", mname, "has wrong number of ins:", mtype.NumIn())
			}
			continue
		}

		// 第一个参数必须是context.Context
		ctxType := mtype.In(1)
		if !ctxType.Implements(typeOfContext) {
			if reportErr {
				glog.Error("method", mname, " must use context.Context as the first parameter")
			}
			continue
		}

		// 第二个参数是arg
		argType := mtype.In(2)
		if !isExportedOrBuiltinType(argType) {
			if reportErr {
				glog.Error(mname, "parameter type not exported:", argType)
			}
			continue
		}
		// 第三个参数是返回值，必须是指针类型的
		replyType := mtype.In(3)
		if replyType.Kind() != reflect.Ptr {
			if reportErr {
				glog.Error("method", mname, "reply type not a pointer:", replyType)
			}
			continue
		}
		// 返回值的类型必须是可导出的
		if !isExportedOrBuiltinType(replyType) {
			if reportErr {
				glog.Error("method", mname, "reply type not exported:", replyType)
			}
			continue
		}
		// 必须有一个返回值
		if mtype.NumOut() != 1 {
			if reportErr {
				glog.Error("method", mname, "has wrong number of outs:", mtype.NumOut())
			}
			continue
		}
		// 返回值类型必须是error
		if returnType := mtype.Out(0); returnType != typeOfError {
			if reportErr {
				glog.Error("method", mname, "returns", returnType.String(), "not error")
			}
			continue
		}
		methods[mname] = &methodType{method: method, ArgType: argType, ReplyType: replyType}
	}
	return methods
}
