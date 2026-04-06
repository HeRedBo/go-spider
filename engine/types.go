package engine

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
