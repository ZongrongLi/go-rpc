/*
 * File: client.go
 * Project: client
 * File Created: Friday, 5th April 2019 5:50:17 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Friday, 5th April 2019 5:50:27 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * null lizongrong - 2019
 */
package client

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/golang/glog"

	"github.com/tiancai110a/go-rpc/registry"
	"github.com/tiancai110a/go-rpc/selector"
)

type SGClient interface {
	Call(ctx context.Context, serviceMethod string, arg interface{}, reply interface{}) error
	Close() error
}

type sgClient struct {
	shutdown             bool
	option               SGOption
	clients              sync.Map       //map[string]RPCClient
	clientsHeartbeatFail map[string]int //TODO：考虑要不要绑定clients封装成一个结构体
	breakers             sync.Map       //map[string]CircuitBreaker   clients 的信息越加越多，要单独放一个结构体
	serversMu            sync.RWMutex
	servers              []registry.Provider
	mu                   sync.Mutex
	watcher              registry.Watcher
}

//NewRPCClient 工厂函数
func NewSGClient(option SGOption) SGClient {
	c := new(sgClient)
	c.option = option

	providers := c.option.Registry.GetServiceList()
	glog.Info("初始拉全量", providers)
	c.watcher = c.option.Registry.Watch()
	glog.Info("providers", providers)

	go c.watchService(c.watcher)
	c.serversMu.Lock()
	defer c.serversMu.Unlock()
	for _, p := range providers {
		c.servers = append(c.servers, p)
	}
	AddWrapper(&c.option, NewLogWrapper())
	AddWrapper(&c.option, NewMetaDataWrapper())
	AddWrapper(&c.option, &OpenTracingInterceptor{})
	//AddWrapper(&c.option, &RateLimitInterceptor{Limit: &ratelimit.DefaultRateLimiter{Num: 1}})

	if c.option.Heartbeat {
		go c.heartbeat()
		c.option.SelectOption.Filters = append(c.option.SelectOption.Filters,
			selector.DegradeProviderFilter)
	}

	if c.option.Tagged && c.option.Tags != nil {
		c.option.SelectOption.Filters = append(c.option.SelectOption.Filters,
			selector.TaggedProviderFilter(c.option.Tags))
	}
	c.clientsHeartbeatFail = make(map[string]int, 0)

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
		glog.Info("service changed!")

		c.serversMu.Lock()

		//如果曾经下线的重新上线，会创建新的连接
		for _, p := range c.servers {
			for _, pe := range event.Providers {
				if p.ProviderKey == pe.ProviderKey && p.Isdegred {
					cl, err := NewRPCClient(pe.Network, pe.Addr, &c.option.Option)
					if err != nil {
						glog.Error("update clietn error err: ", err)
						return
					}
					c.clients.Store(pe.ProviderKey, cl)
				}
			}
		}
		c.servers = event.Providers

		c.serversMu.Unlock()
	}
}

func (c *sgClient) getClient(provider registry.Provider) (cl RPCClient, err error) {
	key := provider.ProviderKey
	breaker, ok := c.breakers.Load(key)
	if ok && !breaker.(CircuitBreaker).AllowRequest() {
		glog.Info("circuit breaker triggered")
		return nil, errors.New("breaker open") //TODO error全部收敛到一个文件中
	}
	glog.Info("circuit breaker passed")
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
		if c.option.CircuitBreakerThreshold > 0 && c.option.CircuitBreakerWindow > 0 {
			c.breakers.Store(key, NewDefaultCircuitBreaker(c.option.CircuitBreakerThreshold, c.option.CircuitBreakerWindow))
		}
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

	provider, err := c.option.Selector.Next(c.providers(), ctx, ServiceMethod, arg, c.option.SelectOption)
	if err != nil {
		glog.Error("selector failed！", err)
		return registry.Provider{}, nil, err
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
		return err
	}
	if rpcClient == nil {
		glog.Error("getClient failed！")
		return errors.New("getClient failed！")
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
					if breaker, ok := c.breakers.Load(provider.ProviderKey); ok {
						breaker.(CircuitBreaker).Success()
					}
					return err
				}

				if err != nil {
					glog.Error("getclient err:", err)
					if breaker, ok := c.breakers.Load(provider.ProviderKey); ok {
						breaker.(CircuitBreaker).Fail(err)
					}
					return err
				}
			}

			c.removeClient(provider.ProviderKey, rpcClient)
			rpcClient, err = c.getClient(provider)
		}
	case FailOver:
		retries := c.option.Retries
		for retries > 0 {
			retries--
			if rpcClient != nil {
				err = c.wrapCall(rpcClient.Call)(ctx, serviceMethod, arg, reply)
				if err == nil {
					if breaker, ok := c.breakers.Load(provider.ProviderKey); ok {
						breaker.(CircuitBreaker).Success()
					}
					return err
				}

				if err != nil {
					glog.Error("selectClient err:", err)
					if breaker, ok := c.breakers.Load(provider.ProviderKey); ok {
						breaker.(CircuitBreaker).Fail(err)
					}
					return err
				}
			}
			c.removeClient(provider.ProviderKey, rpcClient)
			provider, rpcClient, err = c.selectClient(ctx, serviceMethod, arg)
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
	c.breakers.Delete(clientKey)
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

func (c *sgClient) heartbeat() {
	if c.option.HeartbeatInterval <= 0 {
		return
	}
	//根据指定的时间间隔发送心跳
	t := time.NewTicker(c.option.HeartbeatInterval)
	for range t.C {
		if c.shutdown {
			t.Stop()
			return
		}

		//遍历每个RPCClient进行心跳检查
		c.clients.Range(func(k, v interface{}) bool {

			err := v.(RPCClient).Call(context.Background(), "", "", nil)
			c.mu.Lock()
			if err != nil {
				glog.Info("heartbeat failed")
				//心跳失败进行计数
				if fail, ok := c.clientsHeartbeatFail[k.(string)]; ok {
					fail++
					c.clientsHeartbeatFail[k.(string)] = fail
				} else {
					c.clientsHeartbeatFail[k.(string)] = 1
				}
			} else {
				glog.Info("heartbeat succeed")
				//心跳成功则进行恢复
				c.clientsHeartbeatFail[k.(string)] = 0
				c.serversMu.Lock()
				for i, p := range c.servers {
					if p.ProviderKey == k {
						c.servers[i].Isdegred = false
					}
				}
				c.serversMu.Unlock()
			}
			c.mu.Unlock()
			//心跳失败次数超过阈值则进行降级
			if c.clientsHeartbeatFail[k.(string)] > c.option.HeartbeatDegradeThreshold {
				c.serversMu.Lock()
				for i, p := range c.servers {
					if p.ProviderKey == k {
						//执行降级
						glog.Info("degred")
						c.servers[i].Isdegred = true
					}
				}
				c.serversMu.Unlock()
			}
			return true
		})
	}
}
