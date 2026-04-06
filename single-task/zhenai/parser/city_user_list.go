package parser

import (
	"go-spider/single-task/engine"
	"regexp"
	"strings"
)

var InstallRe = `window.__INITIAL_STATE__=(.*?)</script>`

func ParseCityUserList(contents []byte, url string) engine.ParseResult {
	result := engine.ParseResult{}
	matches := regexp.MustCompile(InstallRe).FindAllSubmatch(contents, -1)
	for _, m := range matches {
		jsonStr := string(m[1])
		Text := strings.Replace(jsonStr, ";(function(){var s;(s=document.currentScript||document.scripts[document.scripts.length-1]).parentNode.removeChild(s);}());", "", 1)

		result.Requests = append(result.Requests, engine.Request{
			Type: "json",
			Url:  url,
			Text: Text,
			ParserFunc: func(bytes []byte) engine.ParseResult {
				return ParseMemberListProfile(bytes, url)
			},
		})
	}
	return result
}
