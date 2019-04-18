/*
 * File: router.go
 * Project: service
 * File Created: Tuesday, 16th April 2019 4:32:34 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Tuesday, 16th April 2019 4:34:09 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * Copyright nil - 2019
 */

package service

import (
	"context"
	"errors"
	"strings"

	"github.com/golang/glog"
)

const Methodpath = "methodpath"
const Groupname = "groupname"

type HTTPErrCode byte

const (
	HTTPErrCodeOK = iota
	HTTPErrCodeFailed
)

type RespBase struct {
	Errcode   HTTPErrCode
	ErrString string
}

type Resp struct {
	RespBase
	Data map[string]string
}

func (r *Resp) Add(key string, value string) {
	r.Data[key] = value
}

func NewResp() Resp {
	r := Resp{}
	r.Errcode = HTTPErrCodeOK
	m := make(map[string]string)
	r.Data = m
	return r
}

type RouterFunc func(ctx context.Context, r *Resp)

type MapRouterFunc map[string]RouterFunc

type MethodType byte

const (
	POST MethodType = iota
	GET
)

var PostGroup2Func = make(map[string]*MapRouterFunc)
var GetGroup2Func = make(map[string]*MapRouterFunc)

func (m *MapRouterFunc) Route(key string, f RouterFunc) {
	key = strings.Trim(key, "/")
	(*m)[key] = f
}

type RouterRequest struct {
}

type RouterResponse struct {
	Data *Resp
}

func NewRouterResponse() RouterResponse {
	r := RouterResponse{}
	rsp := NewResp()
	r.Data = &rsp
	return r
}

type RouterService struct {
}

func (t RouterService) PostRouter(ctx context.Context, req *RouterRequest, res *RouterResponse) error {

	methodpath := ctx.Value(Methodpath).(string)
	groupname := ctx.Value(Groupname).(string)
	m, ok := PostGroup2Func[groupname]
	if !ok {
		glog.Error("group is not registed:", groupname, methodpath)
		//没找到对应的处理group
		return errors.New("method is not registed")
	}
	realfun, ok := (*m)[methodpath]
	if !ok {
		glog.Error("method is not registed: ", groupname, methodpath)
		//没找到对应的处理方法
		return errors.New("method is not registed")
	}
	r := NewResp()
	realfun(ctx, &r)
	if r.RespBase.Errcode != HTTPErrCodeOK {
		r.RespBase.Errcode = HTTPErrCodeFailed
		r.RespBase.ErrString = "就是错了呀"
	}
	//返回结果

	res.Data = &r
	return nil
}

func (t RouterService) GetRouter(ctx context.Context, req *RouterRequest, res *RouterResponse) error {
	methodpath := ctx.Value(Methodpath).(string)
	groupname := ctx.Value(Groupname).(string)
	m, ok := GetGroup2Func[groupname]
	if !ok {
		glog.Error("group is not registed", groupname, methodpath)
		//没找到对应的处理group
		return errors.New("method is not registed")
	}
	realfun, ok := (*m)[methodpath]
	if !ok {
		glog.Error("method is not registed", groupname, methodpath)
		//没找到对应的处理方法
		return errors.New("method is not registed")
	}
	r := NewResp()
	realfun(ctx, &r)
	if r.RespBase.Errcode != HTTPErrCodeOK {
		r.RespBase.Errcode = HTTPErrCodeFailed
		r.RespBase.ErrString = "就是错了呀"
	}
	//返回结果
	res.Data = &r
	return nil

}
