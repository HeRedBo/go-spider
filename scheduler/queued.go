package scheduler

import "go-spider/types"

type QueuedScheduler struct {
	requestChan chan types.Request
	workerChan  chan chan types.Request
}

func (s *QueuedScheduler) Submit(r types.Request) { // 提交请求
	s.requestChan <- r
}

func (s *QueuedScheduler) WorkerChan() chan types.Request {
	return make(chan types.Request)
}

func (s *QueuedScheduler) WorkerReady(w chan types.Request) {
	s.workerChan <- w
}

func (s *QueuedScheduler) Run() {
	s.workerChan = make(chan chan types.Request)
	s.requestChan = make(chan types.Request)
	// 核心逻辑
	go func() {
		var requestQ []types.Request
		var workersQ []chan types.Request

		for {

			var activeRequest types.Request
			var activeWorker chan types.Request

			if len(requestQ) > 0 && len(workersQ) > 0 {
				activeRequest = requestQ[0]
				activeWorker = workersQ[0]
			}

			select {
			case r := <-s.requestChan:
				requestQ = append(requestQ, r)
			case w := <-s.workerChan:
				workersQ = append(workersQ, w)
			case activeWorker <- activeRequest:
				workersQ = workersQ[1:]
				requestQ = requestQ[1:]
			}

			//
		}

	}()

}
