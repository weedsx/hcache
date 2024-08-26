package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32

type Map struct {
	hash     Hash
	replicas int            // 虚拟节点倍数
	ring     []int          // 哈希环
	hashMap  map[int]string // 虚拟节点与真实节点的对应关系，键是虚拟节点的哈希值，值是真实节点的名称
}

func New(replicas int, hashFn Hash) *Map {
	if hashFn == nil {
		hashFn = crc32.ChecksumIEEE
	}
	return &Map{
		hash:     hashFn,
		replicas: replicas,
		hashMap:  make(map[int]string),
	}
}

// Add 添加真实节点/机器
func (m *Map) Add(Keys ...string) {
	for _, key := range Keys {
		for i := range m.replicas {
			hash := int(m.hash([]byte(key + strconv.Itoa(i))))
			m.ring = append(m.ring, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.ring)
}

// Get 获取哈希值中与提供的键值最接近的节点
func (m *Map) Get(key string) string {
	if len(m.ring) == 0 {
		return ""
	}
	hash := int(m.hash([]byte(key)))
	// 二分查找合适的虚拟节点
	idx := sort.Search(len(m.ring), func(i int) bool {
		return m.ring[i] >= hash
	})
	virtualNodeHash := m.ring[idx%len(m.ring)]
	return m.hashMap[virtualNodeHash]
}
