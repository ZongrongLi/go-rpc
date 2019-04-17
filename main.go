/*
 * File: main.go
 * Project: go-rpc
 * File Created: Friday, 5th April 2019 12:00:35 am
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Friday, 5th April 2019 4:48:07 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * null lizongrong - 2019
 */
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/docker/libkv/store"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/tiancai110a/go-rpc/registry"
	"github.com/tiancai110a/go-rpc/registry/libkv"
	"github.com/tiancai110a/msgpack"

	"github.com/golang/glog"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/tiancai110a/go-rpc/client"
	"github.com/tiancai110a/go-rpc/protocol"
	"github.com/tiancai110a/go-rpc/server"
	"github.com/tiancai110a/go-rpc/service"
	"github.com/tiancai110a/go-rpc/transport"
)

func testMiddleware1(rw *http.ResponseWriter, r *http.Request, c *server.Middleware) {

	fmt.Println("before===testMiddlewarec1")
	c.Next(nil, nil)

	fmt.Println("after===testMiddlewarec1")
}

func testMiddleware2(rw *http.ResponseWriter, r *http.Request, c *server.Middleware) {
	fmt.Println("before===testMiddlewarec2")
	c.Next(nil, nil)

	fmt.Println("after===testMiddlewarec2")
}

func testMiddleware3(rw *http.ResponseWriter, r *http.Request, c *server.Middleware) {
	fmt.Println("before===testMiddlewarec3")
	c.Next(nil, nil)
	fmt.Println("after===testMiddlewarec3")
}
func TestAdd(ctx context.Context, resp *service.Resp) {

	glog.Info("===========================================================================================resultful func")
	glog.Info("==================================test1:", ctx.Value("test1"))
	glog.Info("==================================test:", ctx.Value("test"))
	glog.Info("==================================name:", ctx.Value("name"))
	glog.Info("==================================pass:", ctx.Value("pass"))
	//	res.data

	resp.Add("name", "tiancai")
	resp.Add("res1", "3.14")
	resp.Add("list1", "1234,4567,1234,0987,3333")
	return
}

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

		sk := s.Group(service.POST, "/invoke")
		if sk == nil {
			glog.Error("server dose not implement http server")
			return
		}
		sk.Route("/Add", TestAdd)
		s.Use(testMiddleware1)
		s.Use(testMiddleware2)
		s.Use(testMiddleware3)
		go s.Serve("tcp", "127.0.0.1:8888", nil)
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

func MakeRequest(req *http.Request,
	msgtype protocol.MessageType,
	comrpesstype protocol.CompressType,
	serliazetype protocol.SerializeType,
	statuscode protocol.StatusCode,
	servicename string,
	methodname string,
	err string,
	meta *map[string]interface{}) *http.Request {

	req.Header.Set(server.HEADER_SEQ, "1")
	req.Header.Set(server.HEADER_MESSAGE_TYPE, strconv.FormatUint((uint64)(msgtype), 10))
	req.Header.Set(server.HEADER_COMPRESS_TYPE, strconv.FormatUint((uint64)(comrpesstype), 10))
	req.Header.Set(server.HEADER_SERIALIZE_TYPE, strconv.FormatUint((uint64)(serliazetype), 10))
	req.Header.Set(server.HEADER_STATUS_CODE, strconv.FormatUint((uint64)(statuscode), 10))
	req.Header.Set(server.HEADER_SERVICE_NAME, servicename)
	req.Header.Set(server.HEADER_METHOD_NAME, methodname)
	req.Header.Set(server.HEADER_ERROR, err)

	metaJson, _ := json.Marshal(meta)
	req.Header.Set(server.HEADER_META_DATA, string(metaJson))
	return req
}

func MakeHttpCall(ctx context.Context, servicename, methoname string, c client.SGClient, a, b int) {
	arg := service.ArithRequest{a, b}

	data, _ := msgpack.Marshal(arg)
	body := bytes.NewBuffer(data)
	req, err := http.NewRequest("POST", "http://localhost:5080/invoke", body)
	if err != nil {
		glog.Info(err)
		return
	}
	meta := map[string]interface{}{"idc": "lf"}

	req = MakeRequest(req,
		protocol.MessageTypeRequest,
		protocol.CompressTypeNone,
		protocol.SerializeTypeMsgpack,
		protocol.StatusOK,
		servicename,
		methoname,
		"",
		&meta)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		glog.Info(err)
		return
	}
	if response.StatusCode != 200 {
		glog.Info(response)
	} else if response.Header.Get(server.HEADER_ERROR) != "" {
		glog.Info(response.Header.Get(server.HEADER_ERROR))
	} else {
		data, err = ioutil.ReadAll(response.Body)
		result := service.ArithResponse{}
		msgpack.Unmarshal(data, &result)
		glog.Infof("===========================++++++++++++++++++++++++++++++++ %d %s %d = %d", a, methoname, b, result.Reply)
	}
}
func main() {

	opentracing.SetGlobalTracer(mocktracer.New())

	//单机伪集群
	// r1 := zookeeper.NewZookeeperRegistry("my-app", "/root/lizongrong/service",
	// 	[]string{"127.0.0.1:1181", "127.0.0.1:2181", "127.0.0.1:3181"}, 1e10, nil)

	r1 := libkv.NewKVRegistry(store.ZK, "my-app", "/root/lizongrong/service",
		[]string{"127.0.0.1:1181", "127.0.0.1:2181", "127.0.0.1:3181"}, 1e10, nil)
	servertOption := server.Option{
		ProtocolType:   protocol.Default,
		SerializeType:  protocol.SerializeTypeMsgpack,
		CompressType:   protocol.CompressTypeNone,
		TransportType:  transport.TCPTransport,
		ShutDownWait:   time.Second * 12,
		Registry:       r1,
		RegisterOption: registry.RegisterOption{"my-app"},
		Tags:           map[string]string{"idc": "lf"}, //只允许机房为lf的请求，客户端取到信息会自己进行转移
	}

	//servertOption.Wrappers = append(slice, elems)

	StartServer(&servertOption)
	time.Sleep(time.Second)

	//ctx := context.Background()
	// op := &client.DefaultSGOption
	// op.AppKey = "my-app"
	// op.RequestTimeout = time.Millisecond * 100
	// op.DialTimeout = time.Millisecond * 100
	// op.HeartbeatInterval = time.Second
	// op.HeartbeatDegradeThreshold = 5
	// op.Heartbeat = true
	// op.SerializeType = protocol.SerializeTypeMsgpack
	// op.CompressType = protocol.CompressTypeNone
	// op.TransportType = transport.TCPTransport
	// op.ProtocolType = protocol.Default
	// op.FailMode = client.FailRetry
	// op.Retries = 3
	// op.Auth = "hello01"
	// //一秒钟失败20次 就会进入贤者模式.. 因为lastupdate时间在不断更新，熔断后继续调用有可能恢复
	// op.CircuitBreakerThreshold = 20
	// op.CircuitBreakerWindow = time.Second

	// //基于标签的路由策略
	// op.Tagged = true
	// op.Tags = map[string]string{"idc": "lf"}

	// op.Wrappers = append(op.Wrappers, &client.RateLimitInterceptor{Limit: ratelimit.NewRateLimiter(10, 2)}) //一秒10个，最多有两个排队

	// r2 := libkv.NewKVRegistry(store.ZK, "my-app", "/root/lizongrong/service",
	// 	[]string{"127.0.0.1:1181", "127.0.0.1:2181", "127.0.0.1:3181"}, 1e10, nil)
	// //r.Register(registry.RegisterOption{"my-app"}, registry.Provider{ProviderKey: "tcp@:8888", Network: "tcp", Addr: ":8888"})
	// op.Registry = r2

	// c := client.NewSGClient(*op)

	// for i := 0; i < 2; i++ {
	// 	makecall(ctx, c, i, i+1)
	// 	time.Sleep(time.Second)
	// }

	// //模拟服务器宕机
	// //gs.Close()
	// time.Sleep(time.Second * 3)
	// for i := 0; i < 2000; i++ {
	// 	makecall(ctx, c, i, i+1)
	// 	time.Sleep(time.Second)
	// }

	// for i := 0; i < 20; i++ {
	// 	MakeHttpCall(ctx, "ArithService", "Add", c, i, i+1)
	// 	time.Sleep(time.Second)

	// 	MakeHttpCall(ctx, "ArithService", "Minus", c, i, i+1)
	// 	time.Sleep(time.Second)

	// 	MakeHttpCall(ctx, "ArithService", "Mul", c, i, i+1)
	// 	time.Sleep(time.Second)

	// 	MakeHttpCall(ctx, "ArithService", "Divide", c, i, i+1)
	// 	time.Sleep(time.Second)

	// }
	time.Sleep(time.Second * 265)

}
