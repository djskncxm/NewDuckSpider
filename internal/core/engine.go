package core

import (
	"github.com/djskncxm/NewDuckSpider/internal/download"
	"github.com/djskncxm/NewDuckSpider/pkg/httpc"
	"github.com/djskncxm/NewDuckSpider/pkg/spider"
	"sync"
)

type Engine struct {
	spider    spider.Spider
	download  download.Download
	scheduler *Scheduler
}

func InitEngine(s spider.Spider, cap int) Engine {
	return Engine{
		spider:    s,
		download:  download.InitDownload(),
		scheduler: NewScheduler(cap),
	}
}

func (e *Engine) StartSpider() {
	var concurrency int = 10 // 等待添加配置文件
	// 入队种子请求
	for _, req := range e.spider.Start() {
		e.scheduler.EnRequest(req)
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
		req := e.scheduler.NextRequest()
		if req == nil {
			return
		}

		// 下载
		resp := e.download.Fetch(req)

		// Spider 回调处理，返回新的请求
		newReqs := e.spider.Handle(resp)
		for _, r := range newReqs {
			e.scheduler.EnRequest(r)
		}
	}
}

func (e *Engine) fetch(request *httpc.Request) *httpc.Response {
	return e.download.Fetch(request)
}

func (e *Engine) EnRequest(request *httpc.Request) {
	e.scheduler.EnRequest(request)
}

func (e *Engine) GetRequest() *httpc.Request {
	req, ok := e.scheduler.NextRequest()
	if ok {
		return req
	}
	return nil
}
