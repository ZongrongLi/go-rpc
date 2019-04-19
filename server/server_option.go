/*
 * File: server_option.go
 * Project: server
 * File Created: Sunday, 7th April 2019 12:37:32 am
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Monday, 8th April 2019 2:07:54 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * null 2019 - 2019
 */
package server

import (
	"time"

	"github.com/tiancai110a/go-rpc/protocol"
	"github.com/tiancai110a/go-rpc/registry"
	"github.com/tiancai110a/go-rpc/transport"
)

type Option struct {
	Registry       registry.Registry
	RegisterOption registry.RegisterOption
	ProtocolType   protocol.ProtocolType
	SerializeType  protocol.SerializeType
	CompressType   protocol.CompressType
	TransportType  transport.TransportType
	ShutDownWait   time.Duration
	Wrappers       []Wrapper
	HttpWraper     []HTTPServeFunc
	HttpBeginPoint *Middleware
	HttpServePort  int
	HttpServeOpen  bool
	ShutDownHooks  []ShutDownHook
	Tags           map[string]string
}

var DefaultOption = Option{
	ProtocolType:   protocol.Default,
	SerializeType:  protocol.SerializeTypeJson,
	CompressType:   protocol.CompressTypeNone,
	TransportType:  transport.TCPTransport,
	ShutDownWait:   time.Second * 12,
	HttpBeginPoint: nil,
	HttpServeOpen:  false,
}

type ShutDownHook func(s *SGServer)
