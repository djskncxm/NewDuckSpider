package core

import (
	"fmt"
	"sync"

	"github.com/djskncxm/NewDuckSpider/internal/download"
	"github.com/djskncxm/NewDuckSpider/internal/setting"
	"github.com/djskncxm/NewDuckSpider/pkg/httpc"
	"github.com/djskncxm/NewDuckSpider/pkg/item"
	"github.com/djskncxm/NewDuckSpider/pkg/logger"
	"github.com/djskncxm/NewDuckSpider/pkg/spider"
)

type Engine struct {
	spider       spider.Spider
	download     download.Download
	scheduler    *Scheduler
	Config       *setting.SettingsManager
	ItemPipeline *item.ItemPipeline
	Logger       *logger.Logger
	mu           sync.Mutex
}

func InitEngine(spider spider.Spider, Config *setting.SettingsManager, LogConfig logger.LogConfig, PipelineConfig item.PipelineConfig) Engine {
	logger, err := logger.NewLogger(&LogConfig)
	if err != nil {
		panic(fmt.Errorf("日志系统初始化错误: %w", err))
	}

	spider.Logger = logger

	return Engine{
		spider:       spider,
		download:     download.InitDownload(),
		scheduler:    NewScheduler(),
		Config:       Config,
		ItemPipeline: item.NewItemPipeline(PipelineConfig),
		Logger:       logger,
	}
}

func (e *Engine) StartSpider() {
	e.Logger.Debug("框架启动")
	var concurrency int = e.Config.GetInt("Spider.Worker", 3)
	e.Logger.Info(concurrency)

	if concurrency == 3 {
	}

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
	e.Logger.Debug("框架关闭")
}

func (e *Engine) worker() {
	for {
		req := e.GetRequest()
		if req == nil {
			if e.isAllWorkDone() {
				return
			}
		}

		resp := e.fetch(req)

		if req.Callback == nil {
			panic("Request without callback")
		}

		ParseResult_ := req.Callback(resp)
		for _, req := range ParseResult_.Requests {
			e.EnRequest(req)
		}

		for _, it := range ParseResult_.Items {
			it.Metadata.SpiderName = e.spider.Name()
			e.EnItem(it)
		}

		if !e.ItemPipeline.IsEmpty() {
			e.ItemPipeline.ProcessNext()
		}
	}
}

func (e *Engine) EnItem(item *item.StrictItem) {
	e.ItemPipeline.EnqueueItem(item)
}

func (e *Engine) fetch(request *httpc.Request) *httpc.Response {
	return e.download.Fetch(request)
}

func (e *Engine) EnRequest(request *httpc.Request) {
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
