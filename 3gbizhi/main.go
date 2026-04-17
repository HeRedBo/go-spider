package main

import (
	"go-spider/3gbizhi/download"
	"go-spider/3gbizhi/model"
	"go-spider/3gbizhi/parser"
	"go-spider/engine"
	"go-spider/fetcher"
	"go-spider/persist"
	"go-spider/scheduler"
	"go-spider/types"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	BaseURL       = "https://www.3gbizhi.com/meinv/xgmn/"
	MaxPages      = 3
	WorkerCount   = 10
	DownloadCount = 20
	OutputDir     = "images"
	DataFile      = "3gbizhi/data.csv"
)

func main() {
	log.Println("=== 3G壁纸爬虫启动 ===")

	if err := os.MkdirAll("3gbizhi", 0755); err != nil {
		log.Fatalf("创建目录失败: %v", err)
	}

	itemSaver, err := persist.NewFileItemSaver(
		DataFile,
		persist.WithConcurrency(DownloadCount),
		persist.WithImageDir(OutputDir),
	)
	if err != nil {
		log.Fatalf("初始化文件存储失败: %v", err)
	}
	defer itemSaver.Close()

	downloader := download.NewImageDownloader(OutputDir, DownloadCount)

	albumChan := make(chan *model.Album, 100)
	var albumWg sync.WaitGroup

	go func() {
		for album := range albumChan {
			albumWg.Add(1)
			go func(a *model.Album) {
				defer albumWg.Done()

				log.Printf("开始下载图集: %s (%d张图片)", a.Title, len(a.Items))
				_, err := downloader.DownloadAlbumImages(a)
				if err != nil {
					log.Printf("下载图集失败: %s - %v", a.Title, err)
					return
				}

				record := persist.AlbumRecord{
					AlbumID:    a.AlbumID,
					Title:      a.Title,
					ImageCount: a.ImageCount,
					CreateTime: time.Now(),
				}
				for _, item := range a.Items {
					record.Items = append(record.Items, persist.ImageRecord{
						Title:     item.Title,
						URL:       item.URL,
						LocalPath: item.LocalPath,
					})
				}

				itemSaver.Out() <- record
				log.Printf("图集保存完成: %s", a.Title)
			}(album)
		}
		albumWg.Wait()
		closeAlbumSaver(itemSaver)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	e := engine.ConcurrentEngine{
		Scheduler:   &scheduler.QueuedScheduler{},
		WorkerCount: WorkerCount,
		ItemChan:    nil,
	}
	e.WithTimeout(5 * time.Minute)

	go func() {
		seeds := parser.GetListSeeds(BaseURL, MaxPages)
		log.Printf("开始爬取 %d 个列表页面", len(seeds))
		for _, seed := range seeds {
			e.Scheduler.Submit(seed)
		}
	}()

	outChan := make(chan types.ParseResult, 10)
	e.Scheduler.Run()

	for i := 0; i < WorkerCount; i++ {
		createWorker(outChan, e.Scheduler)
	}

albumLoop:
	for {
		select {
		case <-sigChan:
			log.Println("收到退出信号...")
			break albumLoop
		case result, ok := <-outChan:
			if !ok {
				break albumLoop
			}
			for _, item := range result.Items {
				if album, ok := item.(model.Album); ok {
					album.CreateTime = time.Now()
					albumChan <- &album
				}
			}
			for _, req := range result.Requests {
				e.Scheduler.Submit(req)
			}
		}
	}

	log.Println("=== 爬虫结束 ===")
}

func createWorker(out chan types.ParseResult, s scheduler.Scheduler) {
	go func() {
		for {
			r := s.WorkerChan()
			req := <-r
			result := worker(req)
			out <- result
			s.WorkerChan() <- req
		}
	}()
}

func worker(r types.Request) types.ParseResult {
	var body []byte
	log.Printf("Fetching: %s", r.Url)
	body, err := fetcher.Fetch(r.Url, "GET")
	if err != nil {
		log.Printf("Fetch error: %s - %v", r.Url, err)
		return types.ParseResult{}
	}
	return r.ParserFunc(body)
}

func closeAlbumSaver(saver *persist.FileItemSaver) {
	if saver != nil {
		saver.Close()
	}
}
