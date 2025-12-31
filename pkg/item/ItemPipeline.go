package item

import (
	"github.com/emirpasic/gods/queues/linkedlistqueue"
	"sync"
)

type ItemPipeline struct {
	ItemQueue *linkedlistqueue.Queue
	mu        sync.Mutex
}

func NewItemPipeline() *ItemPipeline {
	return &ItemPipeline{
		ItemQueue: linkedlistqueue.New(),
	}
}

func (ItemPipeline *ItemPipeline) NextItem() (item *StrictItem) {
	if ItemPipeline.Empty() {
		return nil
	}

	value, ok := ItemPipeline.ItemQueue.Dequeue()

	if !ok {
		return nil
	}

	req, ok := value.(*StrictItem)
	if !ok {
		return nil
	}
	return req
}

func (ItemPipeline *ItemPipeline) EnqueueItem(item *StrictItem) {
	ItemPipeline.mu.Lock()
	defer ItemPipeline.mu.Unlock()
	ItemPipeline.ItemQueue.Enqueue(item)
}

func (ItemPipeline *ItemPipeline) Empty() bool {
	return ItemPipeline.ItemQueue.Empty()
}
