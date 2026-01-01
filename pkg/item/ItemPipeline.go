package item

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emirpasic/gods/queues/linkedlistqueue"
)

var (
	// ErrPipelineClosed 表示管道已关闭
	ErrPipelineClosed = errors.New("pipeline is closed")
	// ErrPipelineFull 表示管道已满
	ErrPipelineFull = errors.New("pipeline is full")
	// ErrItemInvalid 表示项目无效
	ErrItemInvalid = errors.New("item is invalid")
)

// PipelineConfig 管道配置
type PipelineConfig struct {
	MaxSize       int           // 最大容量，0表示无限制
	MaxWaitTime   time.Duration // 入队最大等待时间，0表示无限等待
	AutoFlushSize int           // 自动刷新大小，达到此数量后触发刷新
}

// PipelineStats 管道统计信息
type PipelineStats struct {
	TotalEnqueued   int64         // 总入队数量
	TotalDequeued   int64         // 总出队数量
	TotalProcessed  int64         // 总处理数量
	TotalFailed     int64         // 失败数量
	CurrentSize     int           // 当前大小
	AvgProcessTime  time.Duration // 平均处理时间
	LastProcessTime time.Time     // 最后处理时间
}

// PipelineCallback 管道回调函数
type PipelineCallback struct {
	OnItemEnqueued  func(item *StrictItem)            // 项目入队时调用
	OnItemDequeued  func(item *StrictItem)            // 项目出队时调用
	OnItemProcessed func(item *StrictItem, err error) // 项目处理完成时调用
	OnPipelineFull  func()                            // 管道满时调用
	OnPipelineEmpty func()                            // 管道空时调用
}

// ItemPipeline 项目管道
type ItemPipeline struct {
	queue      *linkedlistqueue.Queue
	mu         sync.RWMutex
	cond       *sync.Cond
	config     PipelineConfig
	stats      PipelineStats
	callbacks  PipelineCallback
	closed     bool
	processor  func(*StrictItem) error   // 项目处理函数
	processors []func(*StrictItem) error // 多个处理函数
	validators []func(*StrictItem) error // 验证器
	filters    []func(*StrictItem) bool  // 过滤器
	batchQueue *linkedlistqueue.Queue    // 批处理队列
	batchSize  int
}

// NewItemPipeline 创建新的项目管道
func NewItemPipeline(config PipelineConfig) *ItemPipeline {
	if config.MaxSize < 0 {
		config.MaxSize = 0
	}
	if config.AutoFlushSize <= 0 {
		config.AutoFlushSize = 10
	}

	p := &ItemPipeline{
		queue:      linkedlistqueue.New(),
		config:     config,
		callbacks:  PipelineCallback{},
		processor:  nil,
		processors: make([]func(*StrictItem) error, 0),
		validators: make([]func(*StrictItem) error, 0),
		filters:    make([]func(*StrictItem) bool, 0),
		batchQueue: linkedlistqueue.New(),
		batchSize:  0,
	}
	p.cond = sync.NewCond(&p.mu)

	return p
}

// EnqueueItem 入队项目（阻塞式）
func (p *ItemPipeline) EnqueueItem(item *StrictItem) error {
	return p.enqueueItem(item, true)
}

// TryEnqueueItem 尝试入队项目（非阻塞）
func (p *ItemPipeline) TryEnqueueItem(item *StrictItem) error {
	return p.enqueueItem(item, false)
}

// enqueueItem 内部入队实现
func (p *ItemPipeline) enqueueItem(item *StrictItem, blocking bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrPipelineClosed
	}

	// 验证项目
	if item == nil {
		return ErrItemInvalid
	}

	// 检查管道是否已满
	if p.config.MaxSize > 0 && p.queue.Size() >= p.config.MaxSize {
		if !blocking {
			if p.callbacks.OnPipelineFull != nil {
				p.callbacks.OnPipelineFull()
			}
			return ErrPipelineFull
		}

		// 阻塞等待直到有空间
		startTime := time.Now()
		for p.queue.Size() >= p.config.MaxSize {
			if p.config.MaxWaitTime > 0 {
				if time.Since(startTime) > p.config.MaxWaitTime {
					return ErrPipelineFull
				}
			}
			p.cond.Wait()
			if p.closed {
				return ErrPipelineClosed
			}
		}
	}

	// 执行验证
	for _, validator := range p.validators {
		if err := validator(item); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	// 执行过滤
	for _, filter := range p.filters {
		if !filter(item) {
			return nil // 被过滤，不报错
		}
	}

	// 入队
	p.queue.Enqueue(item)
	atomic.AddInt64(&p.stats.TotalEnqueued, 1)
	p.stats.CurrentSize = p.queue.Size()

	// 触发回调
	if p.callbacks.OnItemEnqueued != nil {
		p.callbacks.OnItemEnqueued(item)
	}

	// 通知等待的消费者
	p.cond.Signal()

	// 检查是否需要自动刷新
	if p.config.AutoFlushSize > 0 && p.queue.Size() >= p.config.AutoFlushSize {
		go p.Flush()
	}

	return nil
}

// DequeueItem 出队项目（阻塞式）
func (p *ItemPipeline) DequeueItem() (*StrictItem, error) {
	return p.dequeueItem(true)
}

// TryDequeueItem 尝试出队项目（非阻塞）
func (p *ItemPipeline) TryDequeueItem() (*StrictItem, error) {
	return p.dequeueItem(false)
}

// dequeueItem 内部出队实现
func (p *ItemPipeline) dequeueItem(blocking bool) (*StrictItem, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed && p.queue.Empty() {
		return nil, ErrPipelineClosed
	}

	// 如果队列为空且阻塞模式，等待
	if p.queue.Empty() {
		if !blocking {
			if p.callbacks.OnPipelineEmpty != nil {
				p.callbacks.OnPipelineEmpty()
			}
			return nil, nil
		}

		for p.queue.Empty() && !p.closed {
			p.cond.Wait()
		}

		if p.closed && p.queue.Empty() {
			return nil, ErrPipelineClosed
		}
	}

	value, ok := p.queue.Dequeue()
	if !ok {
		return nil, nil
	}

	item, ok := value.(*StrictItem)
	if !ok {
		return nil, ErrItemInvalid
	}

	atomic.AddInt64(&p.stats.TotalDequeued, 1)
	p.stats.CurrentSize = p.queue.Size()

	// 触发回调
	if p.callbacks.OnItemDequeued != nil {
		p.callbacks.OnItemDequeued(item)
	}

	// 通知等待的生产者
	p.cond.Signal()

	return item, nil
}

// ProcessNext 处理下一个项目
func (p *ItemPipeline) ProcessNext() error {
	item, err := p.DequeueItem()
	if err != nil {
		return err
	}
	if item == nil {
		return nil
	}

	return p.processItem(item)
}

// processItem 处理单个项目
func (p *ItemPipeline) processItem(item *StrictItem) error {
	startTime := time.Now()
	var processErr error

	// 使用单个处理器
	if p.processor != nil {
		processErr = p.processor(item)
	} else if len(p.processors) > 0 {
		// 使用多个处理器链
		for _, processor := range p.processors {
			if err := processor(item); err != nil {
				processErr = err
				break
			}
		}
	}

	// 更新统计
	duration := time.Since(startTime)
	atomic.AddInt64(&p.stats.TotalProcessed, 1)
	if processErr != nil {
		atomic.AddInt64(&p.stats.TotalFailed, 1)
	}

	// 更新平均处理时间（简单移动平均）
	oldAvg := p.stats.AvgProcessTime
	count := p.stats.TotalProcessed
	if count == 1 {
		p.stats.AvgProcessTime = duration
	} else {
		p.stats.AvgProcessTime = (oldAvg*time.Duration(count-1) + duration) / time.Duration(count)
	}
	p.stats.LastProcessTime = time.Now()

	// 触发回调
	if p.callbacks.OnItemProcessed != nil {
		p.callbacks.OnItemProcessed(item, processErr)
	}

	return processErr
}

// Flush 刷新所有项目
func (p *ItemPipeline) Flush() error {
	var lastErr error
	count := 0

	for {
		item, err := p.TryDequeueItem()
		if err != nil {
			lastErr = err
			break
		}
		if item == nil {
			break
		}

		if err := p.processItem(item); err != nil {
			lastErr = err
		}
		count++
	}

	return lastErr
}

// BatchEnqueue 批量入队
func (p *ItemPipeline) BatchEnqueue(items []*StrictItem) []error {
	errors := make([]error, len(items))

	for i, item := range items {
		errors[i] = p.EnqueueItem(item)
	}

	return errors
}

// BatchDequeue 批量出队
func (p *ItemPipeline) BatchDequeue(count int) ([]*StrictItem, error) {
	items := make([]*StrictItem, 0, count)

	for i := 0; i < count; i++ {
		item, err := p.TryDequeueItem()
		if err != nil {
			return items, err
		}
		if item == nil {
			break
		}
		items = append(items, item)
	}

	return items, nil
}

// Close 关闭管道
func (p *ItemPipeline) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	p.cond.Broadcast() // 唤醒所有等待的goroutine

	// 处理剩余项目
	if !p.queue.Empty() {
		go p.Flush()
	}

	return nil
}

// Size 获取当前大小
func (p *ItemPipeline) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.queue.Size()
}

// IsEmpty 检查是否为空
func (p *ItemPipeline) IsEmpty() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.queue.Empty()
}

// IsFull 检查是否已满
func (p *ItemPipeline) IsFull() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config.MaxSize > 0 && p.queue.Size() >= p.config.MaxSize
}

// IsClosed 检查是否已关闭
func (p *ItemPipeline) IsClosed() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.closed
}

// GetStats 获取统计信息
func (p *ItemPipeline) GetStats() PipelineStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := p.stats
	stats.CurrentSize = p.queue.Size()
	return stats
}

// ResetStats 重置统计信息
func (p *ItemPipeline) ResetStats() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stats = PipelineStats{
		CurrentSize:     p.queue.Size(),
		LastProcessTime: p.stats.LastProcessTime,
	}
}

// SetProcessor 设置处理器
func (p *ItemPipeline) SetProcessor(processor func(*StrictItem) error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.processor = processor
}

// AddProcessor 添加处理器到链
func (p *ItemPipeline) AddProcessor(processor func(*StrictItem) error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.processors = append(p.processors, processor)
}

// AddValidator 添加验证器
func (p *ItemPipeline) AddValidator(validator func(*StrictItem) error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.validators = append(p.validators, validator)
}

// AddFilter 添加过滤器
func (p *ItemPipeline) AddFilter(filter func(*StrictItem) bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.filters = append(p.filters, filter)
}

// SetCallbacks 设置回调函数
func (p *ItemPipeline) SetCallbacks(callbacks PipelineCallback) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.callbacks = callbacks
}

// Drain 清空队列（不处理）
func (p *ItemPipeline) Drain() []*StrictItem {
	p.mu.Lock()
	defer p.mu.Unlock()

	items := make([]*StrictItem, 0, p.queue.Size())
	for !p.queue.Empty() {
		value, ok := p.queue.Dequeue()
		if !ok {
			break
		}
		if item, ok := value.(*StrictItem); ok {
			items = append(items, item)
		}
	}

	p.stats.CurrentSize = 0
	p.cond.Broadcast() // 通知可能等待的生产者

	return items
}
