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
	"time"

	"github.com/tiancai110a/go-rpc/client"
	"github.com/tiancai110a/go-rpc/server"

	"github.com/golang/glog"
)

func main() {

	s := server.SimpleServer{}
	go s.Serve()

	time.Sleep(time.Second * 3)

	c := client.SimpleClient{}
	err := c.Connect("tcp", ":8888")
	defer c.Close()
	if err != nil {
		glog.Error("connect failed,err:", err)
		return
	}

	for i := 0; i < 3; i++ {
		err := c.Call(3, 4)
		if err != nil {
			glog.Error("Send failed")
		}
		time.Sleep(time.Second * 2)
	}

}
