/*
 * File: interceptor.go
 * Project: client
 * File Created: Wednesday, 10th April 2019 5:45:28 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Wednesday, 10th April 2019 5:46:23 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * null 2019 - 2019
 */

package client

import (
	"context"

	"github.com/tiancai110a/go-rpc/protocol"

	"github.com/golang/glog"
)

type LogWrapper struct {
}

func NewLogWrapper() Wrapper {
	return &LogWrapper{}
}

func (*LogWrapper) WrapCall(option *SGOption, callFunc CallFunc) CallFunc {
	return func(ctx context.Context, ServiceMethod string, arg interface{}, reply interface{}) error {
		glog.Infof("before calling, ServiceMethod:%+v, arg:%+v", ServiceMethod, arg)
		err := callFunc(ctx, ServiceMethod, arg, reply)
		glog.Infof("after calling, ServiceMethod:%+v, reply:%+v, error: %+v", ServiceMethod, reply, err)
		return err
	}
}

type MetaDataWrapper struct {
}

func NewMetaDataWrapper() *MetaDataWrapper {
	return &MetaDataWrapper{}
}

func (w *MetaDataWrapper) WrapCall(option *SGOption, callFunc CallFunc) CallFunc {
	return func(ctx context.Context, ServiceMethod string, arg interface{}, reply interface{}) error {
		ctx = wrapContext(ctx, option)
		return callFunc(ctx, ServiceMethod, arg, reply)
	}
}

//鉴权的key可以由上家穿透过来或者配置里直接指定，优先级上家>配置
func wrapContext(ctx context.Context, option *SGOption) context.Context {

	metaDataInterface := ctx.Value(protocol.MetaDataKey)
	var metaData map[string]interface{}
	if metaDataInterface == nil {
		metaData = make(map[string]interface{})
	} else {
		metaData = metaDataInterface.(map[string]interface{})
	}

	if option.Auth != "" {
		metaData[protocol.AuthKey] = option.Auth
	}

	if auth, ok := ctx.Value(protocol.AuthKey).(string); ok {
		metaData[protocol.AuthKey] = auth
	}
	ctx = context.WithValue(ctx, protocol.MetaDataKey, metaData)
	return ctx

}
