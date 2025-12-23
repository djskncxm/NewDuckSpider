package httpc

type Response struct {
	URL      string
	Status   int
	Headers  map[string]string
	Body     []byte
	Request  Request
	Protocol string
}

func NewResponse(URL string,
	Status int,
	Headers map[string]string,
	Body []byte,
	Request Request,
	ProTocol string,
) *Response {
	return &Response{
		URL:      URL,
		Status:   Status,
		Headers:  Headers,
		Body:     Body,
		Request:  Request,
		Protocol: ProTocol,
	}
}
