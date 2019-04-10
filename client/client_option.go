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
	Option
	Selector selector.Selector
}

var DefaultSGOption = SGOption{
	AppKey:   "",
	Option:   DefaultOption,
	Selector: selector.NewRandomSelector(),
}
