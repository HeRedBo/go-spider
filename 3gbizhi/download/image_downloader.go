package download

import (
	"context"
	"fmt"
	"go-spider/3gbizhi/model"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/avast/retry-go"
)

type ImageDownloader struct {
	client      *http.Client
	concurrency int
	outputDir   string
	workers     int
}

type ImageTask struct {
	URL      string
	Title    string
	SaveDir  string
	AlbumDir string
}

type DownloadResult struct {
	Task     ImageTask
	LocalPath string
	Err      error
}

func NewImageDownloader(outputDir string, concurrency int) *ImageDownloader {
	return &ImageDownloader{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		concurrency: concurrency,
		outputDir:   outputDir,
		workers:     concurrency,
	}
}

func (d *ImageDownloader) SetConcurrency(n int) {
	if n > 0 {
		d.concurrency = n
		d.workers = n
	}
}

func (d *ImageDownloader) GetConcurrency() int {
	return d.concurrency
}

func (d *ImageDownloader) DownloadSingle(url, title, albumDir string) (string, error) {
	saveDir := filepath.Join(d.outputDir, albumDir)
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return "", err
	}

	filename := d.sanitizeFilename(title) + d.getExt(url)
	savePath := filepath.Join(saveDir, filename)

	if _, err := os.Stat(savePath); err == nil {
		return savePath, nil
	}

	var body []byte
	err := retry.Do(
		func() error {
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				return err
			}
			req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

			resp, err := d.client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return fmt.Errorf("status code: %d", resp.StatusCode)
			}

			body, err = io.ReadAll(resp.Body)
			return err
		},
		retry.Attempts(3),
		retry.Delay(500*time.Millisecond),
	)

	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}

	if err := os.WriteFile(savePath, body, 0644); err != nil {
		return "", fmt.Errorf("save file failed: %w", err)
	}

	return savePath, nil
}

func (d *ImageDownloader) DownloadAlbumImages(album *model.Album) ([]model.ImageItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	albumDir := album.GetDirName()
	saveDir := filepath.Join(d.outputDir, albumDir)
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return nil, err
	}

	tasks := make(chan ImageTask, len(album.Items))
	results := make(chan DownloadResult, len(album.Items))

	var wg sync.WaitGroup

	for i := 0; i < d.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d.worker(ctx, tasks, results)
		}()
	}

	for _, item := range album.Items {
		tasks <- ImageTask{
			URL:      item.URL,
			Title:    item.Title,
			SaveDir:  saveDir,
			AlbumDir: albumDir,
		}
	}
	close(tasks)

	go func() {
		wg.Wait()
		close(results)
	}()

	updatedItems := make([]model.ImageItem, 0, len(album.Items))
	successCount := 0

	for result := range results {
		if result.Err != nil {
			log.Printf("download error: %s - %v", result.Task.URL, result.Err)
			continue
		}
		relPath, _ := filepath.Rel(d.outputDir, result.LocalPath)
		updatedItems = append(updatedItems, model.ImageItem{
			Title:     result.Task.Title,
			URL:       result.Task.URL,
			LocalPath: relPath,
		})
		successCount++
		log.Printf("downloaded: %s -> %s", result.Task.Title, relPath)
	}

	log.Printf("album %s: %d/%d images downloaded", album.Title, successCount, len(album.Items))

	album.Items = updatedItems
	album.ImageCount = len(updatedItems)

	return updatedItems, nil
}

func (d *ImageDownloader) worker(ctx context.Context, tasks <-chan ImageTask, results chan<- DownloadResult) {
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-tasks:
			if !ok {
				return
			}
			result := d.processTask(task)
			results <- result
		}
	}
}

func (d *ImageDownloader) processTask(task ImageTask) DownloadResult {
	filename := d.sanitizeFilename(task.Title) + d.getExt(task.URL)
	savePath := filepath.Join(task.SaveDir, filename)

	if _, err := os.Stat(savePath); err == nil {
		return DownloadResult{Task: task, LocalPath: savePath}
	}

	var body []byte
	err := retry.Do(
		func() error {
			req, err := http.NewRequest(http.MethodGet, task.URL, nil)
			if err != nil {
				return err
			}
			req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

			resp, err := d.client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return fmt.Errorf("status code: %d", resp.StatusCode)
			}

			body, err = io.ReadAll(resp.Body)
			return err
		},
		retry.Attempts(3),
		retry.Delay(500*time.Millisecond),
	)

	if err != nil {
		return DownloadResult{Task: task, Err: fmt.Errorf("download failed: %w", err)}
	}

	if err := os.WriteFile(savePath, body, 0644); err != nil {
		return DownloadResult{Task: task, Err: fmt.Errorf("save file failed: %w", err)}
	}

	return DownloadResult{Task: task, LocalPath: savePath}
}

func (d *ImageDownloader) sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, "*", "-")
	name = strings.ReplaceAll(name, "?", "-")
	name = strings.ReplaceAll(name, `"`, "-")
	name = strings.ReplaceAll(name, "<", "-")
	name = strings.ReplaceAll(name, ">", "-")
	name = strings.ReplaceAll(name, "|", "-")
	name = strings.TrimSpace(name)
	if name == "" {
		name = "untitled"
	}
	return name
}

func (d *ImageDownloader) getExt(url string) string {
	ext := filepath.Ext(url)
	if ext == "" || len(ext) > 5 {
		return ".jpg"
	}
	if !strings.HasPrefix(ext, ".") {
		return "." + ext
	}
	return ext
}

type BatchDownloader struct {
	downloader *ImageDownloader
	resultChan chan *model.Album
	errChan    chan error
}

func NewBatchDownloader(outputDir string, concurrency int) *BatchDownloader {
	return &BatchDownloader{
		downloader: NewImageDownloader(outputDir, concurrency),
		resultChan: make(chan *model.Album, 100),
		errChan:    make(chan error, 100),
	}
}

func (b *BatchDownloader) DownloadAlbums(albums []*model.Album) []*model.Album {
	var wg sync.WaitGroup
	var mu sync.Mutex
	completed := make([]*model.Album, 0, len(albums))

	for _, album := range albums {
		wg.Add(1)
		go func(a *model.Album) {
			defer wg.Done()

			_, err := b.downloader.DownloadAlbumImages(a)
			if err != nil {
				b.errChan <- fmt.Errorf("album %s download error: %w", a.Title, err)
				return
			}

			mu.Lock()
			completed = append(completed, a)
			mu.Unlock()

			b.resultChan <- a
		}(album)
	}

	go func() {
		wg.Wait()
		close(b.resultChan)
		close(b.errChan)
	}()

	return completed
}

func (b *BatchDownloader) GetResultChan() <-chan *model.Album {
	return b.resultChan
}

func (b *BatchDownloader) GetErrChan() <-chan error {
	return b.errChan
}
