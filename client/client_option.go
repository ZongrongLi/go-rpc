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
	RequestTimeout: time.Microsecond * 500,
	TransportType:  transport.TCPTransport,
}
