package scheduler

import (
	"go-spider/engine"
)

// Scheduler 调度器接口
type Scheduler interface {
	Submit(req engine.Request)
	WorkerChan() chan engine.Request
	ReadyNotifier
	Run()
}

type ReadyNotifier interface {
	WorkerReady(chan engine.Request)
}
