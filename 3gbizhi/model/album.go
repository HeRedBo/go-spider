package model

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Album struct {
	AlbumID    string      `json:"album_id"`
	Title      string      `json:"title"`
	ImageCount int         `json:"image_count"`
	Items      []ImageItem `json:"items"`
	CreateTime time.Time   `json:"create_time"`
}

type ImageItem struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	LocalPath string `json:"local_path"`
}

func (a Album) ID() string {
	return a.AlbumID
}

func (a Album) IsPersistable() bool {
	return true
}

func (a Album) GetDirName() string {
	safeTitle := strings.ReplaceAll(a.Title, "/", "-")
	safeTitle = strings.ReplaceAll(safeTitle, "\\", "-")
	safeTitle = strings.ReplaceAll(safeTitle, " ", "_")
	return safeTitle
}

func (a Album) GetAlbumDir(basePath string) string {
	return filepath.Join(basePath, a.GetDirName())
}

func (a *Album) AddImageItem(item ImageItem) {
	item.LocalPath = filepath.Join(a.GetDirName(), item.Title+getExt(item.URL))
	a.Items = append(a.Items, item)
	a.ImageCount = len(a.Items)
}

func getExt(url string) string {
	ext := filepath.Ext(url)
	if ext == "" || len(ext) > 5 {
		return ".jpg"
	}
	return ext
}

func (a *Album) SetIDFromURL(url string) {
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		a.AlbumID = strings.TrimSuffix(last, ".html")
	}
}

type AlbumListPage struct {
	CurrentPage int    `json:"current_page"`
	TotalPages  int    `json:"total_pages"`
	PageSize    int    `json:"page_size"`
	BaseURL     string `json:"base_url"`
}

func NewAlbumListPage(baseURL string) *AlbumListPage {
	return &AlbumListPage{
		BaseURL:  baseURL,
		PageSize: 20,
	}
}

func (p *AlbumListPage) GetPageURL(page int) string {
	if page == 1 {
		return p.BaseURL
	}
	return p.BaseURL + "index_" + strconv.Itoa(page) + ".html"
}
