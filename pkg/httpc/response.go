package httpc

type Response struct {
	URL      string
	Status   int
	Headers  map[string]string
	Body     []byte
	Request  Request
	ProTocol string
}

func New(URL string,
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
		ProTocol: ProTocol,
	}
}
