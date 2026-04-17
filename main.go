package main

import (
	bizhiParser "go-spider/3gbizhi/parser"
	"go-spider/engine"
	"go-spider/persist"
	"go-spider/scheduler"
	"go-spider/types"
	"time"
)

func main() {
	// ========== 3gbizhi 壁纸爬虫 ==========

	// 分页限制: 0=不限制(自动跟随下一页), >0=限制爬取页数
	bizhiParser.PageLimit = 1

	itemChan, done, err := persist.FileItemSaver("3gbizhi", 10)
	if err != nil {
		panic(err)
	}

	e := engine.ConcurrentEngine{
		Scheduler:   &scheduler.QueuedScheduler{},
		WorkerCount: 10,
		ItemChan:    itemChan,
	}
	e.WithTimeout(10 * time.Minute)
	e.Run(types.Request{
		Type:       "url",
		Url:        "https://www.3gbizhi.com/meinv/index.html",
		ParserFunc: bizhiParser.ParseListPage,
	})

	// 等待所有图片下载和 CSV 写入完成
	<-done

	// ========== 珍爱网爬虫（已注释）==========
	// itemChan, err := persist.ItemSaver("dating_profile")
	// if err != nil {
	// 	panic(err)
	// }
	// e := engine.ConcurrentEngine{
	// 	Scheduler:   &scheduler.QueuedScheduler{},
	// 	WorkerCount: 10,
	// 	ItemChan:    itemChan,
	// }
	// e.WithTimeout(3 * time.Minute)
	// e.Run(types.Request{
	// 	Type: "url",
	// 	Url:  "http://www.zhenai.com/zhenghun/akesu/1",
	// 	ParserFunc: func(bytes []byte) types.ParseResult {
	// 		return parser.ParseCityUserList(bytes, "http://www.zhenai.com/zhenghun/akesu/1")
	// 	},
	// })
}
