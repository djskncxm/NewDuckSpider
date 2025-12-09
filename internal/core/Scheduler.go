package core

import "github.com/djskncxm/NewDuckSpider/pkg/httpc"

type Scheduler struct {
	queue chan *httpc.Request
}

func NewScheduler(cap int) *Scheduler {
	return &Scheduler{
		queue: make(chan *httpc.Request, cap),
	}
}

// 入队
func (s *Scheduler) EnRequest(req *httpc.Request) {
	s.queue <- req
}

// 阻塞取队列
func (s *Scheduler) NextRequest() *httpc.Request {
	req, ok := <-s.queue
	if !ok {
		return nil
	}
	return req
}
