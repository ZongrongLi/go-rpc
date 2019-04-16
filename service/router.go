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

type RouterFunc func(ctx context.Context)

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
}

type RouterService struct {
}

func (t RouterService) PostRouter(ctx context.Context, req *RouterRequest, res *RouterRequest) error {

	methodpath := ctx.Value(Methodpath).(string)
	groupname := ctx.Value(Groupname).(string)
	m, ok := PostGroup2Func[groupname]
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
	realfun(ctx)
	//返回结果

	return nil

}

func (t RouterService) GetRouter(ctx context.Context, req *RouterRequest, res *RouterRequest) error {
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
	realfun(ctx)
	//返回结果

	return nil

}
