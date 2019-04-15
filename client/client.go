/*
 * File: client.go
 * Project: client
 * File Created: Friday, 5th April 2019 5:50:17 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Friday, 5th April 2019 5:50:27 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * null lizongrong - 2019
 */
package client

import (
	"context"
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

type Call struct {
	ServiceMethod string     // 服务名.方法名
	Error         error      // 错误信息
	Done          chan *Call // 在调用结束时激活
	Args          interface{}
	Reply         interface{}
}

func (c *Call) done() {
	c.Done <- c
}

//RPCClient  客户端接口
type RPCClient interface {
	Call(ctx context.Context, serviceMethod string, request interface{}, response interface{}) error
	pack(ctx context.Context, serviceMethod string, done chan *Call, request interface{}, response interface{}) *Call
	Close() error
}

type simpleClient struct {
	rwc          io.ReadWriteCloser
	pendingCalls sync.Map
	serializer   protocol.Serializer
	seq          uint64
	option       Option
}

func (c *simpleClient) input() {
	var err error
	for err == nil {
		proto := protocol.ProtocolMap[c.option.ProtocolType]
		responseMsg, err := proto.DecodeMessage(c.rwc, c.serializer)
		if err != nil {
			break
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
			if responseMsg.MessageType == protocol.MessageTypeHeartbeat {
				call.done()
				continue
			}
			err = c.serializer.Unmarshal(responseMsg.Data, call.Reply)
			if err != nil {
				glog.Error("Unmarshal failed: ", err)
				continue
			}
			//glog.Infof("=====>%p %+v %+v", call.Response, call.Response, response)
			call.done()
		}

	}

}

//NewRPCClient 工厂函数
func NewRPCClient(network string, addr string, op *Option) (RPCClient, error) {
	c := simpleClient{}
	tr := transport.Socket{}
	err := tr.Dial(network, addr, transport.DialOption{Timeout: op.DialTimeout})
	if err != nil {
		glog.Error("Connect err:", err)
		return nil, err
	}
	c.rwc = &tr
	if op == nil {
		c.option = DefaultOption
	} else {
		c.option = *op
	}

	c.serializer, err = protocol.NewSerializer(c.option.SerializeType)
	if err != nil {
		//glog.Error("new serializer failed", err)
		return nil, err
	}

	go c.input()
	return &c, nil
}

//Close 关闭连接
//TODO 关闭：协程安全清理通信用的那个map
func (c *simpleClient) Close() error {
	c.pendingCalls.Range(func(key, value interface{}) bool {
		call, ok := value.(*Call)
		if ok {
			call.Error = errors.New("client is shut down")
			call.done()
		}

		c.pendingCalls.Delete(key)
		return true
	})
	return nil
}

//Call call是调用rpc的入口，pack打包request，send负责序列化和发送
func (c *simpleClient) Call(ctx context.Context, serviceMethod string, request interface{}, response interface{}) error {
	seq := atomic.AddUint64(&c.seq, 1)
	ctx = context.WithValue(ctx, protocol.RequestSeqKey, seq)
	canFn := func() {}
	ctx, canFn = context.WithTimeout(ctx, c.option.RequestTimeout)

	done := make(chan *Call, 1)

	call := c.pack(ctx, serviceMethod, done, request, response)
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

func (c *simpleClient) pack(ctx context.Context, serviceMethod string, done chan *Call, request interface{}, response interface{}) *Call {
	call := new(Call)
	call.ServiceMethod = serviceMethod

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

	if call.ServiceMethod != "" {
		serviceMethod := strings.SplitN(call.ServiceMethod, ".", 2)
		if len(serviceMethod) != 2 {
			glog.Error("wrong request name")
			return errors.New("wrong request name")
		}
		requestMsg.ServiceName = serviceMethod[0]
		requestMsg.MethodName = serviceMethod[1]
		requestMsg.SerializeType = c.option.SerializeType
		requestMsg.CompressType = protocol.CompressTypeNone
	} else {
		requestMsg.MessageType = protocol.MessageTypeHeartbeat
	}
	if ctx.Value(protocol.MetaDataKey) != nil {
		requestMsg.MetaData = ctx.Value(protocol.MetaDataKey).(map[string]interface{})
	}

	requestdata, err := c.serializer.Marshal(call.Args)

	requestMsg.Data = requestdata
	data := proto.EncodeMessage(requestMsg, c.serializer)

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
