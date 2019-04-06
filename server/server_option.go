package server

import (
	"github.com/tiancai110a/go-rpc/protocol"
	"github.com/tiancai110a/go-rpc/transport"
)

type Option struct {
	ProtocolType  protocol.ProtocolType
	SerializeType protocol.SerializeType
	CompressType  protocol.CompressType
	TransportType transport.TransportType
}

var DefaultOption = Option{
	ProtocolType:  protocol.Default,
	SerializeType: protocol.SerializeTypeJson,
	CompressType:  protocol.CompressTypeNone,
	TransportType: transport.TCPTransport,
}
