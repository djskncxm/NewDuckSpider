package crawler

import (
	"github.com/djskncxm/NewDuckSpider/internal/core"
	"github.com/djskncxm/NewDuckSpider/internal/setting"
	"github.com/djskncxm/NewDuckSpider/pkg/spider"
	"gopkg.in/yaml.v3"
	"os"
)

type Crawler struct {
	Engine  *core.Engine
	Config  *setting.SettingsManager
	spiders []spider.Spider
}

func New() *Crawler {
	configPath := "config/config.yaml"
	data, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}

	var cfg setting.Setting
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		panic(err)
	}

	sm := setting.NewSettingsManager()
	sm.LoadFromSetting(cfg)

	// 初始化 Engine，并传入配置
	engine := core.InitEngine(sm)

	return &Crawler{
		Engine: &engine,
		Config: sm,
	}
}

func (c *Crawler) StartRequest() {
	c.Engine.StartSpider()
}

func (c *Crawler) RegisterSpider(spiders ...spider.Spider) {
	for _, sp := range spiders {
		c.Engine.AddSpider(sp)
	}
}
