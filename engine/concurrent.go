package engine

import (
	"context"
	"go-spider/scheduler"
	"go-spider/types"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gookit/goutil/dump"
)

type ConcurrentEngine struct {
	Scheduler   scheduler.Scheduler
	WorkerCount int // 工人协程数
	ItemChan    chan interface{}

	activeReq int           // 新增：记录活跃请求数，用于判断是否全部结束
	timeout   time.Duration // 全局超时时间
}

// 对外设置超时
func (e *ConcurrentEngine) WithTimeout(d time.Duration) {
	e.timeout = d
}

func (e *ConcurrentEngine) Run(seeds ...types.Request) {

	// ========== 1. 创建上下文：支持 超时 + 优雅退出 ==========
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)

	if e.timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), e.timeout)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	// ========== 2. 监听系统信号：Ctrl+C 优雅退出 ==========
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		dump.P("\n=== 收到退出信号，开始优雅关闭爬虫 ===")
		cancel() // 触发所有协程退出
	}()

	// 3. 启动调度器 + 协程
	outChan := make(chan types.ParseResult, 10)
	e.Scheduler.Run()

	// 创建工人协程
	for i := 0; i < e.WorkerCount; i++ {
		createWorker(ctx, e.Scheduler.WorkerChan(), outChan, e.Scheduler)
	}

	// 4. 提交种子请求
	for _, r := range seeds {
		e.activeReq++
		e.Scheduler.Submit(r)
	}
	//5. 等待组：确保所有数据发送完毕
	var itemWg sync.WaitGroup
	ProfileCount := 0
	// 主循环: 处理结果
	for {
		select {
		case <-ctx.Done():
			dump.P("=== 超时/信号，退出主循环 ===")
			goto END
		case result, ok := <-outChan:
			if !ok {
				dump.P("=== outChan 已关闭，退出 ===")
				goto END
			}
			// 处理 item
			for _, item := range result.Items {
				//dump.P("Got item: %v", item)
				//fmt.Printf("Got item: %v", item)
				if p, ok := item.(types.Persistable); ok && p.IsPersistable() {
					//fmt.Printf("Got CityProfile Item #%d %v", ProfileCount, item)
					//dump.P("Got CityProfile Item #%d %v", ProfileCount, item)
					dump.P("抓取到用户 #%d: %+v", ProfileCount, item)
					ProfileCount++
					// 发送数据并等待完成
					itemWg.Add(1)
					go func(v interface{}) {
						defer itemWg.Done()
						e.ItemChan <- v
					}(item)

				}
			}
			// 提交新请求
			for _, request := range result.Requests {
				e.activeReq++
				e.Scheduler.Submit(request)
			}

			// 没处理完一个请求结果 活跃数 - 1
			e.activeReq--
			if e.activeReq == 0 {
				dump.P("=== 爬虫全部完成，正常退出 ===")
				goto END
			}
		}
	}
	//最终：优雅关闭
END:
	// 1. 关闭调度器
	// 2. 等待所有数据发送到 ItemChan
	itemWg.Wait()
	//// 关闭 ItemChan → 触发 ItemSaver 自动 flush 剩余数据
	close(e.ItemChan)
	// 4. 【关键】等待批量保存完成（给1.5秒，确保ES写入）
	time.Sleep(1500 * time.Millisecond)
	dump.P("=== 爬虫程序全部结束，数据处理逻辑完成 ===")
}

// createWorker 传入 ctx 支持主动退出
func createWorker(ctx context.Context, in chan types.Request, out chan types.ParseResult, ready types.ReadyNotifier) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				// 接收到退出信号，worker 直接退出
				return
			default:

			}
			ready.WorkerReady(in)
			//request := <-in // 注意死锁 没有任务了永远阻塞
			// 管道关闭后自动退出
			request, ok := <-in
			if !ok {
				return
			}
			// 执行爬取任务
			res, err := worker(request)
			if err != nil {
				continue
			}

			// 发送结果（支持 ctx 退出）
			select {
			case out <- res:
			case <-ctx.Done():
				return
			}
		}
	}()
}
