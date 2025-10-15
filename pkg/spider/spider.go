package spider

type SpiderIns interface {
	Name() string
}
type Spider struct {
	name string
}

func (s Spider) Name() string {
	return s.name
}
