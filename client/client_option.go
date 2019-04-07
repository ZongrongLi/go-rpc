package client

import (
	"time"

	"github.com/megaredfan/rpc-demo/transport"
	"github.com/tiancai110a/go-rpc/protocol"
)

type Option struct {
	ProtocolType   protocol.ProtocolType
	SerializeType  protocol.SerializeType
	CompressType   protocol.CompressType
	TransportType  transport.TransportType
	RequestTimeout time.Duration
}

var DefaultOption = Option{
	RequestTimeout: time.Microsecond * 1000,
	SerializeType:  protocol.SerializeTypeJson,
	CompressType:   protocol.CompressTypeNone,
	TransportType:  transport.TCPTransport,
	ProtocolType:   protocol.Default,
}
