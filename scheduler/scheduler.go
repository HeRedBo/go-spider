package scheduler

import (
	"go-spider/types"
)

// Scheduler 调度器接口
type Scheduler interface {
	Submit(req types.Request)
	WorkerChan() chan types.Request
	ReadyNotifier
	Run()
}

type ReadyNotifier interface {
	WorkerReady(chan types.Request)
}
