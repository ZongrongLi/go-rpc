/*
 * File: registry.go
 * Project: registry
 * File Created: Monday, 8th April 2019 11:52:36 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Wednesday, 10th April 2019 3:56:17 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * null 2019 - 2019
 */
package registry

type Registry interface {
	Register(option RegisterOption, provider ...Provider)
	Unregister(option RegisterOption, provider ...Provider)
	GetServiceList() []Provider
	Watch() Watcher
	Unwatch(watcher Watcher)
}

type RegisterOption struct {
	AppKey string
}

type Watcher interface {
	Next() (*Event, error)
	Close()
}

type EventAction byte

const (
	Create EventAction = iota
	Update
	Delete
)

type Event struct {
	Action    EventAction
	AppKey    string
	Providers []Provider
}

type Provider struct {
	ProviderKey string // Network+"@"+Addr
	Network     string
	Addr        string
	Isdegred    bool
	Meta        map[string]interface{}
}

type Peer2PeerDiscovery struct {
	providers []Provider
}

func (p *Peer2PeerDiscovery) Register(option RegisterOption, providers ...Provider) {
	p.providers = providers
}

func (p *Peer2PeerDiscovery) Unregister(option RegisterOption, provider ...Provider) {
	p.providers = []Provider{}
}

func (p *Peer2PeerDiscovery) GetServiceList() []Provider {
	return p.providers
}

func (p *Peer2PeerDiscovery) Watch() Watcher {
	return nil
}

func (p *Peer2PeerDiscovery) Unwatch(watcher Watcher) {
	return
}

func (p *Peer2PeerDiscovery) WithProvider(provider Provider) *Peer2PeerDiscovery {
	p.providers = append(p.providers, provider)
	return p
}

func (p *Peer2PeerDiscovery) WithProviders(providers []Provider) *Peer2PeerDiscovery {
	for _, provider := range providers {
		p.providers = append(p.providers, provider)
	}
	return p
}

func NewPeer2PeerRegistry() *Peer2PeerDiscovery {
	r := &Peer2PeerDiscovery{}
	return r
}
