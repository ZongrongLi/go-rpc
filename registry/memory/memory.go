/*
 * File: memory.go
 * Project: memory
 * File Created: Monday, 8th April 2019 11:52:36 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Wednesday, 10th April 2019 3:56:24 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * null 2019 - 2019
 */
package memory

import (
	"errors"
	"sync"
	"time"

	"github.com/golang/glog"

	"github.com/google/uuid"
	"github.com/tiancai110a/go-rpc/registry"
)

var (
	timeout = time.Millisecond * 10
)

type Registry struct {
	mu        sync.RWMutex
	providers []registry.Provider
	watchers  map[string]*Watcher
}

func (r *Registry) Register(option registry.RegisterOption, providers ...registry.Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	glog.Info("============================Register:", option.AppKey)
	go r.sendWatcherEvent(registry.Create, option.AppKey, providers...)

	var providers2Register []registry.Provider
	for _, p := range providers {
		exist := false
		for _, cp := range r.providers {
			if cp.ProviderKey == p.ProviderKey {
				exist = true
				break
			}
		}
		if !exist {
			providers2Register = append(providers2Register, p)
		}
	}

	r.providers = append(r.providers, providers2Register...)
}

func (r *Registry) Unregister(option registry.RegisterOption, providers ...registry.Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	go r.sendWatcherEvent(registry.Delete, option.AppKey, providers...)

	var newList []registry.Provider
	for _, p := range r.providers {
		remain := true
		//fix
		for _, up := range providers {
			if p.ProviderKey != up.ProviderKey {
				remain = false
			}
		}
		if remain {
			newList = append(newList, p)
		}
	}
	r.providers = newList
}

func (r *Registry) GetServiceList() []registry.Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.providers
}

func (r *Registry) Watch() registry.Watcher {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.watchers == nil {
		r.watchers = make(map[string]*Watcher)
	}
	event := make(chan *registry.Event)
	exit := make(chan bool)
	id := uuid.New().String()

	w := &Watcher{
		id:   id,
		res:  event,
		exit: exit,
	}

	r.watchers[id] = w
	return w
}

func (r *Registry) Unwatch(watcher registry.Watcher) {
	target, ok := watcher.(*Watcher)
	if !ok {
		return
	}

	r.mu.Lock()
	defer r.mu.Lock()

	var newWatcherList []registry.Watcher
	for _, w := range r.watchers {
		if w.id != target.id {
			newWatcherList = append(newWatcherList, w)
		}
	}
}

func (r *Registry) sendWatcherEvent(action registry.EventAction, AppKey string, providers ...registry.Provider) {
	var watchers []*Watcher
	event := &registry.Event{
		Action:    action,
		AppKey:    AppKey,
		Providers: providers,
	}
	r.mu.RLock()
	glog.Info("=============================================watcher num:", len(r.watchers))
	for _, w := range r.watchers {
		watchers = append(watchers, w)
	}
	r.mu.RUnlock()

	for _, w := range watchers {
		select {
		case <-w.exit:
			r.mu.Lock()
			delete(r.watchers, w.id)
			r.mu.Unlock()
		default:
			select {
			case w.res <- event:
			case <-time.After(timeout):
			}
		}
	}
}

type Watcher struct {
	id   string
	res  chan *registry.Event
	exit chan bool
}

func (m *Watcher) Next() (*registry.Event, error) {
	for {
		select {
		case r := <-m.res:
			return r, nil
		case <-m.exit:
			return nil, errors.New("watcher stopped")
		}
	}
}

func (m *Watcher) Close() {
	select {
	case <-m.exit:
		return
	default:
		close(m.exit)
	}
}

func NewInMemoryRegistry() registry.Registry {
	r := &Registry{}
	return r
}
