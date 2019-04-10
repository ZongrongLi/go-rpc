package server

import (
	"context"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/golang/glog"
	"github.com/tiancai110a/go-rpc/protocol"
	"github.com/tiancai110a/go-rpc/transport"
)

type DefaultServerWrapper struct {
}

func (w *DefaultServerWrapper) WrapServe(s *SGServer, serveFunc ServeFunc) ServeFunc {
	return func(network string, addr string) error {
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

		//这里注册服务

		glog.Error("server started")
		return serveFunc(network, addr)
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
