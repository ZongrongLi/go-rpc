package server

import (
	"context"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/golang/glog"
	"github.com/tiancai110a/go-rpc/protocol"
	"github.com/tiancai110a/go-rpc/registry"
	Service "github.com/tiancai110a/go-rpc/service"
	"github.com/tiancai110a/go-rpc/transport"
)

type DefaultServerWrapper struct {
}

func (w *DefaultServerWrapper) WrapServe(s *SGServer, serveFunc ServeFunc) ServeFunc {
	return func(network string, addr string, meta map[string]interface{}) error {
		//注册shutdownHook
		go func(s *SGServer) {
			ch := make(chan os.Signal, 1)
			signal.Notify(ch, syscall.SIGTERM)
			sig := <-ch
			if sig.String() == "terminated" {
				glog.Info("system terminal catched!")
				for _, hook := range s.option.ShutDownHooks {
					hook(s)
				}
				os.Exit(0)
			}
		}(s)
		if meta == nil {
			meta = make(map[string]interface{})
		}
		if len(s.option.Tags) > 0 {
			meta["tags"] = s.option.Tags
		}

		//这里注册服务
		provider := registry.Provider{
			ProviderKey: network + "@" + addr,
			Network:     network,
			Addr:        addr,
			Meta:        meta,
		}

		s.option.Registry.Register(s.option.RegisterOption, provider)
		glog.Error("server started")
		s.Register(Service.RouterService{})
		if s.option.HttpServeOpen == true {
			s.startGateway(s.option.HttpServePort)
		}
		return serveFunc(network, addr, meta)
	}
}

func (w *DefaultServerWrapper) WrapHandleRequest(s *SGServer, requestFunc HandleRequestFunc) HandleRequestFunc {
	return func(ctx context.Context, request *protocol.Message, response *protocol.Message, tr transport.Transport) {
		glog.Info("WrapHandleRequest triggered")
		atomic.AddInt64(&s.requestInProcess, 1)
		defer atomic.AddInt64(&s.requestInProcess, -1)
		requestFunc(ctx, request, response, tr)
	}
}

type ServerAuthInterceptor struct {
	authFunc AuthFunc
}

func NewAuthInterceptor(authFunc AuthFunc) Wrapper {
	return &ServerAuthInterceptor{authFunc}
}

func (*ServerAuthInterceptor) WrapServe(s *SGServer, serveFunc ServeFunc) ServeFunc {
	return serveFunc
}

func (sai *ServerAuthInterceptor) WrapHandleRequest(s *SGServer, requestFunc HandleRequestFunc) HandleRequestFunc {
	return func(ctx context.Context, request *protocol.Message, response *protocol.Message, tr transport.Transport) {
		//心跳不鉴权
		if request.MessageType == protocol.MessageTypeHeartbeat {
			requestFunc(ctx, response, response, tr)
			return
		}
		auth, ok := request.MetaData[protocol.AuthKey].(string)
		if ok {
			//鉴权通过则执行业务逻辑
			if sai.authFunc(auth) {
				glog.Info("auth passed")
				requestFunc(ctx, response, response, tr)
				return
			}
		}

		auth, ok = ctx.Value(protocol.AuthKey).(string)
		if ok {
			//鉴权通过则执行业务逻辑
			if sai.authFunc(auth) {
				glog.Info("auth passed")
				requestFunc(ctx, response, response, tr)
				return
			}
		}

		//鉴权失败则返回异常
		glog.Info("auth reject", auth)
		s.writeErrorResponse(response, tr, "auth failed")
	}
}
