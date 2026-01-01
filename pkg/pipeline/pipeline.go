package pipeline

import (
	"github.com/djskncxm/NewDuckSpider/pkg/item"
	"github.com/djskncxm/NewDuckSpider/pkg/spider"
	"sync"
)

type BasePipeline interface {
	ProcessItem(item *item.StrictItem, spider spider.Spider) error
	Start()
	Close()
}

type Pipline struct {
	mu    sync.Mutex
	state bool
}
