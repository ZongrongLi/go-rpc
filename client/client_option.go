package client

import (
	"time"

	"github.com/megaredfan/rpc-demo/transport"
)

type Option struct {
	TransportType  transport.TransportType
	RequestTimeout time.Duration
}

var DefaultOption = Option{
	RequestTimeout: time.Microsecond * 200,
	TransportType:  transport.TCPTransport,
}
