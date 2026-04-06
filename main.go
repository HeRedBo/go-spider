package main

import (
	"go-spider/engine"
	"go-spider/scheduler"
	"go-spider/types"
	"go-spider/zhenai/parser"
)

func main() {

	e := engine.ConcurrentEngine{
		Scheduler:   &scheduler.QueuedScheduler{},
		WorkerCount: 10,
	}
	e.Run(types.Request{
		Type: "url",
		//Url:  "http://www.zhenai.com/zhenghun",
		//ParserFunc: parser.ParseCityList,
		Url: "http://www.zhenai.com/zhenghun/akesu/1",
		ParserFunc: func(bytes []byte) types.ParseResult {
			return parser.ParseCityUserList(bytes, "http://www.zhenai.com/zhenghun/akesu/1")
		},
	})
}
