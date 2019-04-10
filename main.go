/*
 * File: main.go
 * Project: go-rpc
 * File Created: Friday, 5th April 2019 12:00:35 am
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Friday, 5th April 2019 4:48:07 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * Copyright lizongrong - 2019
 */
package main

import (
	"context"
	"time"

	"github.com/tiancai110a/go-rpc/registry/memory"

	"github.com/golang/glog"
	"github.com/tiancai110a/go-rpc/client"
	"github.com/tiancai110a/go-rpc/protocol"
	"github.com/tiancai110a/go-rpc/registry"
	"github.com/tiancai110a/go-rpc/server"
	"github.com/tiancai110a/go-rpc/service"
	"github.com/tiancai110a/go-rpc/transport"
)

func StartServer(op *server.Option) {
	go func() {
		s, err := server.NewSimpleServer(op)
		if err != nil {
			glog.Error("new serializer failed", err)
			return
		}
		//	s.Register(service.TestService{})
		err = s.Register(service.ArithService{})
		if err != nil {
			glog.Error("Register failed,err:", err)

		}

		go s.Serve("tcp", ":8888")
	}()
}

func makecall(ctx context.Context, c client.SGClient, a, b int) {

	arithrequest := service.ArithRequest{a, b}
	arithresponse := service.ArithResponse{}
	err := c.Call(ctx, "ArithService.Add", &arithrequest, &arithresponse)
	if err != nil {
		glog.Error("Send failed ", err)
	}

	err = c.Call(ctx, "ArithService.Minus", &arithrequest, &arithresponse)
	if err != nil {
		glog.Error("Send failed ", err)
	}

	err = c.Call(ctx, "ArithService.Mul", &arithrequest, &arithresponse)
	if err != nil {
		glog.Error("Send failed ", err)
	}

	err = c.Call(ctx, "ArithService.Divide", &arithrequest, &arithresponse)
	if err != nil {
		glog.Error("Send failed ", err)
	}
}
func main() {
	ctx := context.Background()

	servertOption := server.Option{
		ProtocolType:  protocol.Default,
		SerializeType: protocol.SerializeTypeMsgpack,
		CompressType:  protocol.CompressTypeNone,
		TransportType: transport.TCPTransport,
		ShutDownWait:  time.Second * 12,
	}
	StartServer(&servertOption)
	time.Sleep(time.Second * 3)

	op := &client.DefaultSGOption
	op.AppKey = "my-app"
	op.RequestTimeout = time.Millisecond * 100
	op.DialTimeout = time.Millisecond * 100
	op.SerializeType = protocol.SerializeTypeMsgpack
	op.CompressType = protocol.CompressTypeNone
	op.TransportType = transport.TCPTransport
	op.ProtocolType = protocol.Default
	op.FailMode = client.FailRetry
	op.Retries = 3

	r := memory.NewInMemoryRegistry()
	r.Register(registry.RegisterOption{"my-app"}, registry.Provider{ProviderKey: "tcp@:8888", Network: "tcp", Addr: ":8888"})
	op.Registry = r

	c := client.NewSGClient(*op)

	for i := 0; i < 2; i++ {
		makecall(ctx, c, i, i+1)
		time.Sleep(time.Second)
	}

	time.Sleep(time.Second * 2)

}
