package parser

import (
	"bytes"
	"go-spider/3gbizhi/model"
	"go-spider/types"
	"log"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/PuerkitoBio/goquery"
)

var albumIDRe = regexp.MustCompile(`pic(\d+)`)

// PageLimit 列表页爬取页数限制
// 0 = 不限制，自动跟随下一页直到无数据
// >0 = 只爬取指定页数（例如 3 表示从种子页开始爬取3页）
var PageLimit = 1

// visited 已访问列表页 URL 去重集合（线程安全）
var visited sync.Map

// pageCount 已爬取的列表页计数（原子操作）
var pageCount int32

// ResetState 重置爬虫状态（用于测试或重新启动）
func ResetState() {
	visited = sync.Map{}
	atomic.StoreInt32(&pageCount, 0)
}

// markVisited 标记 URL 为已访问，返回 true 表示首次访问
func markVisited(rawURL string) bool {
	_, loaded := visited.LoadOrStore(normalizeURL(rawURL), true)
	return !loaded
}

// normalizeURL 标准化 URL，用路径去重（忽略 query 参数）
func normalizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Path
}

// ParseListPage 通用列表页解析器
// - 从当前页面提取图集列表
// - 从页面底部分页栏提取"下一页"链接（非硬编码 URL）
// - 支持 URL 去重，避免重复爬取
// - 支持 PageLimit 控制最大爬取页数
// - 支持从任意页开始（种子 URL 决定起始页）
func ParseListPage(contents []byte) types.ParseResult {
	current := atomic.AddInt32(&pageCount, 1)
	result := parseAlbumList(contents)
	albumCount := len(result.Requests)
	log.Printf("ParseListPage: 第%d页发现 %d 个图集 (PageLimit=%d)", current, albumCount, PageLimit)

	// 达到页数限制，停止分页
	if PageLimit > 0 && int(current) >= PageLimit {
		log.Printf("ParseListPage: 已达到 PageLimit=%d，停止分页", PageLimit)
		return result
	}

	// 从页面内容提取"下一页"链接
	nextURL := extractNextPageURL(contents)
	if nextURL == "" {
		log.Printf("ParseListPage: 未找到下一页链接，分页结束")
		return result
	}

	// URL 去重检查
	if !markVisited(nextURL) {
		log.Printf("ParseListPage: 下一页已访问过，跳过: %s", nextURL)
		return result
	}

	log.Printf("ParseListPage: 下一页 -> %s", nextURL)
	result.Requests = append(result.Requests, types.Request{
		Type:       "url",
		Url:        nextURL,
		ParserFunc: ParseListPage,
	})

	return result
}

// extractNextPageURL 从分页栏 HTML 提取"下一页"链接
// 选择器: div.pagination a[title="下一页"]
// 如果 href 为 javascript:; 表示已是最后一页，返回空
func extractNextPageURL(contents []byte) string {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(contents))
	if err != nil {
		return ""
	}

	var nextURL string
	doc.Find("div.pagination a").Each(func(i int, s *goquery.Selection) {
		title, _ := s.Attr("title")
		if title == "下一页" {
			href, exists := s.Attr("href")
			if exists && href != "" && href != "javascript:;" {
				nextURL = href
			}
		}
	})

	if nextURL == "" {
		return ""
	}

	// 相对 URL 转绝对 URL
	if strings.HasPrefix(nextURL, "/") {
		nextURL = "https://www.3gbizhi.com" + nextURL
	}

	return nextURL
}

// parseAlbumList 提取单页中的所有图集（纯解析逻辑，不处理分页）
func parseAlbumList(contents []byte) types.ParseResult {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(contents))
	if err != nil {
		log.Printf("parseAlbumList: goquery parse error: %v", err)
		return types.ParseResult{}
	}

	var result types.ParseResult
	doc.Find("div.contlistw").First().Find("li.box_black").Each(func(i int, s *goquery.Selection) {
		detailURL, exists := s.Find("a.imgw").Attr("href")
		if !exists || detailURL == "" {
			return
		}

		matches := albumIDRe.FindStringSubmatch(detailURL)
		if len(matches) < 2 {
			return
		}
		albumID := matches[1]
		title := strings.TrimSpace(s.Find("div.title div.text").Text())
		coverURL, _ := s.Find("a.imgw img").Attr("lay-src")

		result.Requests = append(result.Requests, types.Request{
			Type: "url",
			Url:  detailURL,
			ParserFunc: func(b []byte) types.ParseResult {
				return ParseDetail(b, albumID, title, coverURL, detailURL)
			},
		})
	})
	return result
}

// ParseDetail 解析图集详情页，提取图片列表，返回 Album Item
func ParseDetail(contents []byte, albumID, indexTitle, coverURL, detailURL string) types.ParseResult {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(contents))
	if err != nil {
		log.Printf("ParseDetail: goquery parse error: %v", err)
		return types.ParseResult{}
	}

	title := strings.TrimSpace(doc.Find("div.titlew div.item-title h2.title").Text())
	if title == "" {
		title = indexTitle
	}

	var tags []string
	doc.Find("div.showtaglistw a").Each(func(i int, s *goquery.Selection) {
		if tag := strings.TrimSpace(s.Text()); tag != "" {
			tags = append(tags, tag)
		}
	})

	var images []model.ImageInfo
	doc.Find("div.showImglistw div.row div.col").Each(func(i int, s *goquery.Selection) {
		subPageURL, _ := s.Find("a").Attr("href")
		thumbURL, _ := s.Find("img").Attr("src")
		if subPageURL != "" {
			images = append(images, model.ImageInfo{
				SubPageURL: subPageURL,
				ThumbURL:   thumbURL,
				Index:      i + 1,
			})
		}
	})

	album := model.Album{
		AlbumID:   albumID,
		Title:     title,
		DetailURL: detailURL,
		CoverURL:  coverURL,
		Tags:      tags,
		Images:    images,
	}

	log.Printf("ParseDetail: 图集 [%s] 共 %d 张图片", title, len(images))
	return types.ParseResult{
		Items: []interface{}{album},
	}
}

// ParseSubPageImageURL 从子页面 HTML 中提取大图 URL
func ParseSubPageImageURL(contents []byte) string {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(contents))
	if err != nil {
		return ""
	}
	src, _ := doc.Find("img#contpic").Attr("src")
	return src
}
