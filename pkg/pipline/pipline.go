package pipline

import (
	"github.com/djskncxm/NewDuckSpider/pkg/item"
	"sync"
)

type ItemPipeline interface {
	Process(item *item.StrictItem) error
	Start()
	Close()
}

type Pipline struct {
	mu    sync.Mutex
	state bool
}

func (p *Pipline) Process(item *item.StrictItem) error {
	return nil
}

func (p *Pipline) Start() {
	p.state = true
}

func (p *Pipline) Close() {
	p.state = false
}

func (p *Pipline) SendUser() {
	p.Start()
	p.Process()
	p.Close()
}
