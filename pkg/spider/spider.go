package spider

import (
	"github.com/djskncxm/NewDuckSpider/pkg/httpc"
	"github.com/djskncxm/NewDuckSpider/pkg/logger"
)

type SpiderIns interface {
	Name() string
	Start() []*httpc.Request
}

type Spider struct {
	SpiderName string
	URL        string
	URLs       []string
	Callback   func(*httpc.Response) *httpc.ParseResult // Request | Iiem 使用chan进行动态流处理item
	Logger     *logger.Logger
}

func (s Spider) Name() string {
	return s.SpiderName
}

func (s Spider) Start() []*httpc.Request {
	res := make([]*httpc.Request, 0)

	if s.URL != "" {
		// 初始请求绑定 Spider.Callback
		res = append(res, httpc.New(s.URL).WithCallback(s.Callback))
	}

	for _, u := range s.URLs {
		res = append(res, httpc.New(u).WithCallback(s.Callback))
	}

	return res
}
