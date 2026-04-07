package persist

// Entity 所有要存入ES的结构体必须实现这个接口
type Entity interface {
	ID() string // 返回唯一ID
}
