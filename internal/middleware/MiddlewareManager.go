package middleware

import (
	"fmt"
	"github.com/djskncxm/NewDuckSpider/pkg/httpc"
	"sort"
)

// 中间件接口定义
type RequestProcessor interface {
	ProcessRequest(*httpc.Request) error
}

type ResponseProcessor interface {
	ProcessResponse(*httpc.Response) error
}

type ExceptionProcessor interface {
	ProcessException(error) (handled bool, newErr error)
}

type MiddlewarePriority int

const (
	PriorityFirst  MiddlewarePriority = 100
	PriorityHigh   MiddlewarePriority = 50
	PriorityNormal MiddlewarePriority = 0
	PriorityLow    MiddlewarePriority = -50
	PriorityLast   MiddlewarePriority = -100
)

type MiddlewareConfig struct {
	Name     string
	Priority MiddlewarePriority
	Enabled  bool
	Group    string
}

type DecoratedMiddleware struct {
	ID         string
	Processor  interface{}
	Config     MiddlewareConfig
	Middleware interface{} // 原始中间件实例，用于反射等
}

type MiddlewareManager struct {
	requestChain   []DecoratedMiddleware
	responseChain  []DecoratedMiddleware
	exceptionChain []DecoratedMiddleware

	// 中间件查找和禁用功能
	middlewareMap  map[string]DecoratedMiddleware
	disabledGroups map[string]bool
}

func NewMiddlewareManager() *MiddlewareManager {
	return &MiddlewareManager{
		middlewareMap:  make(map[string]DecoratedMiddleware),
		disabledGroups: make(map[string]bool),
	}
}

// 智能注册，自动检测接口类型
func (mm *MiddlewareManager) Register(middleware interface{}, config ...MiddlewareConfig) error {
	cfg := MiddlewareConfig{}
	if len(config) > 0 {
		cfg = config[0]
	}
	if cfg.Name == "" {
		cfg.Name = fmt.Sprintf("%T", middleware)
	}

	id := generateID(cfg.Name)

	dm := DecoratedMiddleware{
		ID:         id,
		Processor:  middleware,
		Config:     cfg,
		Middleware: middleware,
	}

	// 检测实现了哪些接口
	if rp, ok := middleware.(RequestProcessor); ok {
		dm.Processor = rp
		mm.requestChain = append(mm.requestChain, dm)
		sortRequestChain(mm.requestChain)
	}

	if rp, ok := middleware.(ResponseProcessor); ok {
		dm.Processor = rp
		mm.responseChain = append(mm.responseChain, dm)
		sortResponseChain(mm.responseChain)
	}

	if ep, ok := middleware.(ExceptionProcessor); ok {
		dm.Processor = ep
		mm.exceptionChain = append(mm.exceptionChain, dm)
	}

	mm.middlewareMap[id] = dm
	return nil
}

func (mm *MiddlewareManager) ProcessRequest(req *httpc.Request) error {
	for _, dm := range mm.getEnabledMiddlewares(mm.requestChain) {
		if rp, ok := dm.Processor.(RequestProcessor); ok {
			if err := rp.ProcessRequest(req); err != nil {
				return fmt.Errorf("middleware %s failed: %w", dm.Config.Name, err)
			}
		}
	}
	return nil
}

func (mm *MiddlewareManager) ProcessResponse(resp *httpc.Response) error {
	enabled := mm.getEnabledMiddlewares(mm.responseChain)
	for i := len(enabled) - 1; i >= 0; i-- {
		dm := enabled[i]
		if rp, ok := dm.Processor.(ResponseProcessor); ok {
			if err := rp.ProcessResponse(resp); err != nil {
				return fmt.Errorf("middleware %s failed: %w", dm.Config.Name, err)
			}
		}
	}
	return nil
}

func (mm *MiddlewareManager) ProcessException(err error) error {
	for _, dm := range mm.getEnabledMiddlewares(mm.exceptionChain) {
		if ep, ok := dm.Processor.(ExceptionProcessor); ok {
			if handled, newErr := ep.ProcessException(err); handled {
				return newErr
			}
		}
	}
	return err
}

// 辅助方法
func (mm *MiddlewareManager) getEnabledMiddlewares(chain []DecoratedMiddleware) []DecoratedMiddleware {
	result := make([]DecoratedMiddleware, 0, len(chain))
	for _, dm := range chain {
		if dm.Config.Enabled && !mm.disabledGroups[dm.Config.Group] {
			result = append(result, dm)
		}
	}
	return result
}

func (mm *MiddlewareManager) EnableGroup(group string) {
	delete(mm.disabledGroups, group)
}

func (mm *MiddlewareManager) DisableGroup(group string) {
	mm.disabledGroups[group] = true
}

// 排序函数
func sortRequestChain(chain []DecoratedMiddleware) {
	sort.Slice(chain, func(i, j int) bool {
		return chain[i].Config.Priority > chain[j].Config.Priority
	})
}

func sortResponseChain(chain []DecoratedMiddleware) {
	sort.Slice(chain, func(i, j int) bool {
		return chain[i].Config.Priority < chain[j].Config.Priority
	})
}

func generateID(name string) string {
	// 简单的ID生成逻辑
	return fmt.Sprintf("mw-%s", name)
}
