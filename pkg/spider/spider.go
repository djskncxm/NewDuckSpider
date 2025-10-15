package spider

type SpiderIns interface {
	Name() string
}
type Spider struct {
	SpiderName string
}

func (s Spider) Name() string {
	return s.SpiderName
}
