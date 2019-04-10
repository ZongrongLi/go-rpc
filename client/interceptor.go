/*
 * File: interceptor.go
 * Project: client
 * File Created: Wednesday, 10th April 2019 5:45:28 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Wednesday, 10th April 2019 5:46:23 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * Copyright 2019 - 2019
 */

package client

import (
	"context"

	"github.com/golang/glog"
)

type LogWrapper struct {
}

func NewLogWrapper() Wrapper {
	return &LogWrapper{}
}

func (*LogWrapper) WrapCall(option *SGOption, callFunc CallFunc) CallFunc {
	return func(ctx context.Context, ServiceMethod string, arg interface{}, reply interface{}) error {
		glog.Info("before calling, ServiceMethod:%+v, arg:%+v", ServiceMethod, arg)
		err := callFunc(ctx, ServiceMethod, arg, reply)
		glog.Info("after calling, ServiceMethod:%+v, reply:%+v, error: %s", ServiceMethod, reply, err)
		return err
	}
}
