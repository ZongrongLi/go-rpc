/*
 * File: client_rate_limit.go
 * Project: client
 * File Created: Saturday, 13th April 2019 9:08:33 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Saturday, 13th April 2019 9:09:22 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * null 2019 - 2019
 */
package client

import (
	"context"
	"errors"

	"github.com/golang/glog"

	"github.com/tiancai110a/go-rpc/ratelimit"
)

type RateLimitInterceptor struct {
	DefaultClientInterceptor
	Limit ratelimit.RateLimiter
}

var ErrRateLimited = errors.New("request limited")

func (r *RateLimitInterceptor) WrapCall(option *SGOption, callFunc CallFunc) CallFunc {
	return func(ctx context.Context, ServiceMethod string, arg interface{}, reply interface{}) error {
		if r.Limit != nil {
			if r.Limit.TryAcquire() {
				glog.Info("not limited")
				return callFunc(ctx, ServiceMethod, arg, reply)
			} else {
				glog.Info("limited")
				return ErrRateLimited
			}
		} else {
			return callFunc(ctx, ServiceMethod, arg, reply)
		}
	}
}
