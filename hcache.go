package hcache

import (
	"fmt"
	pb "hcache/hcachepb/hcachepb"
	"hcache/singleflight"
	"log"
	"sync"
)

type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc 接口型函数，
// 既能够将普通的函数类型（需类型转换）作为参数，
// 也可以将实现了该接口的结构体作为参数，使用更为灵活。
// 有关接口型函数，详见：https://geektutu.com/post/7days-golang-q1.html
type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// Group 是一个缓存命名空间和加载的相关数据
type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers     PeerPicker
	loader    *singleflight.Group // 使用 singleflight.Group 来确保每个键只被获取一次 (防止缓存击穿)
}

var mu sync.RWMutex
var groups = make(map[string]*Group)

// NewGroup 初始化一个Group
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

// GetGroup 返回先前用 NewGroup 创建的命名组，如果没有这样的组，则返回nil
func GetGroup(name string) *Group {
	mu.Lock()
	defer mu.Unlock()
	return groups[name]
}

// Get 从缓存中获取键值
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	if value, ok := g.mainCache.get(key); ok {
		log.Println("[HCache] hit")
		return value, nil
	}
	return g.load(key)
}

// RegisterPeers 注册一个 PeerPicker 以选择远程 peer
func (g *Group) RegisterPeers(peer PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peer
}

func (g *Group) load(key string) (ByteView, error) {
	// 确保每个键只被获取一次 (防止缓存击穿)
	val, err := g.loader.Do(key, func() (any, error) {
		// 从分布式节点中读取缓存
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if values, err := g.getFromPeer(peer, key); err != nil {
					return values, nil
				}
			}
			log.Println("[HCache] Failed to get from peer")
		}
		// 本地读取缓存
		return g.getLocally(key)
	})
	if err != nil {
		return ByteView{}, err
	}
	return val.(ByteView), nil
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}

	if err := peer.Get(req, res); err != nil {
		return ByteView{}, err
	}

	return ByteView{b: res.GetValue()}, nil
}

// getLocally 调用用户回调函数 g.getter.Get() 获取源数据，并且将源数据添加到缓存 mainCache 中
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
