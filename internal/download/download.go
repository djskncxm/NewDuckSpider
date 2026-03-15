package download

import (
	// "fmt"
	"github.com/djskncxm/NewDuckSpider/pkg/httpc"
	"github.com/djskncxm/NewDuckSpider/pkg/logger"
	"github.com/djskncxm/NewDuckSpider/pkg/middleware"
	"github.com/emirpasic/gods/sets/treeset"
	"time"
	// "github.com/enetx/surf"
	"io"
	"net/http"
	"sync"
)

type Download struct {
	activeQueue       *treeset.Set
	MiddlewareManager *middleware.MiddlewareManager
	Logger            *logger.Logger
	mu                sync.Mutex
}

var client = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
	},
	Timeout: 15 * time.Second,
}

func InitDownload(Loggger *logger.Logger, MiddlewareManager *middleware.MiddlewareManager) Download {
	return Download{
		Logger:            Loggger,
		MiddlewareManager: MiddlewareManager,
	}
}

func (d *Download) Fetch(request *httpc.Request) *httpc.Response {
	d.MiddlewareManager.ProcessRequest(request)
	// surfClient := surf.NewClient().Builder().Impersonate().Linux().Chrome().Session().Build().Unwrap()
	// stdClient := surfClient.Std()
	// resp, err := stdClient.Get(request.URL)
	// resp, err := client.Get(request.URL)

	req, err := http.NewRequest(request.Method, request.URL, nil)
	if err != nil {
		panic(err)
	}
	for k, v := range request.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		d.Logger.Stats.AddInt("Request 请求失败", 1)
		d.MiddlewareManager.ProcessException(err)
		return nil
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		d.Logger.Stats.AddInt("Request Body 解析失败", 1)
		d.MiddlewareManager.ProcessException(err)
		return nil
	}

	// 将 http.Header 转换为 map[string]string
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	d.Logger.Stats.AddInt("Request 下载完成", 1)
	response := httpc.NewResponse(
		resp.Request.URL.String(), // 使用实际请求的URL（可能会有重定向）
		resp.StatusCode,
		headers,
		body,
		request,    // 原始的请求对象
		resp.Proto, // HTTP协议版本
	)
	d.MiddlewareManager.ProcessResponse(response)
	return response
}
