package spider

import "github.com/djskncxm/NewDuckSpider/pkg/httpc"

type SpiderIns interface {
	Name() string
	Start() []*httpc.Request
	Handle(httpc.Response) []*httpc.Request
}

type Spider struct {
	SpiderName string
	URL        string
	URLs       []string
	Callback   func(*httpc.Response) []*httpc.Request
}

func (s Spider) Name() string {
	return s.SpiderName
}

func (s Spider) Start() []*httpc.Request {
	res := make([]*httpc.Request, 0)

	if s.URL != "" {
		res = append(res, httpc.New(s.URL))
	}

	if len(s.URLs) > 0 {
		for _, u := range s.URLs {
			res = append(res, httpc.New(u))
		}
	}

	return res
}

func (s Spider) Handle(response *httpc.Response) []*httpc.Request {

	if s.Callback != nil {
		return s.Callback(response)
	}
	return nil
}
