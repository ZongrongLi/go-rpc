/*
 * File: client.go
 * Project: client
 * File Created: Friday, 5th April 2019 5:50:17 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Friday, 5th April 2019 5:50:27 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * Copyright lizongrong - 2019
 */
package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/golang/glog"
	"github.com/tiancai110a/go-rpc/protocol"
	"github.com/tiancai110a/go-rpc/transport"
)

//用来传递参数的通用结构体
type TestRequest struct {
	A int //发送的参数
	B int
}

type TestResponse struct {
	Reply int //返回的参数
}

type Call struct {
	ServiceMethod string     // 服务名.方法名
	Error         error      // 错误信息
	Done          chan *Call // 在调用结束时激活
	Args          *TestRequest
	Reply         *TestResponse
}

func (c *Call) done() {
	c.Done <- c
}

//RPCClient  客户端接口
type RPCClient interface {
	Call(ctx context.Context, request *TestRequest, response *TestResponse) error
	Close() error
}

type simpleClient struct {
	rwc          io.ReadWriteCloser
	pendingCalls sync.Map
	seq          uint64
	option       Option
}

func (c *simpleClient) input(s transport.Transport) {
	var err error
	for err == nil {
		proto := protocol.ProtocolMap[c.option.ProtocolType]
		responseMsg, err := proto.DecodeMessage(c.rwc)
		if err != nil {
			break
		}
		response := TestResponse{}
		err = json.Unmarshal(responseMsg.Data, &response)

		if err != nil {
			glog.Error("read failed: ", err)
			continue
		}

		seq := responseMsg.Seq
		CallInterface, ok := c.pendingCalls.Load(seq)
		if !ok {
			glog.Error("sequence number  not found")
			continue
		}
		call, ok := CallInterface.(*Call)
		if !ok {
			glog.Error("CallInterface converse failed")
			continue
		}
		c.pendingCalls.Delete(seq)

		switch {
		case call == nil:
			glog.Error("call is canceled before")
		default:

			*(call.Reply) = response
			//glog.Infof("=====>%p %+v %+v", call.Response, call.Response, response)
			call.done()
		}

	}

}

//NewRPCClient 工厂函数
func NewRPCClient(network string, addr string) (RPCClient, error) {
	c := simpleClient{}
	tr := transport.Socket{}
	err := tr.Dial(network, addr)
	if err != nil {
		glog.Error("Connect err:", err)
		return nil, err
	}
	c.rwc = &tr
	c.option = DefaultOption
	go c.input(&tr)
	return &c, nil
}

//Close 关闭连接
func (c *simpleClient) Close() error {
	err := c.rwc.Close()
	if err != nil {
		glog.Info("socket already clsosed")
	}
	return err
}

//Call call是调用rpc的入口，pack打包request，send负责序列化和发送
func (c *simpleClient) Call(ctx context.Context, request *TestRequest, response *TestResponse) error {
	seq := atomic.AddUint64(&c.seq, 1)
	ctx = context.WithValue(ctx, protocol.RequestSeqKey, seq)
	canFn := func() {}
	ctx, canFn = context.WithTimeout(ctx, c.option.RequestTimeout)

	done := make(chan *Call, 1)

	call := c.pack(ctx, done, request, response)
	select {
	case <-ctx.Done():
		canFn()
		c.pendingCalls.Delete(seq)
		glog.Errorf("rpc timeout: server: %s", call.ServiceMethod)
		call.Error = errors.New("client request time out")
	case <-call.Done:
	}
	return call.Error
}

func (c *simpleClient) pack(ctx context.Context, done chan *Call, request *TestRequest, response *TestResponse) *Call {
	call := new(Call)
	call.ServiceMethod = "test.add"

	call.Reply = response
	call.Args = request
	if done == nil {
		done = make(chan *Call, 10) // buffered.
	} else {
		if cap(done) == 0 {
			panic("rpc: done channel is unbuffered")
		}
	}
	call.Done = done

	c.send(ctx, call)

	return call
}

func (c *simpleClient) send(ctx context.Context, call *Call) error {
	seq := ctx.Value(protocol.RequestSeqKey).(uint64)
	c.pendingCalls.Store(seq, call)
	proto := protocol.ProtocolMap[c.option.ProtocolType]
	requestMsg := proto.NewMessage()
	requestMsg.Seq = seq
	requestMsg.MessageType = protocol.MessageTypeRequest
	serviceMethod := strings.SplitN(call.ServiceMethod, ".", 2)
	requestMsg.ServiceName = serviceMethod[0]
	requestMsg.MethodName = serviceMethod[1]
	requestMsg.SerializeType = c.option.SerializeType
	requestMsg.CompressType = protocol.CompressTypeNone

	requestdata, err := json.Marshal(call.Args)
	requestMsg.Data = requestdata
	data := proto.EncodeMessage(requestMsg)

	if err != nil {
		glog.Error("Marshal failed", err)
		return err
	}

	_, err = c.rwc.Write(data)
	if err != nil {
		glog.Error("Write failed", err)
		return err
	}
	return err

}
