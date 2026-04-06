package main

import (
	"go-spider/engine"
	"go-spider/scheduler"
	"go-spider/types"
	"go-spider/zhenai/parser"
	"time"
)

func main() {

	e := engine.ConcurrentEngine{
		Scheduler:   &scheduler.QueuedScheduler{},
		WorkerCount: 10,
	}
	// ========== 开启超时退出（例如 3 分钟自动退出） ==========
	e.WithTimeout(3 * time.Minute)
	e.Run(types.Request{
		Type: "url",
		//Url:  "http://www.zhenai.com/zhenghun",
		//ParserFunc: parser.ParseCityList,
		Url: "http://www.zhenai.com/zhenghun/akesu/2",
		ParserFunc: func(bytes []byte) types.ParseResult {
			return parser.ParseCityUserList(bytes, "http://www.zhenai.com/zhenghun/akesu/1")
		},
	})
}
