/*
 * File: zookeeper.go
 * Project: zookeeper
 * File Created: Thursday, 11th April 2019 6:12:39 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Friday, 12th April 2019 1:21:08 am
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * Copyright 2019 - 2019
 */
package zookeeper

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/libkv"
	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/zookeeper"
	"github.com/golang/glog"
	"github.com/tiancai110a/go-rpc/registry"
	"github.com/tiancai110a/go-rpc/share"
)

func init() {
	zookeeper.Register()
}

type ZookeeperRegistry struct {
	AppKey         string        //一个ZookeeperRegistry实例和一个appkey关联
	ServicePath    string        //数据存储的基本路径位置，比如/service/providers
	UpdateInterval time.Duration //定时拉取数据的时间间隔

	kv store.Store //store实例是一个封装过的zk客户端

	providersMu sync.RWMutex
	providers   []registry.Provider //本地缓存的列表

	watchersMu sync.Mutex
	watchers   []*Watcher //watcher列表
}

type Watcher struct {
	event chan *registry.Event
	exit  chan struct{}
}

func (w *Watcher) Next() (*registry.Event, error) {
	for {
		select {
		case r := <-w.event:
			return r, nil
		case <-w.exit:
			return nil, errors.New("watcher stopped")
		}
	}
}

func (w *Watcher) Close() {
	select {
	case <-w.exit:
		return
	default:
		close(w.exit)
	}
}

func NewZookeeperRegistry(AppKey string, ServicePath string, zkAddrs []string,
	updateInterval time.Duration, cfg *store.Config) registry.Registry {
	zk := new(ZookeeperRegistry)
	zk.AppKey = AppKey
	zk.ServicePath = ServicePath
	zk.UpdateInterval = updateInterval

	kv, err := libkv.NewStore(store.ZK, zkAddrs, cfg)
	if err != nil {
		glog.Fatalf("cannot create zk registry: %v", err)
	}
	zk.kv = kv

	basePath := zk.ServicePath
	if basePath[0] == '/' { //路径不能以"/"开头
		basePath = basePath[1:]
		zk.ServicePath = basePath
	}

	//先创建基本路径
	err = zk.kv.Put(basePath, []byte("base path"), &store.WriteOptions{IsDir: true})
	if err != nil {
		glog.Fatalf("cannot create zk path %s: %v", zk.ServicePath, err)
	}

	//显式拉取一次数据
	zk.doGetServiceList()
	go func() {
		t := time.NewTicker(updateInterval)

		for range t.C {
			//定时拉取数据
			zk.doGetServiceList()
		}
	}()

	go func() {
		//watch数据
		zk.watch()
	}()
	return zk
}

func (zk *ZookeeperRegistry) watch() {
	//每次watch到数据后都需要重新watch，所以是一个死循环
	for {
		//监听appkey对应的目录,一旦父级目录的数据有变更就重新读取服务列表
		appkeyPath := constructServiceBasePath(zk.ServicePath, zk.AppKey)

		//监听时先检查路径是否存在
		if exist, _ := zk.kv.Exists(appkeyPath); !exist {
			lastUpdate := strconv.Itoa(int(time.Now().UnixNano()))
			err := zk.kv.Put(appkeyPath, []byte(lastUpdate), &store.WriteOptions{IsDir: true})
			if err != nil {
				glog.Errorf("create path before watch error,  key %v", appkeyPath)
			}
		}
		ch, err := zk.kv.Watch(appkeyPath, nil)
		if err != nil {
			glog.Fatalf("error watch %v", err)
		}

		watchFinish := false
		for !watchFinish {
			//循环读取watch到的数据
			select {
			case pairs := <-ch:
				if pairs == nil {
					glog.Errorf("read finish")
					//watch数据结束，跳出这次循环
					watchFinish = true
				}

				//重新读取服务列表
				latestPairs, err := zk.kv.List(appkeyPath)
				if err != nil {
					watchFinish = true
				}

				zk.providersMu.RLock()
				list := zk.providers
				zk.providersMu.RUnlock()
				for _, p := range latestPairs {
					glog.Errorf("got provider %v", kv2Provider(p))
					list = append(list, kv2Provider(p))
				}

				zk.providersMu.Lock()
				zk.providers = list
				zk.providersMu.Unlock()

				//通知watcher
				for _, w := range zk.watchers {
					w.event <- &registry.Event{AppKey: zk.AppKey, Providers: list}
				}
			}
		}
	}
}

func (zk *ZookeeperRegistry) Register(option registry.RegisterOption, provider ...registry.Provider) {
	serviceBasePath := constructServiceBasePath(zk.ServicePath, option.AppKey)

	for _, p := range provider {
		if p.Addr[0] == ':' {
			p.Addr = share.LocalIpV4() + p.Addr
		}
		key := serviceBasePath + p.Network + "@" + p.Addr
		data, _ := json.Marshal(p.Meta)
		err := zk.kv.Put(key, data, nil)
		if err != nil {
			glog.Errorf("zookeeper register error: %v, provider: %v", err, p)
		}

		//注册时更新父级目录触发watch
		lastUpdate := strconv.Itoa(int(time.Now().UnixNano()))
		err = zk.kv.Put(serviceBasePath, []byte(lastUpdate), nil)
		if err != nil {
			glog.Errorf("zookeeper register modify lastupdate error: %v, provider: %v", err, p)
		}
	}
}

func (zk *ZookeeperRegistry) Unregister(option registry.RegisterOption, provider ...registry.Provider) {
	serviceBasePath := constructServiceBasePath(zk.ServicePath, option.AppKey)

	for _, p := range provider {
		if p.Addr[0] == ':' {
			p.Addr = share.LocalIpV4() + p.Addr
		}
		key := serviceBasePath + p.Network + "@" + p.Addr
		err := zk.kv.Delete(key)
		if err != nil {
			glog.Errorf("zookeeper unregister error: %v, provider: %v", err, p)
		}

		//注销时更新父级目录触发watch
		lastUpdate := strconv.Itoa(int(time.Now().UnixNano()))
		err = zk.kv.Put(serviceBasePath, []byte(lastUpdate), nil)
		if err != nil {
			glog.Errorf("zookeeper register modify lastupdate error: %v, provider: %v", err, p)
		}
	}
}

func (zk *ZookeeperRegistry) GetServiceList() []registry.Provider {
	zk.providersMu.RLock()
	defer zk.providersMu.RUnlock()
	return zk.providers
}

func (zk *ZookeeperRegistry) doGetServiceList() {
	path := constructServiceBasePath(zk.ServicePath, zk.AppKey)
	kvPairs, err := zk.kv.List(path)

	if err != nil {
		glog.Errorf("error get service list %v", err)
		return
	}
	var list []registry.Provider
	for _, pair := range kvPairs {
		provider := kv2Provider(pair)
		list = append(list, provider)
	}
	glog.Errorf("get service list:", list)
	zk.providersMu.Lock()
	zk.providers = list
	zk.providersMu.Unlock()
}

func (zk *ZookeeperRegistry) Watch() registry.Watcher {
	w := &Watcher{event: make(chan *registry.Event, 10), exit: make(chan struct{}, 10)}
	zk.watchersMu.Lock()
	zk.watchers = append(zk.watchers, w)
	zk.watchersMu.Unlock()
	return w
}

func (zk *ZookeeperRegistry) Unwatch(watcher registry.Watcher) {
	var list []*Watcher
	zk.watchersMu.Lock()
	defer zk.watchersMu.Unlock()
	for _, w := range zk.watchers {
		if w != watcher {
			list = append(list, w)
		}
	}
	zk.watchers = list
}

func constructServiceBasePath(basePath string, appkey string) string {
	serviceBasePathBuffer := bytes.NewBufferString(basePath)
	serviceBasePathBuffer.WriteString("/")
	serviceBasePathBuffer.WriteString(appkey)
	serviceBasePathBuffer.WriteString("/")
	return serviceBasePathBuffer.String()
}

func kv2Provider(kv *store.KVPair) registry.Provider {
	provider := registry.Provider{}
	provider.ProviderKey = kv.Key
	networkAndAddr := strings.SplitN(kv.Key, "@", 2)
	provider.Network = networkAndAddr[0]
	provider.Addr = networkAndAddr[1]

	return provider
}
