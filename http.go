package hcache

import (
	"fmt"
	"hcache/consistenthash"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const defaultBasePath = "/_hcache/"
const defaultReplicas = 50

// HTTPPool 表示一个节点 (服务端)
type HTTPPool struct {
	self        string // 当前节点的基础 URL
	basePath    string // 作为节点间通讯地址的前缀
	mu          sync.Mutex
	peers       *consistenthash.Map
	httpGetters map[string]*httpGetter // 远程节点与对应的 httpGetter 实例的映射
}

// NewHTTPPool 初始化一个 HTTPPool
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{self: self, basePath: defaultBasePath}
}

func (p *HTTPPool) Log(format string, v ...any) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP 处理所有节点间通讯请求。
// 约定访问路径格式为 /<basepath>/<groupname>/<key>，
// 通过 groupname 得到 group 实例，再使用 group.Get(key) 获取缓存数据
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)

	// /<basepath>/<groupname>/<key> required
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}
	v, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	body := v.ByteSlice()
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(body)
}

// Set 更新 HTTPPool 的节点列表
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// PickPeer 实现 PeerPicker 接口，根据 key 挑选一个节点
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

// 类型断言，检查 HTTPPool 是否实现了 PeerPicker 接口
var _ PeerPicker = (*HTTPPool)(nil)

// 节点客户端
type httpGetter struct {
	baseURL string // 表示将要访问的远程节点的地址，例如 http://example.com/_hcache/
}

// Get 实现 PeerGetter 接口
func (h httpGetter) Get(group, key string) ([]byte, error) {
	u := fmt.Sprintf("%v%v%v", h.baseURL, url.QueryEscape(group), url.QueryEscape(key))
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil
}
