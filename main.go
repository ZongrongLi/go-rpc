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
)

func main() {
	ctx := context.Background()
	s := server.SimpleServer{}
	go s.Serve("tcp", ":8888")

	time.Sleep(time.Second * 3)

	c, err := client.NewRPCClient("tcp", ":8888")
	defer c.Close()

	if err != nil {
		glog.Error("NewRPCClient failed,err:", err)
		return
	}

	for i := 0; i < 3; i++ {
		reply := 0
		err := c.Call(ctx, i, i+1, &reply)
		if err != nil {
			glog.Error("Send failed")
		}
		glog.Info("+================>", reply)
		time.Sleep(time.Second * 2)
	}

}
