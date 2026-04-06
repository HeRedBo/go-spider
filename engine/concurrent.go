package engine

import (
	"context"
	"go-spider/scheduler"
	"go-spider/types"
	"go-spider/zhenai/model"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gookit/goutil/dump"
)

type ConcurrentEngine struct {
	Scheduler   scheduler.Scheduler
	WorkerCount int // 工人协程数

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

	outChan := make(chan types.ParseResult)
	e.Scheduler.Run()

	// 创建工人协程
	for i := 0; i < e.WorkerCount; i++ {
		createWorker(ctx, e.Scheduler.WorkerChan(), outChan, e.Scheduler)
	}

	// 提交请求
	for _, r := range seeds {
		e.activeReq++
		e.Scheduler.Submit(r)
	}
	ProfileCount := 0

	// 主循环:接收结果
	for result := range outChan {
		// 处理 item
		for _, item := range result.Items {
			dump.P("Got item: %v", item)
			//fmt.Printf("Got item: %v", item)
			if _, ok := item.(model.Member); ok {
				ProfileCount++
				dump.P(ProfileCount)
				//log.Printf("Got CityProfile Item #%d %v", ProfileCount, item)
			}
		}

		for _, request := range result.Requests {
			e.activeReq++
			e.Scheduler.Submit(request)
		}

		// 没处理完一个请求结果 活跃数 - 1
		e.activeReq--

		if e.activeReq == 0 {
			close(outChan)
			break
		}
		dump.P("=== 爬虫全部完成，正常退出 ===")
	}

	for {
		result := <-outChan // 注意程序死锁，任务跑完后永远收不到数据
		for _, item := range result.Items {
			dump.P("Got item: %v", item)
			//fmt.Printf("Got item: %v", item)
			if _, ok := item.(model.Member); ok {
				ProfileCount++
				dump.P(ProfileCount)
				//log.Printf("Got CityProfile Item #%d %v", ProfileCount, item)
			}
		}

		for _, request := range result.Requests {
			e.Scheduler.Submit(request)
		}
		// 请求和item都为空时，退出循环
		if len(result.Requests) == 0 && len(result.Items) == 0 {
			break
		}
	}

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
			//requestm := <-in // 注意死锁 没有任务了永远阻塞
			// 管道关闭后自动退出
			request, ok := <-in
			if !ok {
				return
			}
			// 执行爬取任务
			result, err := worker(request)
			if err != nil {
				continue
			}

			// 发送结果（支持 ctx 退出）
			select {
			case out <- result:
			case <-ctx.Done():
				return
			}
		}
	}()
}
