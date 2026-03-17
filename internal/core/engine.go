package core

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/djskncxm/NewDuckSpider/internal/download"
	"github.com/djskncxm/NewDuckSpider/internal/setting"
	"github.com/djskncxm/NewDuckSpider/pkg/httpc"
	"github.com/djskncxm/NewDuckSpider/pkg/item"
	"github.com/djskncxm/NewDuckSpider/pkg/logger"
	"github.com/djskncxm/NewDuckSpider/pkg/middleware"
	"github.com/djskncxm/NewDuckSpider/pkg/spider"
)

type Engine struct {
	spider            spider.Spider
	download          download.Download
	scheduler         *Scheduler
	Config            *setting.SettingsManager
	ItemPipeline      *item.ItemPipeline
	Logger            *logger.Logger
	MiddlewareManager *middleware.MiddlewareManager
	mu                sync.Mutex
}

func InitEngine(spider spider.Spider, Config *setting.SettingsManager, LogConfig logger.LogConfig, PipelineConfig item.PipelineConfig) Engine {
	logger, err := logger.NewLogger(&LogConfig)
	if err != nil {
		panic(fmt.Errorf("日志系统初始化错误: %w", err))
	}

	mi := middleware.NewMiddlewareManager()

	spider.Logger = logger

	return Engine{
		spider:            spider,
		download:          download.InitDownload(logger, mi),
		scheduler:         NewScheduler(1000),
		Config:            Config,
		ItemPipeline:      item.NewItemPipeline(PipelineConfig, logger),
		Logger:            logger,
		MiddlewareManager: mi,
	}
}

func (e *Engine) StartSpider() {
	e.Logger.Debug("框架启动")
	var concurrency int = e.Config.GetInt("Spider.Worker", 3)
	e.Logger.Debug("并发数 -> " + strconv.Itoa(concurrency))

	if concurrency == 3 {
	}

	go e.ItemPipeline.ProcessNext()

	for _, req := range e.spider.Start() {
		e.EnRequest(req)
	}

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.worker()
		}()
	}
	wg.Wait()
	e.ItemPipeline.Close()
	e.scheduler.CloseScheduler()
	e.Logger.Debug("框架关闭")
}
func (e *Engine) worker() {
	for {
		req, ok := e.scheduler.NextRequestBlocking()
		if !ok {
			// 这个地方是永远走不到了，TM得我忘记了channel是阻塞的
			if e.isAllWorkDone() {
				return
			}
			continue
		}

		e.Logger.Stats.AddInt("Request 出队", 1)

		resp := e.fetch(req)

		if req.Callback != nil {
			parseResult := req.Callback(resp)
			if parseResult != nil {
				// 新请求入队
				for _, newReq := range parseResult.Requests {
					e.EnRequest(newReq)
				}
				// Items 入队
				for _, it := range parseResult.Items {
					it.Metadata.SpiderName = e.spider.Name()
					e.EnItem(it)
				}
			}
		}
	}
}

func (e *Engine) EnItem(item *item.StrictItem) {
	e.Logger.Stats.AddInt("Item 入队", 1)
	e.ItemPipeline.EnqueueItem(item)
}

func (e *Engine) fetch(request *httpc.Request) *httpc.Response {
	return e.download.Fetch(request)
}

func (e *Engine) EnRequest(request *httpc.Request) {
	e.Logger.Stats.AddInt("Request 入队", 1)
	e.scheduler.EnqueueRequest(request)
}

func (e *Engine) GetRequest() *httpc.Request {
	req := e.scheduler.NextRequest()
	if req != nil {
		return req
	}
	return nil
}

func (e *Engine) isAllWorkDone() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.scheduler.Empty() && e.ItemPipeline.IsEmpty()
}
