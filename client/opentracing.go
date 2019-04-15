/*
 * File: opentracing.go
 * Project: client
 * File Created: Sunday, 14th April 2019 12:50:47 am
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Sunday, 14th April 2019 12:50:52 am
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * null 2019 - 2019
 */
package client

import (
	"context"
	"log"

	"github.com/golang/glog"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	opentracingLog "github.com/opentracing/opentracing-go/log"
	"github.com/tiancai110a/go-rpc/share/metadata"
	"github.com/tiancai110a/go-rpc/share/trace"
)

type OpenTracingInterceptor struct {
	DefaultClientInterceptor
}

func (*OpenTracingInterceptor) WrapCall(option *SGOption, callFunc CallFunc) CallFunc {
	return func(ctx context.Context, ServiceMethod string, arg interface{}, reply interface{}) error {
		glog.Info("opentracing called")
		var clientSpan opentracing.Span
		//不是心跳的请求才进行追踪
		if ServiceMethod != "" {
			var parentCtx opentracing.SpanContext
			if parent := opentracing.SpanFromContext(ctx); parent != nil {
				parentCtx = parent.Context()
			}
			//开始埋点
			clientSpan := opentracing.StartSpan(
				ServiceMethod,
				opentracing.ChildOf(parentCtx),
				ext.SpanKindRPCClient)
			defer clientSpan.Finish()

			meta := metadata.FromContext(ctx)
			writer := &trace.MetaDataCarrier{&meta}

			injectErr := opentracing.GlobalTracer().Inject(clientSpan.Context(), opentracing.TextMap, writer)
			if injectErr != nil {
				log.Printf("inject trace error: %v", injectErr)
			}
			ctx = metadata.WithMeta(ctx, meta)
		}

		err := callFunc(ctx, ServiceMethod, arg, reply)
		if err != nil && clientSpan != nil {
			clientSpan.LogFields(opentracingLog.String("error", err.Error()))
		}
		return err
	}
}
