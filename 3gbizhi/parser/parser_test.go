package parser

import (
	"go-spider/3gbizhi/model"
	"os"
	"strings"
	"testing"
)

func TestParseListPage_PageLimit1(t *testing.T) {
	contents, err := os.ReadFile("../../fetcher/3gbizhi_index.html")
	if err != nil {
		t.Fatalf("读取测试HTML失败: %v", err)
	}

	ResetState()
	PageLimit = 1
	result := ParseListPage(contents)

	// PageLimit=1 只爬第1页，不生成下一页请求，只有图集详情请求
	t.Logf("PageLimit=1: 共 %d 个请求", len(result.Requests))
	if len(result.Requests) == 0 {
		t.Fatal("未解析到任何图集请求")
	}

	// 检查所有请求都是图集详情页（不应有分页请求）
	for _, r := range result.Requests {
		if strings.Contains(r.Url, "index_") {
			t.Errorf("PageLimit=1 不应生成分页请求, 但发现: %s", r.Url)
		}
	}

	first := result.Requests[0]
	if first.Type != "url" || first.Url == "" {
		t.Errorf("第一个请求异常: Type=%s, Url=%s", first.Type, first.Url)
	}
	t.Logf("第一个图集URL: %s", first.Url)
}

func TestParseListPage_PageLimit3(t *testing.T) {
	contents, err := os.ReadFile("../../fetcher/3gbizhi_index.html")
	if err != nil {
		t.Fatalf("读取测试HTML失败: %v", err)
	}

	ResetState()
	PageLimit = 3
	result := ParseListPage(contents)

	// 第1页: 应该有图集请求 + 1个"下一页"分页请求（从页面内容提取）
	t.Logf("PageLimit=3: 共 %d 个请求", len(result.Requests))

	nextPageRequests := 0
	for _, r := range result.Requests {
		if strings.Contains(r.Url, "index_2") {
			nextPageRequests++
			t.Logf("发现下一页请求: %s", r.Url)
		}
	}
	if nextPageRequests != 1 {
		t.Errorf("期望 1 个下一页请求（从页面提取）, 实际 %d", nextPageRequests)
	}
}

func TestParseListPage_PageLimit0(t *testing.T) {
	contents, err := os.ReadFile("../../fetcher/3gbizhi_index.html")
	if err != nil {
		t.Fatalf("读取测试HTML失败: %v", err)
	}

	ResetState()
	PageLimit = 0
	result := ParseListPage(contents)

	// 不限制模式: 应该有图集请求 + 1个"下一页"请求（从分页栏提取）
	t.Logf("PageLimit=0: 共 %d 个请求", len(result.Requests))

	last := result.Requests[len(result.Requests)-1]
	if !strings.Contains(last.Url, "index_2") {
		t.Errorf("期望最后一个请求包含 index_2, 实际=%s", last.Url)
	}
	t.Logf("下一页URL: %s", last.Url)
}

func TestExtractNextPageURL(t *testing.T) {
	contents, err := os.ReadFile("../../fetcher/3gbizhi_index.html")
	if err != nil {
		t.Fatalf("读取测试HTML失败: %v", err)
	}

	nextURL := extractNextPageURL(contents)
	if nextURL == "" {
		t.Fatal("未提取到下一页URL")
	}
	t.Logf("下一页URL: %s", nextURL)

	if !strings.Contains(nextURL, "index_2") {
		t.Errorf("期望下一页URL包含 index_2, 实际=%s", nextURL)
	}
	if !strings.HasPrefix(nextURL, "https://") {
		t.Errorf("期望绝对URL, 实际=%s", nextURL)
	}
}

func TestURLDedup(t *testing.T) {
	ResetState()

	url1 := "https://www.3gbizhi.com/meinv/index_2.html?sort=latest&page=2"
	url2 := "https://www.3gbizhi.com/meinv/index_2.html?sort=latest&page=2"
	url3 := "https://www.3gbizhi.com/meinv/index_3.html?sort=latest&page=3"

	// 首次访问应返回 true
	if !markVisited(url1) {
		t.Error("首次访问 url1 应返回 true")
	}
	// 重复访问应返回 false（路径去重，忽略 query）
	if markVisited(url2) {
		t.Error("重复访问 url2 应返回 false")
	}
	// 不同路径应返回 true
	if !markVisited(url3) {
		t.Error("首次访问 url3 应返回 true")
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://www.3gbizhi.com/meinv/index_2.html?sort=latest&page=2", "/meinv/index_2.html"},
		{"https://www.3gbizhi.com/meinv/index_3.html?sort=latest&page=3", "/meinv/index_3.html"},
		{"/meinv/index_2.html?sort=latest&page=2", "/meinv/index_2.html"},
	}
	for _, tt := range tests {
		got := normalizeURL(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeURL(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestParseDetail(t *testing.T) {
	contents, err := os.ReadFile("../../fetcher/pic2284_detail.html")
	if err != nil {
		t.Fatalf("读取测试HTML失败: %v", err)
	}

	result := ParseDetail(contents, "2284", "测试标题", "https://example.com/cover.webp", "https://www.3gbizhi.com/meinv/xgmn/pic2284.html")

	if len(result.Items) == 0 {
		t.Fatal("ParseDetail 未解析到 Album Item")
	}

	album, ok := result.Items[0].(model.Album)
	if !ok {
		t.Fatal("Item 不是 Album 类型")
	}

	t.Logf("图集标题: %s", album.Title)
	t.Logf("图集ID: %s", album.AlbumID)
	t.Logf("标签: %v", album.Tags)
	t.Logf("图片数量: %d", len(album.Images))

	if album.AlbumID != "2284" {
		t.Errorf("期望 AlbumID=2284, 实际=%s", album.AlbumID)
	}
	if album.Title == "" {
		t.Error("标题不应为空")
	}
	if len(album.Images) == 0 {
		t.Fatal("图片列表不应为空")
	}

	for i, img := range album.Images {
		t.Logf("  图片 #%d: SubPageURL=%s", img.Index, img.SubPageURL)
		if img.SubPageURL == "" {
			t.Errorf("图片 #%d SubPageURL 为空", i+1)
		}
	}

	if len(result.Requests) != 0 {
		t.Errorf("ParseDetail 不应生成新请求, 实际=%d", len(result.Requests))
	}
}

func TestParseSubPageImageURL(t *testing.T) {
	contents, err := os.ReadFile("../../fetcher/pic2284_detail.html")
	if err != nil {
		t.Fatalf("读取测试HTML失败: %v", err)
	}

	url := ParseSubPageImageURL(contents)
	if url == "" {
		t.Fatal("未解析到大图URL")
	}
	t.Logf("大图URL: %s", url)
}
