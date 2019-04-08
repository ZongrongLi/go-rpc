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

	"github.com/golang/glog"
	"github.com/tiancai110a/go-rpc/client"
	"github.com/tiancai110a/go-rpc/server"
	"github.com/tiancai110a/go-rpc/service"
)

func main() {
	ctx := context.Background()
	s, err := server.NewSimpleServer(nil)
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
	time.Sleep(time.Second * 3)

	c, err := client.NewRPCClient("tcp", ":8888", nil)
	defer c.Close()

	if err != nil {
		glog.Error("NewRPCClient failed,err:", err)
		return
	}

	for i := 0; i < 3; i++ {

		//TODO rtt 延时太长了 猜测是json序列化太慢
		// testrequest := service.TestRequest{i, i + 1}
		// testresponse := service.TestResponse{}
		// err := c.Call(ctx, "TestService.Add", &testrequest, &testresponse)
		// if err != nil {
		// 	glog.Error("Send failed", err)
		// }
		// glog.Info("TestService.Add ================>", testresponse)

		glog.Infof("args A: %d, args B:%d", i, i+1)
		arithrequest := service.ArithRequest{i, i + 1}
		arithresponse := service.ArithResponse{}
		err = c.Call(ctx, "ArithService.Add", &arithrequest, &arithresponse)
		if err != nil {
			glog.Error("Send failed ", err)
		}
		glog.Info("TestService.Add ================>", arithresponse)

		err = c.Call(ctx, "ArithService.Minus", &arithrequest, &arithresponse)
		if err != nil {
			glog.Error("Send failed ", err)
		}
		glog.Info("TestService.Minus ================>", arithresponse)

		err = c.Call(ctx, "ArithService.Mul", &arithrequest, &arithresponse)
		if err != nil {
			glog.Error("Send failed ", err)
		}
		glog.Info("TestService.Mul ================>", arithresponse)

		err = c.Call(ctx, "ArithService.Divide", &arithrequest, &arithresponse)
		if err != nil {
			glog.Error("Send failed ", err)
		}
		glog.Info("TestService.Divide ================>", arithresponse)

		time.Sleep(time.Second * 2)
	}

}
