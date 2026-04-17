package parser

import (
	"go-spider/3gbizhi/model"
	"go-spider/types"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	ListPageMaxPages = 5
)

type AlbumListParser struct {
	baseURL string
}

func NewAlbumListParser(baseURL string) *AlbumListParser {
	return &AlbumListParser{baseURL: baseURL}
}

func ParseAlbumList(contents []byte) types.ParseResult {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(contents)))
	if err != nil {
		return types.ParseResult{}
	}

	result := types.ParseResult{}

	doc.Find(".pic-list li, .list-box li, .photo-list li").Each(func(i int, s *goquery.Selection) {
		link := s.Find("a")
		href, exists := link.Attr("href")
		if !exists {
			return
		}

		fullURL := href
		if !strings.HasPrefix(href, "http") {
			fullURL = "https://www.3gbizhi.com" + href
		}

		title := link.Find("img").AttrOr("alt", "")
		if title == "" {
			title = link.AttrOr("title", "untitled")
		}

		result.Items = append(result.Items, "Album: "+title)

		result.Requests = append(result.Requests, types.Request{
			Type: "url",
			Url:  fullURL,
			ParserFunc: func(bytes []byte) types.ParseResult {
				return ParseAlbumDetail(bytes, title, fullURL)
			},
		})
	})

	totalPages := parseTotalPages(doc)
	for page := 2; page <= ListPageMaxPages && page <= totalPages; page++ {
		pageURL := getPageURL(page)
		result.Requests = append(result.Requests, types.Request{
			Type: "url",
			Url:  pageURL,
			ParserFunc: func(bytes []byte) types.ParseResult {
				return ParseAlbumList(bytes)
			},
		})
	}

	return result
}

func (p *AlbumListParser) GetPageURL(page int) string {
	if page == 1 {
		return p.baseURL
	}
	return p.baseURL + "index_" + strconv.Itoa(page) + ".html"
}

func parseTotalPages(doc *goquery.Document) int {
	pages := 0
	doc.Find(".page-nav a, .pages a, .pagination a").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if num, err := strconv.Atoi(strings.TrimSpace(text)); err == nil && num > pages {
			pages = num
		}
	})
	if pages == 0 {
		pages = ListPageMaxPages
	}
	return pages
}

func getPageURL(page int) string {
	base := "https://www.3gbizhi.com/meinv/xgmn"
	if page == 1 {
		return base + "/"
	}
	return base + "/index_" + strconv.Itoa(page) + ".html"
}

func ParseAlbumListSimple(url string, contents []byte) types.ParseResult {
	return ParseAlbumList(contents)
}

func BuildAlbumListRequests(baseURL string, maxPages int) []types.Request {
	var requests []types.Request
	for page := 1; page <= maxPages; page++ {
		var pageURL string
		if page == 1 {
			pageURL = baseURL
		} else {
			pageURL = baseURL + "index_" + strconv.Itoa(page) + ".html"
		}
		requests = append(requests, types.Request{
			Type: "url",
			Url:  pageURL,
			ParserFunc: func(bytes []byte) types.ParseResult {
				return ParseAlbumList(bytes)
			},
		})
	}
	return requests
}

func GetListSeeds(baseURL string, maxPages int) []types.Request {
	albumList := model.NewAlbumListPage(baseURL)
	var requests []types.Request
	for page := 1; page <= maxPages; page++ {
		requests = append(requests, types.Request{
			Type: "url",
			Url:  albumList.GetPageURL(page),
			ParserFunc: func(bytes []byte) types.ParseResult {
				return ParseAlbumList(bytes)
			},
		})
	}
	return requests
}
