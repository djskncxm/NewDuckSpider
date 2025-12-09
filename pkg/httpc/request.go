package httpc

type Request struct {
	URL      string
	Method   string
	Headers  map[string]string
	Body     []byte
	Meta     map[string]any
	Callback func(*Response)
}

func New(url string) *Request {
	return &Request{
		URL:     url,
		Method:  "GET",
		Headers: make(map[string]string),
		Meta:    make(map[string]any),
	}
}
func (r *Request) WithMethod(method string) *Request {
	r.Method = method
	return r
}

func (r *Request) WithHeader(key, value string) *Request {
	r.Headers[key] = value
	return r
}

func (r *Request) WithMeta(key string, value any) *Request {
	r.Meta[key] = value
	return r
}

func (r *Request) WithBody(body []byte) *Request {
	r.Body = body
	return r
}

func (r *Request) WithCallback(cb func(*Response)) *Request {
	r.Callback = cb
	return r
}
