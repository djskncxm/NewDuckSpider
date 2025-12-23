package core

import (
	"sync"

	"github.com/djskncxm/NewDuckSpider/internal/download"
	"github.com/djskncxm/NewDuckSpider/internal/setting"
	"github.com/djskncxm/NewDuckSpider/pkg/httpc"
	"github.com/djskncxm/NewDuckSpider/pkg/item"
	"github.com/djskncxm/NewDuckSpider/pkg/spider"
)

type Engine struct {
	spider    []spider.Spider
	download  download.Download
	scheduler *Scheduler
	Config    *setting.SettingsManager
}

func InitEngine(Config *setting.SettingsManager) Engine {
	return Engine{
		download:  download.InitDownload(),
		scheduler: NewScheduler(),
		Config:    Config,
	}
}

func (e *Engine) StartSpider() {
	var concurrency int = e.Config.GetInt("Spider.Worker", 3)

	if concurrency == 3 {
	}

	for _, sp := range e.spider {
		for _, req := range sp.Start() {
			e.EnRequest(req)
		}
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
}

func (e *Engine) worker() {
	for {
		req := e.GetRequest()
		if req == nil && e.scheduler.Empty() {
			return
		}

		resp := e.fetch(req)

		var callback func(selection *httpc.Response) *httpc.ParseResult
		if req.Callback == nil {
			panic("Request without callback")
		}
		callback = req.Callback

		ParseResult_ := callback(resp)
		for _, req := range ParseResult_.Requests {
			e.EnRequest(req)
		}

		for _, it := range ParseResult_.Items {
			e.EnItem(it)
		}
	}
}

func (e *Engine) EnItem(it item.Item) {
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

func (e *Engine) AddSpider(s spider.Spider) {
	e.spider = append(e.spider, s)
}
