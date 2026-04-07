package types

type Request struct {
	Type       string // 请求类型 url  html
	Url        string // 请求链接
	Text       string // 文本内容
	ParserFunc func([]byte) ParseResult
}

type ParseResult struct {
	Requests []Request
	Items    []interface{}
}

func NilParser([]byte) ParseResult {
	return ParseResult{}
}

type ReadyNotifier interface {
	WorkerReady(chan Request)
}

// 标记接口：只要实现了这个接口，就表示可以存入 数据处理模块

type Persistable interface {
	IsPersistable() bool
}
