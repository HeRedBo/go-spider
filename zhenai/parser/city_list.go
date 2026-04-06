package parser

import (
	"go-spider/engine"
	"regexp"
	"strconv"
)

var CityListRe = regexp.MustCompile(`<a href="(http://www.zhenai.com/zhenghun/[0-9a-z]+)"[^>]*>([^<]+)</a>`)
var pageLimit = 2 // 限制抓取页面 根据实际需求调整 xia

func ParseCityList(contents []byte) engine.ParseResult {
	result := engine.ParseResult{}
	matches := CityListRe.FindAllSubmatch(contents, -1)
	for _, m := range matches {
		result.Items = append(result.Items, "City "+string(m[2]))
		// 每个城市只取前 6也数据
		if pageLimit > 0 {
			for i := 1; i <= pageLimit; i++ {
				url := string(m[1]) + "/" + strconv.Itoa(i)
				result.Requests = append(result.Requests, engine.Request{
					Type: "url",
					Url:  url,
					ParserFunc: func(bytes []byte) engine.ParseResult {
						//return engine.ParseResult{}
						return ParseCityUserList(bytes, url)
					},
				})
			}
		} else {
			url := string(m[1])
			result.Requests = append(
				result.Requests,
				engine.Request{
					Type: "url",
					Url:  string(m[1]),
					ParserFunc: func(bytes []byte) engine.ParseResult {
						return ParseCityUserList(bytes, url)
					},
				})
		}
	}
	return result
}
