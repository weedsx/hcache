package hcache

// PeerPicker 实现该接口，用于查找拥有特定密钥的 peer
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter 是 peer 必须实现的接口
type PeerGetter interface {
	Get(group, key string) ([]byte, error)
}
