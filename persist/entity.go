package persist

// Entity 持久化实体接口，所有需要保存的数据模型必须实现此接口
type Entity interface {
	ID() string
}
