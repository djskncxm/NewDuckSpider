package core

import "github.com/djskncxm/NewDuckSpider/pkg/httpc"

type Scheduler struct {
	queue chan *httpc.Request
}

// NewScheduler 创建指定缓冲大小的调度器
func NewScheduler(bufferSize int) *Scheduler {
	return &Scheduler{
		queue: make(chan *httpc.Request, bufferSize),
	}
}

// NextRequest 非阻塞获取请求
func (s *Scheduler) NextRequest() *httpc.Request {
	select {
	case req := <-s.queue:
		return req
	default:
		return nil
	}
}

// EnqueueRequest 入队请求（可能阻塞直到有空闲缓冲）
func (s *Scheduler) EnqueueRequest(req *httpc.Request) {
	s.queue <- req
}

// Empty 判断队列是否为空（快照，非精确）
func (s *Scheduler) Empty() bool {
	return len(s.queue) == 0
}

// NextRequestBlocking 阻塞获取请求，直到有请求或者队列关闭
func (s *Scheduler) NextRequestBlocking() (*httpc.Request, bool) {
	req, ok := <-s.queue
	return req, ok
}

// CloseScheduler 关闭队列，通知 worker 可以退出
func (s *Scheduler) CloseScheduler() {
	close(s.queue)
}
