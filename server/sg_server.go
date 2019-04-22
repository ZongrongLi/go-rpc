/*
 * File: sgserver.go
 * Project: server
 * File Created: Tuesday, 9th April 2019 5:09:30 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Tuesday, 9th April 2019 5:09:33 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * null 2019 - 2019
 */

package server

import (
	"context"
	"errors"
	"io"
	"log"
	"reflect"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/tiancai110a/go-rpc/protocol"
	"github.com/tiancai110a/go-rpc/registry"
	"github.com/tiancai110a/go-rpc/transport"
)

type SGServer struct {
	tr               transport.ServerTransport
	serviceMap       sync.Map
	option           Option
	serializer       protocol.Serializer
	mutex            sync.Mutex
	protocol         protocol.Protocol
	requestInProcess int64 //当前正在处理中的总的请求数
	shutdown         bool

	network string
	addr    string
}

func NewSGServer(op *Option) (RPCServer, error) {
	s := SGServer{}
	proto := protocol.ProtocolMap[s.option.ProtocolType]
	s.protocol = proto
	if op == nil {
		s.option = DefaultOption
	} else {
		s.option = *op
	}
	var err error
	s.option.HttpGroupBeginPoint = make(map[string]*Middleware)
	s.serializer, err = protocol.NewSerializer(s.option.SerializeType)
	if err != nil {
		//glog.Error("new serializer failed", err)
		return nil, err
	}

	//auth
	authWrapper := NewAuthInterceptor(func(key string) bool {
		if len(key) >= 5 && key[:5] == "hello" {
			return true
		}
		return false
	})

	s.option.Wrappers = append(s.option.Wrappers, &DefaultServerWrapper{})
	s.option.Wrappers = append(s.option.Wrappers, authWrapper)
	s.option.Wrappers = append(s.option.Wrappers, &OpenTracingInterceptor{})

	//rateLimitor := NewRequestRateLimitInterceptor(rlimit)
	// s.option.Wrappers = append(s.option.Wrappers, rateLimitor)

	s.AddShutdownHook(func(s *SGServer) {
		provider := registry.Provider{
			ProviderKey: s.network + "@" + s.addr,
			Network:     s.network,
			Addr:        s.addr,
		}
		s.option.Registry.Unregister(s.option.RegisterOption, provider)
		s.Close()
	})

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
		if requestMsg.MessageType == protocol.MessageTypeRequest {
			responseMsg.MessageType = protocol.MessageTypeResponse
		}
		ctx := context.Background()

		s.wrapHandleRequest(s.doHandleRequest)(ctx, requestMsg, responseMsg, tr)
	}
}

func (s *SGServer) writeResponse(ctx context.Context, tr transport.Transport, response *protocol.Message) {
	deadline, ok := ctx.Deadline()
	proto := protocol.ProtocolMap[s.option.ProtocolType]
	if ok {
		if time.Now().Before(deadline) {
			_, err := tr.Write(proto.EncodeMessage(response, s.serializer))
			if err != nil {
				log.Println("write response error:" + err.Error())
			}
		} else {
			log.Println("passed deadline, give up write response")
		}
	} else {
		_, _ = tr.Write(proto.EncodeMessage(response, s.serializer))
	}

}
func (s *SGServer) wrapHandleRequest(handleFunc HandleRequestFunc) HandleRequestFunc {
	for _, w := range s.option.Wrappers {
		handleFunc = w.WrapHandleRequest(s, handleFunc)
	}
	return handleFunc
}
func (s *SGServer) doHandleRequest(ctx context.Context, requestMsg *protocol.Message, responseMsg *protocol.Message, tr transport.Transport) {
	responseMsg = s.process(ctx, requestMsg, responseMsg)
	s.writeResponse(ctx, tr, responseMsg)
}

func errorResponse(message *protocol.Message, err string) *protocol.Message {
	message.Error = err
	message.StatusCode = protocol.StatusError
	message.Data = message.Data[:0]
	return message
}

func (s *SGServer) process(ctx context.Context, requestMsg *protocol.Message, responseMsg *protocol.Message) *protocol.Message {

	if requestMsg.MessageType == protocol.MessageTypeHeartbeat {
		responseMsg.MessageType = protocol.MessageTypeHeartbeat
		return responseMsg
	}
	sname := requestMsg.ServiceName
	mname := requestMsg.MethodName

	srvInterface, ok := s.serviceMap.Load(sname)
	if !ok {
		glog.Error("can not find service")
		return errorResponse(responseMsg, "can not find service")

	}
	srv, ok := srvInterface.(*service)
	if !ok {
		glog.Error("not *service type")
		return errorResponse(responseMsg, "not *service type")

	}

	glog.Infof("%s.%s is called", sname, mname)

	argv, err := reflecttionArgs(srv, mname)
	if err != nil {
		glog.Error("reflecttionArgs failed:", err)
		return errorResponse(responseMsg, err.Error())

	}
	if requestMsg.Data != nil {
		err = s.serializer.Unmarshal(requestMsg.Data, &argv)
		if err != nil {
			glog.Error("Unmarshal args failed: ", err)
			return errorResponse(responseMsg, err.Error())

		}
	}

	//调用方法
	replyv, err := reflectionCall(ctx, srv, mname, argv)
	if err != nil {
		glog.Error("reflectionCall failed: ", err)
		return errorResponse(responseMsg, err.Error())

	}

	responseData, err := s.serializer.Marshal(replyv)
	if err != nil {
		glog.Error("serializer failed: ", err)
		return errorResponse(responseMsg, err.Error())

	}

	responseMsg.StatusCode = protocol.StatusOK
	responseMsg.Data = responseData

	return responseMsg

}

func (s *SGServer) wrapServe(serveFunc ServeFunc) ServeFunc {
	for _, w := range s.option.Wrappers {
		serveFunc = w.WrapServe(s, serveFunc)
	}
	return serveFunc
}
func (s *SGServer) Serve(network string, addr string, meta map[string]interface{}) error {
	s.network = network
	s.addr = addr
	return s.wrapServe(s.serve)(network, addr, nil)
}

func (s *SGServer) serve(network string, addr string, meta map[string]interface{}) error {
	if s.shutdown {
		return nil
	}
	tr := transport.ServerSocket{}
	s.tr = &tr
	defer tr.Close()
	err := tr.Listen(network, addr)
	if err != nil {
		panic(err)
	}
	//这里注册服务
	provider := registry.Provider{
		ProviderKey: network + "@" + addr,
		Network:     network,
		Addr:        addr,
		Meta:        meta,
	}

	s.option.Registry.Register(s.option.RegisterOption, provider)

	glog.Info("tcp server at:", addr)

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

func (s *SGServer) AddShutdownHook(hook ShutDownHook) {
	s.mutex.Lock()
	s.option.ShutDownHooks = append(s.option.ShutDownHooks, hook)
	s.mutex.Unlock()
}

func (s *SGServer) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.shutdown = true

	s.serviceMap.Range(func(key, value interface{}) bool {
		s.serviceMap.Delete(key)
		return true
	})

	//等待当前请求处理完或者直到指定的时间
	ticker := time.NewTicker(s.option.ShutDownWait)
	defer ticker.Stop()
	for {
		if s.requestInProcess <= 0 {
			break
		}
		select {
		case <-ticker.C:
			break
		default:
			continue
		}
		time.Sleep(time.Millisecond * 200)
	}
	err := s.tr.Close()

	if err != nil {
		glog.Error("transport has been released")
	}
	return err
}
