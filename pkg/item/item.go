package item

import "time"

type Item struct {
	Data    map[string]interface{}
	allowed map[string]struct{}

	SpiderName string    // 哪个爬虫生成的
	URL        string    // 抓取来源 URL
	FetchTime  time.Time // 抓取时间
}
