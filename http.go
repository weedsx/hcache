package hcache

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

const DefaultBasePath = "/_hcache/"

// HTTPPool 表示一个节点
type HTTPPool struct {
	self     string // 当前节点的基础 URL
	basePath string // 作为节点间通讯地址的前缀
}

// NewHTTPPool 初始化一个 HTTPPool
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{self: self, basePath: DefaultBasePath}
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
