package hcache

import (
	"fmt"
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

func (g *Group) load(key string) (ByteView, error) {
	// todo 分布式场景下会调用 getFromPeer 从其他节点获取
	return g.getLocally(key)
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
