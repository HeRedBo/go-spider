package engine

import (
	"go-spider/types"

	"github.com/gookit/goutil/dump"
)

type SimpleEngine struct {
}

func (e SimpleEngine) Run(seeds ...types.Request) {
	var requests []types.Request
	for _, req := range seeds {
		requests = append(requests, req)
	}

	// 有请求 处理请求 解析数据返回处理结果
	for len(requests) > 0 {
		r := requests[0]
		requests = requests[1:] // 注意使用 = 赋值，避免 变量遮蔽（variable shadowing）问题
		parseResult, err := worker(r)
		if err != nil {
			continue
		}
		requests = append(requests, parseResult.Requests...)
		for _, item := range parseResult.Items {
			//log.Printf("Got Item %v", item)
			dump.P("go-spider item %v", item)
		}
	}
}
