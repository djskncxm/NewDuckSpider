package crawler

import (
	"fmt"
	"sync"

	"github.com/djskncxm/NewDuckSpider/internal/core"
	"github.com/djskncxm/NewDuckSpider/internal/setting"
	"github.com/djskncxm/NewDuckSpider/pkg/logger"
	"github.com/djskncxm/NewDuckSpider/pkg/spider"
	"github.com/emirpasic/gods/sets/treeset"
	"gopkg.in/yaml.v3"
	"os"
)

type CrawlerManager struct {
	config   *setting.SettingsManager // 统一配置
	crawlers map[string]*Crawler      // 爬虫实例映射
	nameSet  *treeset.Set             // 爬虫名称集合（有序）
}

// Crawler 单个爬虫执行器
type Crawler struct {
	spider spider.Spider            // 爬虫逻辑
	engine *core.Engine             // 执行引擎
	config *setting.SettingsManager // 配置引用
}

// NewCrawlerManager 创建爬虫管理器（入口点）
func NewCrawlerManager() (*CrawlerManager, error) {
	// 统一加载配置（仅此一次）
	config, err := loadConfig("config/config.yaml")
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	return &CrawlerManager{
		config:   config,
		crawlers: make(map[string]*Crawler),
		nameSet:  treeset.NewWithStringComparator(),
	}, nil
}

// loadConfig 集中加载配置（私有方法）
// loadConfig 加载配置，返回错误而不是panic
func loadConfig(configPath string) (*setting.SettingsManager, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg setting.Setting
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	sm := setting.NewSettingsManager()
	sm.LoadFromSetting(cfg)
	return sm, nil
}

// RegisterSpider 注册爬虫
func (cm *CrawlerManager) RegisterSpider(sp spider.Spider) {
	name := sp.Name()

	// 检查是否已存在
	if cm.nameSet.Contains(name) {
		fmt.Printf("[警告] 爬虫 %s 已存在，跳过注册\n", name)
		return
	}

	// 创建爬虫实例（传入统一配置）
	crawler := &Crawler{
		spider: sp,
		config: cm.config,
	}

	// 注册
	cm.nameSet.Add(name)
	cm.crawlers[name] = crawler

	loglevel, ok := cm.config.GetString("Spider.LOGLEVEL")
	if !ok {
		loglevel = "debug"
	}
	LogFormat, ok := cm.config.GetString("Log.LogFormat")
	if !ok {
		LogFormat = "text"
	}
	EnableConsole := cm.config.GetBool("Log.EnableConsole", true)
	ConsoleColor := cm.config.GetBool("Log.ConsoleColor", true)
	EnableFile := cm.config.GetBool("Log.EnableFile", true)

	config := logger.LogConfig{
		AppName:       sp.Name(),
		LogLevel:      loglevel,
		LogFormat:     LogFormat,
		EnableConsole: EnableConsole,
		ConsoleColor:  ConsoleColor,
		EnableFile:    EnableFile,
		EnableStats:   true,
		MaxSize:       100, // MB
		MaxBackups:    10,
		MaxAge:        30, // days
		Compress:      true,
	}
	fmt.Println(config)
	engine := core.InitEngine(sp, cm.config, config)
	cm.crawlers[name].engine = &engine
}

// StartAll 启动所有爬虫（并发执行）
func (cm *CrawlerManager) StartAll() {
	var wg sync.WaitGroup

	iterator := cm.nameSet.Iterator()
	for iterator.Next() {
		name, ok := iterator.Value().(string)
		if !ok {
			continue
		}

		crawler := cm.crawlers[name]
		if crawler == nil {
			continue
		}

		wg.Add(1)
		go func(c *Crawler, n string) {
			defer wg.Done()
			fmt.Printf("启动爬虫: %s\n", n)
			c.engine.StartSpider()
			c.engine.Logger.PrintStats()
		}(crawler, name)
	}
	wg.Wait()
}

// GetConfig 获取配置管理器（如果需要外部访问配置）
func (cm *CrawlerManager) GetConfig() *setting.SettingsManager {
	return cm.config
}
