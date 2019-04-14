package server

import (
	"context"

	"github.com/tiancai110a/go-rpc/protocol"
	"github.com/tiancai110a/go-rpc/ratelimit"
	"github.com/tiancai110a/go-rpc/transport"
)

type RequestRateLimitInterceptor struct {
	DefaultServerWrapper
	Limiter ratelimit.RateLimiter
}

func NewRequestRateLimitInterceptor(limiter ratelimit.RateLimiter) Wrapper {
	return &RequestRateLimitInterceptor{Limiter: limiter}
}

func (rl *RequestRateLimitInterceptor) WrapHandleRequest(s *SGServer, requestFunc HandleRequestFunc) HandleRequestFunc {
	return func(ctx context.Context, request *protocol.Message, response *protocol.Message, tr transport.Transport) {
		if rl.Limiter != nil {
			if rl.Limiter.TryAcquire() {
				requestFunc(ctx, request, response, tr)
			} else {
				s.writeErrorResponse(response, tr, "request limited")
			}
		} else {
			requestFunc(ctx, request, response, tr)
		}
	}
}

func (rl *RequestRateLimitInterceptor) WrapServe(s *SGServer, serveFunc ServeFunc) ServeFunc {
	return func(network string, addr string, meta map[string]interface{}) error {
		return serveFunc(network, addr, meta)
	}
}
