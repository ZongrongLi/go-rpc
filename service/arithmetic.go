/*
 * File: arithmetic.go
 * Project: service
 * File Created: Monday, 8th April 2019 3:50:01 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Monday, 8th April 2019 3:50:12 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * null 2019 - 2019
 */

package service

import (
	"context"
	"errors"
)

type ArithRequest struct {
	A int //发送的参数
	B int
}

type ArithResponse struct {
	Reply int //返回的参数
}

type ArithService struct {
}

func (t ArithService) Add(ctx context.Context, req *ArithRequest, res *ArithResponse) error {

	//	glog.Info("--------------------------------------------------------------------------add")
	res.Reply = req.A + req.B
	return nil
}

func (t ArithService) Minus(ctx context.Context, req *ArithRequest, res *ArithResponse) error {
	res.Reply = req.A - req.B
	//glog.Info("--------------------------------------------------------------------------Minus")

	return nil
}

func (t ArithService) Mul(ctx context.Context, req *ArithRequest, res *ArithResponse) error {
	//	glog.Info("--------------------------------------------------------------------------Mul")

	res.Reply = req.A * req.B
	return nil
}

func (t ArithService) Divide(ctx context.Context, req *ArithRequest, res *ArithResponse) error {
	//	glog.Info("--------------------------------------------------------------------------Divide")

	if req.B == 0 {
		return errors.New("divided by zero")
	}
	res.Reply = req.A / req.B
	return nil
}
