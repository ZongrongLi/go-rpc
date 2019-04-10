/*
 * File: selector.go
 * Project: selector
 * File Created: Tuesday, 9th April 2019 10:54:42 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Wednesday, 10th April 2019 3:56:09 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * Copyright 2019 - 2019
 */
package selector

import (
	"context"

	"github.com/golang/glog"

	"github.com/tiancai110a/go-rpc/registry"
)

type Selector interface {
	Next(providers []registry.Provider, ctx context.Context, ServiceMethod string, arg interface{}) (registry.Provider, error)
}

var RandomSelectorInstance = RandomSelector{}

type RandomSelector struct {
}

func (RandomSelector) Next(providers []registry.Provider, ctx context.Context, ServiceMethod string, arg interface{}) (p registry.Provider, err error) {
	glog.Info("selector:Next proceder num:", len(providers))
	return providers[0], nil
}
func NewRandomSelector() Selector {
	return RandomSelectorInstance
}
