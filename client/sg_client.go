/*
 * File: client.go
 * Project: client
 * File Created: Friday, 5th April 2019 5:50:17 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Friday, 5th April 2019 5:50:27 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * Copyright lizongrong - 2019
 */
package client

import (
	"context"
	"log"
	"sync"

	"github.com/golang/glog"

	"github.com/tiancai110a/go-rpc/registry"
)

type SGClient interface {
	Call(ctx context.Context, serviceMethod string, arg interface{}, reply interface{}) error
	Close() error
}

type sgClient struct {
	shutdown  bool
	option    SGOption
	clients   sync.Map //map[string]RPCClient
	serversMu sync.RWMutex
	servers   []registry.Provider
	mu        sync.Mutex
	watcher   registry.Watcher
}

//NewRPCClient 工厂函数
func NewSGClient(option SGOption) SGClient {
	c := new(sgClient)
	c.option = option

	providers := c.option.Registry.GetServiceList()
	c.watcher = c.option.Registry.Watch()
	glog.Info("==========================================================providers", providers)

	go c.watchService(c.watcher)
	c.serversMu.Lock()
	defer c.serversMu.Unlock()
	for _, p := range providers {
		c.servers = append(c.servers, p)
	}
	AddWrapper(&c.option, NewLogWrapper())

	return c
}
func (c *sgClient) watchService(watcher registry.Watcher) {
	//索性直接拉全量了
	if watcher == nil {
		return
	}
	for {
		event, err := watcher.Next()
		if err != nil {
			log.Println("watch service error:" + err.Error())
			break
		}
		glog.Info("========================================service changed!")

		c.serversMu.Lock()
		c.servers = event.Providers
		c.serversMu.Unlock()
	}
}

func (c *sgClient) getClient(provider registry.Provider) (cl RPCClient, err error) {
	key := provider.ProviderKey
	rc, ok := c.clients.Load(key)

	if ok {
		glog.Info("get client from pool")
		cl = rc.(RPCClient)
	} else {
		glog.Info("new client ")
		cl, err = NewRPCClient(provider.Network, provider.Addr, &c.option.Option)
		if err != nil {
			return
		}
		c.clients.Store(key, cl)
	}
	return
}
func (c *sgClient) providers() []registry.Provider {
	c.serversMu.RLock()
	defer c.serversMu.RUnlock()
	return c.servers
}

//负载均衡接口
func (c *sgClient) selectClient(ctx context.Context, ServiceMethod string, arg interface{}) (registry.Provider, RPCClient, error) {

	//得到下一个provider然后调用client

	provider, err := c.option.Selector.Next(c.providers(), ctx, ServiceMethod, arg)
	if err != nil {
		glog.Error("selector failed！", err)
		return registry.Provider{}, nil, nil
	}

	client, err := c.getClient(provider)
	if err != nil {
		glog.Error("getClient failed！")
		return registry.Provider{}, nil, nil
	}
	return provider, client, nil
}

//Call call是调用rpc的入口，pack打包request，send负责序列化和发送
func (c *sgClient) Call(ctx context.Context, serviceMethod string, arg interface{}, reply interface{}) error {
	provider, rpcClient, err := c.selectClient(ctx, serviceMethod, arg)

	if err != nil {
		glog.Error("getClient failed！")
		return nil
	}
	err = c.wrapCall(rpcClient.Call)(ctx, serviceMethod, arg, reply)
	if err == nil {
		return nil
	}
	switch c.option.FailMode {
	case FailFast:
		glog.Errorf("serviceMethod:%s failed", serviceMethod)
		return err
	case FailRetry:
		retries := c.option.Retries
		for retries > 0 {
			retries--
			if rpcClient != nil {
				err = c.wrapCall(rpcClient.Call)(ctx, serviceMethod, arg, reply)
				if err == nil {
					return err
				}
			}
			c.removeClient(provider.ProviderKey, rpcClient)
			rpcClient, err = c.getClient(provider)

			if err != nil {
				glog.Error("getclient err:", err)
				return err
			}
		}
	case FailOver:
		retries := c.option.Retries
		for retries > 0 {
			retries--
			if rpcClient != nil {
				err = c.wrapCall(rpcClient.Call)(ctx, serviceMethod, arg, reply)
				if err == nil {
					return err
				}
			}
			c.removeClient(provider.ProviderKey, rpcClient)
			provider, rpcClient, err = c.selectClient(ctx, serviceMethod, arg)

			if err != nil {
				glog.Error("selectClient err:", err)
				return err
			}
		}

	default:
		glog.Errorf("serviceMethod:%s failed", serviceMethod)
		return err

	}

	return nil
}

func (c *sgClient) removeClient(clientKey string, client RPCClient) {
	c.clients.Delete(clientKey)
	if client != nil {
		client.Close()
	}
}
func (c *sgClient) wrapCall(callFunc CallFunc) CallFunc {
	for _, wrapper := range c.option.Wrappers {
		callFunc = wrapper.WrapCall(&c.option, callFunc)
	}
	return callFunc
}

func (c *sgClient) Close() error {
	c.shutdown = true

	c.mu.Lock()
	c.clients.Range(func(k, v interface{}) bool {
		if client, ok := v.(simpleClient); ok {
			c.removeClient(k.(string), &client)
		}
		return true
	})
	c.mu.Unlock()

	go func() {
		c.option.Registry.Unwatch(c.watcher)
		c.watcher.Close()
	}()

	return nil
}
