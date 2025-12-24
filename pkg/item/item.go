package item

import (
	"fmt"
	"sync"
	"time"
)

type StrictItem struct {
	Data    map[string]interface{}
	allowed map[string]struct{}
	mu      sync.RWMutex

	SpiderName string    // 哪个爬虫生成的
	URL        string    // 抓取来源 URL
	FetchTime  time.Time // 抓取时间
}

func NewStrictItem(allowedFields []string) *StrictItem {
	a := make(map[string]struct{})
	for _, field := range allowedFields {
		a[field] = struct{}{}
	}
	return &StrictItem{
		Data:    make(map[string]interface{}),
		allowed: a,
	}
}

func (s *StrictItem) Set(key string, value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.allowed[key]; !ok {
		return fmt.Errorf("字段 '%s' 不在预定义字段中", key)
	}
	s.Data[key] = value
	return nil
}

func (s *StrictItem) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.Data[key]
	return val, ok
}

func (s *StrictItem) All() map[string]interface{} {
	return s.Data
}
