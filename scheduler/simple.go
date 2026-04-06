package scheduler

import "go-spider/types"

type SimpleScheduler struct {
	workerChan chan types.Request
}

func (s *SimpleScheduler) Submit(r types.Request) {
	go func() { s.workerChan <- r }()
}

func (s *SimpleScheduler) WorkerChan() chan types.Request {
	return s.workerChan
}

func (s *SimpleScheduler) Run() {
	s.workerChan = make(chan types.Request)
}

func (s *SimpleScheduler) WorkerReady(chan types.Request) {
}
