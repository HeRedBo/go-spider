package persist

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"go-spider/3gbizhi/model"
	"go-spider/3gbizhi/parser"
	"go-spider/fetcher"
	"image/jpeg"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"image"

	_ "golang.org/x/image/webp" // 注册 webp 解码器
)

// downloadResult 单张图片下载结果
type downloadResult struct {
	Index     int
	LargeURL  string
	LocalPath string
	ImageName string
	Err       error
}

// FileItemSaver 文件保存器，使用扇入扇出模型并发下载图片并保存 CSV
// concurrency 控制最大并发下载协程数
func FileItemSaver(baseDir string, concurrency int) (chan interface{}, <-chan struct{}, error) {
	imageDir := filepath.Join(baseDir, "image")
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		return nil, nil, fmt.Errorf("创建图片目录失败: %w", err)
	}

	csvPath := filepath.Join(baseDir, "3gbizhi_images.csv")
	csvFile, err := os.Create(csvPath)
	if err != nil {
		return nil, nil, fmt.Errorf("创建CSV文件失败: %w", err)
	}

	// UTF-8 BOM，方便 Excel 正确识别中文
	csvFile.Write([]byte{0xEF, 0xBB, 0xBF})

	csvWriter := csv.NewWriter(csvFile)
	csvWriter.Write([]string{"图集名称", "图片地址", "图片名称", "图片本地路径地址"})
	csvWriter.Flush()

	out := make(chan interface{})
	done := make(chan struct{})

	go downloadLoop(baseDir, csvWriter, csvFile, out, done, concurrency)

	log.Printf("FileItemSaver 启动 | 基础目录: %s | 并发数: %d", baseDir, concurrency)
	return out, done, nil
}

// downloadLoop 主循环：从 itemChan 读取 Album，分发到一级协程处理
func downloadLoop(baseDir string, csvWriter *csv.Writer, csvFile *os.File, itemChan <-chan interface{}, done chan struct{}, concurrency int) {
	sem := make(chan struct{}, concurrency) // 全局信号量
	var mu sync.Mutex                       // 保护 CSV 写入
	var albumWg sync.WaitGroup              // 跟踪所有一级协程

	for item := range itemChan {
		album, ok := item.(model.Album)
		if !ok {
			log.Printf("FileItemSaver: 跳过非 Album 类型: %T", item)
			continue
		}
		albumWg.Add(1)
		go processAlbum(album, baseDir, sem, csvWriter, &mu, &albumWg)
	}

	// channel 关闭后，等待所有 Album 处理完毕
	albumWg.Wait()
	csvWriter.Flush()
	csvFile.Close()
	close(done)
	log.Println("FileItemSaver: 全部下载完成，CSV 已保存")
}

// processAlbum 一级协程：编排单个图集的下载任务，扇入扇出
func processAlbum(album model.Album, baseDir string, sem chan struct{}, csvWriter *csv.Writer, mu *sync.Mutex, albumWg *sync.WaitGroup) {
	defer albumWg.Done()

	if len(album.Images) == 0 {
		log.Printf("图集 [%s] 无图片，跳过", album.Title)
		return
	}

	// 创建图集目录
	albumDir := filepath.Join(baseDir, "image", sanitizeFilename(album.Title))
	if err := os.MkdirAll(albumDir, 0755); err != nil {
		log.Printf("创建图集目录失败 [%s]: %v", album.Title, err)
		return
	}

	// 预分配结果切片（按索引写入，无竞争）
	results := make([]downloadResult, len(album.Images))
	var imgWg sync.WaitGroup

	// Fan-out: 每张图片启动一个二级协程
	for i, img := range album.Images {
		imgWg.Add(1)
		go func(idx int, imgInfo model.ImageInfo) {
			defer imgWg.Done()
			sem <- struct{}{}        // 获取信号量
			defer func() { <-sem }() // 释放信号量

			results[idx] = downloadImage(imgInfo, albumDir)
		}(i, img)
	}

	// Fan-in: 等待该图集所有图片下载完成
	imgWg.Wait()

	// 批量写入 CSV（mutex 保护跨 Album 安全）
	successCount := 0
	mu.Lock()
	for _, r := range results {
		if r.Err != nil {
			log.Printf("  图集 [%s] 图片 #%d 失败: %v", album.Title, r.Index+1, r.Err)
			continue
		}
		csvWriter.Write([]string{album.Title, r.LargeURL, r.ImageName, r.LocalPath})
		successCount++
	}
	csvWriter.Flush()
	mu.Unlock()

	log.Printf("图集 [%s] 完成: %d/%d 张成功", album.Title, successCount, len(album.Images))
}

// downloadImage 二级协程工作函数：获取子页面 → 解析大图URL → 下载图片 → webp转jpg → 保存文件
func downloadImage(imgInfo model.ImageInfo, albumDir string) downloadResult {
	result := downloadResult{Index: imgInfo.Index - 1}

	// 1. 获取子页面 HTML
	body, err := fetcher.FetchWithRetry(imgInfo.SubPageURL, "GET")
	if err != nil {
		result.Err = fmt.Errorf("获取子页面失败: %w", err)
		return result
	}

	// 2. 解析大图 URL
	largeURL := parser.ParseSubPageImageURL(body)
	if largeURL == "" {
		result.Err = fmt.Errorf("未找到大图URL")
		return result
	}
	result.LargeURL = largeURL

	// 3. 下载图片二进制数据
	imgBytes, err := fetcher.FetchBinary(largeURL)
	if err != nil {
		result.Err = fmt.Errorf("下载图片失败: %w", err)
		return result
	}

	// 4. 转换为 JPG 并保存
	origName := extractFilename(largeURL)
	jpgName := replaceExt(origName, ".jpg")
	filename := fmt.Sprintf("%d_%s", imgInfo.Index, jpgName)
	localPath := filepath.Join(albumDir, filename)

	if err := convertToJPG(imgBytes, localPath); err != nil {
		// 转换失败时，直接保存原始文件
		log.Printf("webp→jpg 转换失败, 保存原文件: %v", err)
		fallbackName := fmt.Sprintf("%d_%s", imgInfo.Index, origName)
		localPath = filepath.Join(albumDir, fallbackName)
		filename = fallbackName
		if err := os.WriteFile(localPath, imgBytes, 0644); err != nil {
			result.Err = fmt.Errorf("保存文件失败: %w", err)
			return result
		}
	}

	result.LocalPath = localPath
	result.ImageName = filename
	return result
}

// convertToJPG 将图片字节（webp/png/gif等）解码后转为 JPEG 保存
func convertToJPG(imgBytes []byte, destPath string) error {
	// image.Decode 会自动选择已注册的解码器（webp 通过 import _ 注册）
	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return fmt.Errorf("图片解码失败: %w", err)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	// JPEG 质量 90，保持高画质
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 90}); err != nil {
		return fmt.Errorf("JPEG 编码失败: %w", err)
	}
	return nil
}

// replaceExt 替换文件扩展名
func replaceExt(filename, newExt string) string {
	ext := path.Ext(filename)
	if ext == "" {
		return filename + newExt
	}
	return filename[:len(filename)-len(ext)] + newExt
}

// sanitizeFilename 清理文件名中的非法字符（Windows兼容）
func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		`\`, "_", `/`, "_", `:`, "_", `*`, "_",
		`?`, "_", `"`, "_", `<`, "_", `>`, "_", `|`, "_",
	)
	name = replacer.Replace(strings.TrimSpace(name))
	// 限制长度
	if len([]rune(name)) > 80 {
		name = string([]rune(name)[:80])
	}
	return name
}

// extractFilename 从URL中提取文件名
func extractFilename(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "unknown.webp"
	}
	return path.Base(u.Path)
}
