package engine

import (
	"go-spider/fetcher"
	"go-spider/types"
	"log"
)

func worker(r types.Request) (types.ParseResult, error) {
	var body []byte
	log.Printf("Fetching type %s: Url: %s", r.Type, r.Url)
	if r.Type == "url" {
		var err error
		body, err = fetcher.Fetch(r.Url, "GET")
		if err != nil {
			log.Printf("Fetcher: error "+" fetching url %s : %s", r.Url, err)
			return types.ParseResult{}, err
		}
	} else if r.Type == "html" || r.Type == "json" {
		var data = []byte(r.Text)
		body = data
	}
	parseResult := r.ParserFunc(body)
	return parseResult, nil
}
