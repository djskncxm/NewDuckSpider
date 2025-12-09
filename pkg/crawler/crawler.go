package crawler

import "github.com/djskncxm/NewDuckSpider/internal/core"
import "github.com/djskncxm/NewDuckSpider/pkg/spider"

type crawler struct {
	Engine core.Engine
}

func InitCrawle(s spider.Spider, cap int) crawler {
	return crawler{
		Engine: core.InitEngine(s, cap),
	}
}

func (c *crawler) StartRequest() {
	c.Engine.StartSpider()
}
