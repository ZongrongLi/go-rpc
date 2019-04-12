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

	"github.com/tiancai110a/go-rpc/registry"

	"github.com/golang/glog"
	"github.com/tiancai110a/go-rpc/client"
	"github.com/tiancai110a/go-rpc/protocol"
	"github.com/tiancai110a/go-rpc/registry/zookeeper"
	"github.com/tiancai110a/go-rpc/server"
	"github.com/tiancai110a/go-rpc/service"
	"github.com/tiancai110a/go-rpc/transport"
)

//用来停止server，测试心跳功能
var gs server.RPCServer

func StartServer(op *server.Option) {
	go func() {
		s, err := server.NewSGServer(op)
		if err != nil {
			glog.Error("new serializer failed", err)
			return
		}
		//s.Register(service.TestService{})
		err = s.Register(service.ArithService{})

		gs = s
		if err != nil {
			glog.Error("Register failed,err:", err)

		}

		go s.Serve("tcp", "127.0.0.1:8888")
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
	//单机伪集群
	r1 := zookeeper.NewZookeeperRegistry("my-app", "/root/lizongrong/service",
		[]string{"127.0.0.1:1181", "127.0.0.1:2181", "127.0.0.1:3181"}, 1e10, nil)
	servertOption := server.Option{
		ProtocolType:   protocol.Default,
		SerializeType:  protocol.SerializeTypeMsgpack,
		CompressType:   protocol.CompressTypeNone,
		TransportType:  transport.TCPTransport,
		ShutDownWait:   time.Second * 12,
		Registry:       r1,
		RegisterOption: registry.RegisterOption{"my-app"},
	}

	StartServer(&servertOption)
	time.Sleep(time.Second)

	op := &client.DefaultSGOption
	op.AppKey = "my-app"
	op.RequestTimeout = time.Millisecond * 100
	op.DialTimeout = time.Millisecond * 100
	op.HeartbeatInterval = time.Second
	op.HeartbeatDegradeThreshold = 5
	op.Heartbeat = true
	op.SerializeType = protocol.SerializeTypeMsgpack
	op.CompressType = protocol.CompressTypeNone
	op.TransportType = transport.TCPTransport
	op.ProtocolType = protocol.Default
	op.FailMode = client.FailRetry
	op.Retries = 3
	op.Auth = "hello01"

	r2 := zookeeper.NewZookeeperRegistry("my-app", "/root/lizongrong/service",
		[]string{"127.0.0.1:1181", "127.0.0.1:2181", "127.0.0.1:3181"}, 1e10, nil)

	//r.Register(registry.RegisterOption{"my-app"}, registry.Provider{ProviderKey: "tcp@:8888", Network: "tcp", Addr: ":8888"})
	op.Registry = r2

	c := client.NewSGClient(*op)

	for i := 0; i < 2; i++ {
		makecall(ctx, c, i, i+1)
		time.Sleep(time.Second)
	}

	//gs.Close()
	time.Sleep(time.Second * 13)
	for i := 0; i < 2; i++ {
		makecall(ctx, c, i, i+1)
		time.Sleep(time.Second)
	}
	time.Sleep(time.Second * 265)

}
