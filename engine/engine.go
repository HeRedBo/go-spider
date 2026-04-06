package engine

import (
	"go-spider/fetcher"
	"log"

	"github.com/gookit/goutil/dump"
)

func Run(seeds ...Request) {
	var requests []Request
	for _, req := range seeds {
		requests = append(requests, req)
	}
	// 有请求 处理请求 解析数据返回处理结果
	for len(requests) > 0 {
		r := requests[0]
		requests = requests[1:] // 注意使用 = 赋值，避免 变量遮蔽（variable shadowing）问题
		var body []byte
		log.Printf("Fetching type %s: Url: %s", r.Type, r.Url)
		if r.Type == "url" {
			var err error
			body, err = fetcher.Fetch(r.Url, "GET")
			if err != nil {
				log.Printf("Fetcher: error "+" fetching url %s : %s", r.Url, err)
				continue
			}
		} else if r.Type == "html" || r.Type == "json" {
			var data = []byte(r.Text)
			body = data
		}
		parseResult := r.ParserFunc(body)
		requests = append(requests, parseResult.Requests...)
		for _, item := range parseResult.Items {
			//log.Printf("Got Item %v", item)
			dump.P("go-spider item %v", item)
		}
	}

}
