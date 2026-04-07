package main

import (
	"go-spider/engine"
	"go-spider/persist"
	"go-spider/scheduler"
	"go-spider/types"
	"go-spider/zhenai/parser"
	"time"
)

func main() {
	// 初始化 es 链接
	itemChan, err := persist.ItemSaver("dating_profile")
	if err != nil {
		panic(err)
	}
	e := engine.ConcurrentEngine{
		Scheduler:   &scheduler.QueuedScheduler{},
		WorkerCount: 10,
		ItemChan:    itemChan,
	}
	// ========== 开启超时退出（例如 3 分钟自动退出） ==========
	e.WithTimeout(3 * time.Minute)
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
