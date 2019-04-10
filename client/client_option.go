/*
 * File: client_option.go
 * Project: client
 * File Created: Saturday, 6th April 2019 10:53:54 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Monday, 8th April 2019 2:07:38 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * Copyright 2019 - 2019
 */
package client

import (
	"time"

	"github.com/tiancai110a/go-rpc/protocol"
	"github.com/tiancai110a/go-rpc/registry"
	"github.com/tiancai110a/go-rpc/selector"
	"github.com/tiancai110a/go-rpc/transport"
)

type FailMode byte

const (
	FailFast  FailMode = iota //快速失败
	FailOver                  //重试其他服务器
	FailRetry                 //重试同一个服务器
)

type Option struct {
	ProtocolType   protocol.ProtocolType
	SerializeType  protocol.SerializeType
	CompressType   protocol.CompressType
	TransportType  transport.TransportType
	RequestTimeout time.Duration
	DialTimeout    time.Duration
}

var DefaultOption = Option{
	RequestTimeout: time.Millisecond * 100,
	DialTimeout:    time.Millisecond * 10,
	SerializeType:  protocol.SerializeTypeJson,
	CompressType:   protocol.CompressTypeNone,
	TransportType:  transport.TCPTransport,
	ProtocolType:   protocol.Default,
}

type SGOption struct {
	AppKey   string
	Registry registry.Registry
	Selector selector.Selector
	FailMode FailMode
	Retries  int
	Option
}

var DefaultSGOption = SGOption{
	AppKey:   "",
	FailMode: FailFast,
	Option:   DefaultOption,
	Retries:  0,
	Selector: selector.NewRandomSelector(),
}
