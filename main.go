/*
 * @Author: lizongrong
 * @since: 2019-04-04 17:38:42
 * @lastTime: 2019-04-04 23:35:45
 */
package main

import (
	"encoding/json"
	"time"

	"github.com/golang/glog"

	"github.com/tiancai110a/go-rpc/transport"
)

type Test struct {
	A int
	B int
}

func Send(s transport.Transport, a int, b int) error {
	t := Test{a, b}
	data, err := json.Marshal(t)

	if err != nil {
		glog.Error("Marshal failed")
		return err
	}

	_, err = s.Write(data)
	return err
}

func Recv(conn transport.Transport) (error, *Test) {
	data := make([]byte, 100)
	n, err := conn.Read(data)
	if err != nil {
		glog.Error("read failed", err)
		return err, nil
	}
	t := Test{}
	err = json.Unmarshal(data[:n], &t)

	if err != nil {
		glog.Error("read failed", err)
		return err, nil

	}
	return err, &t
}

func main() {

	go func() {
		tr := transport.ServerSocket{}
		defer tr.Close()
		err := tr.Listen("tcp", ":8888")
		if err != nil {
			panic(err)
		}

		for {
			s, err := tr.Accept()
			defer s.Close()
			if err != nil {
				glog.Error("accept err:", err)
				return
			}
			err, t := Recv(s)
			if err != nil {
				glog.Error("recv failed ", err)
				return
			}
			glog.Info(t)

			err = Send(s, 1, 2)
			if err != nil {
				glog.Error("Send failed")
			}
		}

	}()

	time.Sleep(time.Second * 3)

	s := transport.Socket{}
	s.Dial("tcp", ":8888")
	defer s.Close()
	err := Send(&s, 3, 4)
	if err != nil {
		glog.Error("Send failed")
	}
	time.Sleep(time.Second * 3)

	err, t := Recv(&s)
	if err != nil {
		glog.Error("Write failed")
	}
	glog.Info(t)

}
