package core

import (
	"github.com/djskncxm/NewDuckSpider/pkg/httpc"
	"github.com/emirpasic/gods/queues/linkedlistqueue"
	"sync"
)

type Scheduler struct {
	RequestQueue *linkedlistqueue.Queue
	mu           sync.Mutex
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		RequestQueue: linkedlistqueue.New(),
	}
}

func (scheduler *Scheduler) NextRequest() (request *httpc.Request) {
	if scheduler.Empty() {
		return nil
	}

	value, ok := scheduler.RequestQueue.Dequeue()

	if !ok {
		return nil
	}

	req, ok := value.(*httpc.Request)
	if !ok {
		return nil
	}
	return req
}

func (scheduler *Scheduler) EnqueueRequest(request *httpc.Request) {
	scheduler.mu.Lock()
	defer scheduler.mu.Unlock()
	scheduler.RequestQueue.Enqueue(request)
}

func (scheduler *Scheduler) Empty() bool {
	return scheduler.RequestQueue.Empty()
}
