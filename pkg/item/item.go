package item

import (
	"fmt"
	"sync"
	"time"
)

// StrictItem 定义了严格的项结构，只允许预定义的字段
type StrictItem struct {
	data     map[string]interface{} // 私有化字段，防止外部直接修改
	allowed  map[string]struct{}    // 允许的字段集合
	mu       sync.RWMutex
	Metadata metadata // 元数据单独分组
}

// metadata 包含项的元数据信息
type metadata struct {
	SpiderName string    // 哪个爬虫生成的
	FetchTime  time.Time // 抓取时间
}

// NewStrictItem 创建新的严格项
func NewStrictItem(allowedFields []string) *StrictItem {
	allowed := make(map[string]struct{})
	for _, field := range allowedFields {
		allowed[field] = struct{}{}
	}
	return &StrictItem{
		data:    make(map[string]interface{}),
		allowed: allowed,
		Metadata: metadata{
			FetchTime: time.Now(),
		},
	}
}

// Set 设置字段值，会验证字段是否允许
func (s *StrictItem) Set(key string, value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.allowed[key]; !ok {
		return fmt.Errorf("字段 '%s' 不在预定义字段中，允许的字段: %v",
			key, s.GetAllowedFields())
	}
	s.data[key] = value
	return nil
}

// Get 获取字段值
func (s *StrictItem) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

// GetString 获取字符串类型的字段值
func (s *StrictItem) GetString(key string) (string, bool) {
	val, ok := s.Get(key)
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// GetInt 获取整数类型的字段值
func (s *StrictItem) GetInt(key string) (int, bool) {
	val, ok := s.Get(key)
	if !ok {
		return 0, false
	}
	switch v := val.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// GetAll 返回数据的深拷贝，防止外部修改内部数据
func (s *StrictItem) GetAll() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]interface{}, len(s.data))
	for k, v := range s.data {
		result[k] = v
	}
	return result
}

// IsAllowed 检查字段是否允许
func (s *StrictItem) IsAllowed(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.allowed[key]
	return ok
}

// GetAllowedFields 返回所有允许的字段
func (s *StrictItem) GetAllowedFields() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fields := make([]string, 0, len(s.allowed))
	for field := range s.allowed {
		fields = append(fields, field)
	}
	return fields
}

// Size 返回已设置的字段数量
func (s *StrictItem) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

// Has 检查字段是否已设置
func (s *StrictItem) Has(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.data[key]
	return ok
}

// Delete 删除字段（需要验证是否是允许的字段）
func (s *StrictItem) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.allowed[key]; !ok {
		return fmt.Errorf("字段 '%s' 不是预定义字段，不能删除", key)
	}
	delete(s.data, key)
	return nil
}

// Clear 清空所有数据字段（保留元数据）
func (s *StrictItem) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string]interface{})
}

// 元数据相关方法
func (s *StrictItem) SetSpiderName(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Metadata.SpiderName = name
}

func (s *StrictItem) GetSpiderName() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Metadata.SpiderName
}

func (s *StrictItem) GetFetchTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Metadata.FetchTime
}

// Marshal 将数据序列化为 map（包含元数据）
func (s *StrictItem) Marshal() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]interface{}, len(s.data)+3)
	// 复制数据字段
	for k, v := range s.data {
		result[k] = v
	}
	// 添加元数据
	result["_spider"] = s.Metadata.SpiderName
	result["_fetch_time"] = s.Metadata.FetchTime

	return result
}

// Validate 验证所有必需字段是否已设置
func (s *StrictItem) Validate(requiredFields []string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	missing := make([]string, 0)
	for _, field := range requiredFields {
		if _, ok := s.data[field]; !ok {
			missing = append(missing, field)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("缺少必需字段: %v", missing)
	}
	return nil
}
