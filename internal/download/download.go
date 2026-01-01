package download

import (
	"fmt"
	"github.com/djskncxm/NewDuckSpider/pkg/httpc"
	"github.com/djskncxm/NewDuckSpider/pkg/logger"
	"github.com/djskncxm/NewDuckSpider/pkg/middleware"
	"github.com/emirpasic/gods/sets/treeset"
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

func InitDownload(Loggger *logger.Logger, MiddlewareManager *middleware.MiddlewareManager) Download {
	return Download{
		Logger:            Loggger,
		MiddlewareManager: MiddlewareManager,
	}
}

func (d *Download) Fetch(request *httpc.Request) *httpc.Response {
	d.MiddlewareManager.ProcessRequest(request)
	
	// 发送请求
	resp, err := http.Get(request.URL)
	if err != nil {
		fmt.Println("请求失败:", err)
		return nil
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取响应失败:", err)
		return nil
	}

	// 将 http.Header 转换为 map[string]string
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// 创建并返回 httpc.Response
	return httpc.NewResponse(
		resp.Request.URL.String(),  // 使用实际请求的URL（可能会有重定向）
		resp.StatusCode,
		headers,
		body,
		request,                    // 原始的请求对象
		resp.Proto,                 // HTTP协议版本
	)
}
