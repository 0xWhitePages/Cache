package geecache

import (
	pb "GeeCache/geecachepb"
	"GeeCache/singleflight"
	"fmt"
	"log"
	"sync"
)

//函数接口，让GetterFunc实现Getter接口
type Getter interface {
	Get(key string) ([]byte, error)
}
type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

//Group:与使用者交互的接口。
type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers     PeerPicker
	// use singleflight.Group to make sure that
	// each key is only fetched once
	loader *singleflight.Group
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

// GetGroup returns the named group previously created with NewGroup, or
// nil if there's no such group.
func GetGroup(name string) *Group {
	mu.RLock()
	defer mu.RUnlock()
	g := groups[name]
	return g
}

//Get
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	if v, ok := g.mainCache.get(key); ok {
		log.Println("缓存命中")
		return v, nil
	}
	return g.load(key) //回调
}

//使用实现了 PeerGetter 接口的 httpGetter 从访问远程节点，获取缓存值。
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {

	//bytes, err := peer.Get(g.name, key)
	//if err != nil {
	//	return ByteView{}, err
	//}
	//return ByteView{b: bytes}, nil
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	err := peer.Get(req, res)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}

// RegisterPeers registers a PeerPicker for choosing remote peer
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

//回调逻辑
/*
	1。从远端节点缓存中取。
	2。缓存中有则返回，没有则调用g.getLocally从数据源取
*/
func (g *Group) load(key string) (ByteView, error) {
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err := g.getFromPeer(peer, key); err != nil {
					return value, nil
				} else {
					log.Println("[GeeCache] Failed to get from peer", err)
				}
			}
		}
		return g.getLocally(key)
	})

	if err != nil {
		return viewi.(ByteView), err
	}

	return viewi.(ByteView), nil
}

func (g *Group) getLocally(key string) (ByteView, error) {
	//调用group的回调函数取本地
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	//将getter从数据源取到的数据复制一份给value
	value := ByteView{b: cloneBytes(bytes)}
	//从数据源取到后，还要在缓存中存一下
	g.populateCache(key, value)
	return value, nil
}

//存入cache。 populate v.（给文件）增添数据，输入数据
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.put(key, value)
}
