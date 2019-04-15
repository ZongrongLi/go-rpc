/*
 * File: selector.go
 * Project: selector
 * File Created: Tuesday, 9th April 2019 10:54:42 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Wednesday, 10th April 2019 3:56:09 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * null 2019 - 2019
 */
package selector

import (
	"context"
	"errors"
	"math/rand"

	"github.com/golang/glog"

	"github.com/tiancai110a/go-rpc/registry"
)

type Filter func(provider registry.Provider, ctx context.Context, ServiceMethod string, arg interface{}) bool

type SelectOption struct {
	Filters []Filter
}

//客户端实现，基于tags进行过滤
//serve 注册的时候将tags放到meta中通过provider一起放到zk中，客户端拉数据的时候一起拿到meta信息，
//selector调用的时候检测检测自身的tags和某个provider的tag是否匹配
func TaggedProviderFilter(tags map[string]string) Filter {
	return func(provider registry.Provider, ctx context.Context, ServiceMethod string, arg interface{}) bool {

		if tags == nil {
			glog.Info("tags passed")
			return true
		}
		if provider.Meta == nil {
			glog.Info("tags not passed")
			return false
		}
		providerTags, ok := provider.Meta["tags"].(map[string]interface{})
		if !ok || len(providerTags) <= 0 {
			glog.Info("tags not passed")
			return false
		}
		for k, v := range tags {
			if tag, ok := providerTags[k].(string); ok {
				if tag != v {
					glog.Info("tags not passed")

					return false
				}
			} else {
				glog.Info("tags not passed")
				return false
			}
		}

		glog.Info("tags passed")
		return true
	}
}

func DegradeProviderFilter(provider registry.Provider, ctx context.Context, ServiceMethod string, arg interface{}) bool {
	return !provider.Isdegred
}

type Selector interface {
	Next(providers []registry.Provider, ctx context.Context, ServiceMethod string, arg interface{}, opt SelectOption) (registry.Provider, error)
}

var RandomSelectorInstance = RandomSelector{}

//TODO 可以接入轮询策略和一致性hash等等
type RandomSelector struct {
}

func (RandomSelector) Next(providers []registry.Provider, ctx context.Context, ServiceMethod string, arg interface{}, opt SelectOption) (p registry.Provider, err error) {
	glog.Info("selector:Next proceder num:", len(providers))

	filters := combineFilter(opt.Filters)
	list := make([]registry.Provider, 0)
	for _, p := range providers {
		if filters(p, ctx, ServiceMethod, arg) {
			list = append(list, p)
		} else {
			glog.Info("degraded")
		}
	}

	if len(list) == 0 {
		err = errors.New("provider list is empty")
		return
	}
	i := rand.Intn(len(providers))
	p = providers[i]
	return p, nil
}

func combineFilter(filters []Filter) Filter {
	return func(provider registry.Provider, ctx context.Context, ServiceMethod string, arg interface{}) bool {
		for _, f := range filters {
			if !f(provider, ctx, ServiceMethod, arg) {
				return false
			}
		}
		return true
	}
}

func NewRandomSelector() Selector {
	return RandomSelectorInstance
}
