package lru

import "container/list"

type Cache struct {
	maxBytes int64                    // 允许使用的最大内存
	nbytes   int64                    // 当前使用的内存
	ll       *list.List               // 双向链表
	cache    map[string]*list.Element // 缓存，键是字符串，值是双向链表中对应节点的指针

	OnEvicted func(key string, value Value) // OnEvicted 当某条记录被移除时的回调函数，可以为 nil
}

type entry struct {
	key   string
	value Value
}

// New Cache 的构造函数
func New(maxBytes int64, OnEvicted func(key string, value Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: OnEvicted,
	}
}

// Get 从缓存中获取一个键值对
func (c *Cache) Get(key string) (value Value, ok bool) {
	if e, ok := c.cache[key]; ok {
		c.ll.MoveToFront(e) // 这里约定 front 为队尾
		kv := e.Value.(*entry)
		return kv.value, true
	}
	return
}

// RemoveOldest 移除最近最少访问的节点
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		c.nbytes -= int64(kv.value.Len()) + int64(len(kv.key))
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Add 向缓存中添加/修改一个键值对
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		kv.value = value
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
	} else {
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

// Len 缓存条数
func (c *Cache) Len() int {
	return c.ll.Len()
}

// Value 只包含了一个方法，用于返回值所占用的内存大小
type Value interface {
	Len() int
}
