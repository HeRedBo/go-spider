package engine

import (
	"go-spider/scheduler"
	"go-spider/types"

	"github.com/gookit/goutil/dump"
)

type ConcurrentEngine struct {
	Scheduler   scheduler.Scheduler
	WorkerCount int // 工人协程数
}

func (e *ConcurrentEngine) Run(seeds ...types.Request) {
	outChan := make(chan types.ParseResult)
	e.Scheduler.Run()

	// 创建工人协程
	for i := 0; i < e.WorkerCount; i++ {
		createWorker(e.Scheduler.WorkerChan(), outChan, e.Scheduler)
	}

	// 提交请求
	for _, r := range seeds {
		e.Scheduler.Submit(r)
	}

	for {
		result := <-outChan
		for _, item := range result.Items {
			dump.P("Got item: %v", item)
			//fmt.Printf("Got item: %v", item)
		}

		for _, request := range result.Requests {

			e.Scheduler.Submit(request)
		}
	}

}

func createWorker(in chan types.Request, out chan types.ParseResult, ready types.ReadyNotifier) {
	go func() {
		for {
			ready.WorkerReady(in)
			request := <-in
			result, err := worker(request)
			if err != nil {
				continue
			}
			out <- result
		}
	}()
}
