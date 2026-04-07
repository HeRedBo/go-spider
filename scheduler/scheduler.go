package scheduler

import (
	"go-spider/types"
)

// Scheduler 调度器接口
type Scheduler interface {
	Submit(req types.Request)
	WorkerChan() chan types.Request
	types.ReadyNotifier
	Run()
}
