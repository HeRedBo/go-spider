package main

import (
	"go-spider/engine"
	"go-spider/zhenai/parser"
)

func main() {
	engine.Run(engine.Request{
		Type:       "url",
		Url:        "http://www.zhenai.com/zhenghun",
		ParserFunc: parser.ParseCityList,
		//Url: "http://www.zhenai.com/zhenghun/akesu/1",
		//ParserFunc: func(bytes []byte) engine.ParseResult {
		//	return parser.ParseCityUserList(bytes, "http://www.zhenai.com/zhenghun/akesu/1")
		//},
	})
}
