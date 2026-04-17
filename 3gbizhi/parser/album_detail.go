package parser

import (
	"go-spider/3gbizhi/model"
	"go-spider/types"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type AlbumDetailParser struct {
	BaseURL string
}

func NewAlbumDetailParser(baseURL string) *AlbumDetailParser {
	return &AlbumDetailParser{BaseURL: baseURL}
}

func ParseAlbumDetail(contents []byte, title string, url string) types.ParseResult {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(contents)))
	if err != nil {
		return types.ParseResult{}
	}

	album := &model.Album{
		Title: title,
	}
	album.SetIDFromURL(url)

	doc.Find(".photo-list li, .pic-list li, .img-list li, .swiper-wrapper .swiper-slide img").Each(func(i int, s *goquery.Selection) {
		imgUrl, _ := s.Attr("src")
		if imgUrl == "" {
			imgUrl, _ = s.Attr("data-src")
		}
		if imgUrl == "" {
			return
		}

		if !strings.HasPrefix(imgUrl, "http") {
			imgUrl = "https://www.3gbizhi.com" + imgUrl
		}

		imgTitle := s.AttrOr("alt", s.Parent().AttrOr("title", "image_"+strconv.Itoa(i+1)))

		item := model.ImageItem{
			Title: imgTitle,
			URL:   imgUrl,
		}
		album.AddImageItem(item)
	})

	if len(album.Items) == 0 {
		doc.Find("img").Each(func(i int, s *goquery.Selection) {
			imgUrl, _ := s.Attr("src")
			if imgUrl == "" {
				imgUrl, _ = s.Attr("data-src")
			}
			if imgUrl == "" || !strings.Contains(imgUrl, "3gbizhi") {
				return
			}

			imgTitle := s.AttrOr("alt", "image_"+strconv.Itoa(i+1))

			item := model.ImageItem{
				Title: imgTitle,
				URL:   imgUrl,
			}
			album.AddImageItem(item)
		})
	}

	result := types.ParseResult{}
	if len(album.Items) > 0 {
		result.Items = append(result.Items, *album)
	}

	return result
}

func ParseAlbumDetailSimple(url string, contents []byte) types.ParseResult {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(contents)))
	if err != nil {
		return types.ParseResult{}
	}

	title := doc.Find("h1, .article-title, .photo-title").First().Text()
	if title == "" {
		title = extractTitleFromURL(url)
	}

	return ParseAlbumDetail(contents, strings.TrimSpace(title), url)
}

func extractTitleFromURL(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		title := strings.TrimSuffix(last, ".html")
		title = strings.ReplaceAll(title, "pic", "")
		return "album_" + title
	}
	return "untitled_album"
}
