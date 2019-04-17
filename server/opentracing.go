package server

import (
	"context"
	"log"

	"github.com/golang/glog"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/tiancai110a/go-rpc/protocol"
	"github.com/tiancai110a/go-rpc/share/metadata"
	"github.com/tiancai110a/go-rpc/share/trace"
	"github.com/tiancai110a/go-rpc/transport"
)

type OpenTracingInterceptor struct {
	DefaultServerWrapper
}

//根据请求方法名等信息生成链路信息
//通过rpc metadata传递追踪信息
func (*OpenTracingInterceptor) WrapHandleRequest(s *SGServer, requestFunc HandleRequestFunc) HandleRequestFunc {
	return func(ctx context.Context, request *protocol.Message, response *protocol.Message, tr transport.Transport) {
		if protocol.MessageTypeHeartbeat != request.MessageType {
			meta := metadata.FromContext(ctx)
			spanContext, err := opentracing.GlobalTracer().Extract(opentracing.TextMap, &trace.MetaDataCarrier{&meta})
			if err != nil && err != opentracing.ErrSpanContextNotFound {
				log.Printf("extract span from meta error: %v", err)
			}

			serverSpan := opentracing.StartSpan(
				request.ServiceName+"."+request.MethodName,
				ext.RPCServerOption(spanContext),
				ext.SpanKindRPCServer)
			defer serverSpan.Finish()
			ctx = opentracing.ContextWithSpan(ctx, serverSpan)

			meta = metadata.FromContext(ctx)
			glog.Infof("request.ServiceName:%s  request.MethodName:%s metadataL:%+v", request.ServiceName, request.MethodName, ctx)
		}
		requestFunc(ctx, request, response, tr)
	}
}
func (*OpenTracingInterceptor) WrapServe(s *SGServer, serveFunc ServeFunc) ServeFunc {
	return serveFunc
}
