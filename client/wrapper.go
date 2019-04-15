/*
 * File: wrapper.go
 * Project: client
 * File Created: Tuesday, 9th April 2019 10:18:43 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Wednesday, 10th April 2019 5:48:05 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * null 2019 - 2019
 */
package client

import "context"

type CallFunc func(ctx context.Context, ServiceMethod string, arg interface{}, reply interface{}) error

type Wrapper interface {
	WrapCall(option *SGOption, callFunc CallFunc) CallFunc
}

type DefaultClientInterceptor struct {
}

func (DefaultClientInterceptor) WrapCall(option *SGOption, callFunc CallFunc) CallFunc {
	return callFunc
}
