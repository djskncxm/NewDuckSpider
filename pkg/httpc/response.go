package httpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"io"
	
	"github.com/andybalholm/cascadia"
	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

type Response struct {
	URL      string
	Status   int
	Headers  map[string]string
	Body     []byte
	Request  Request
	Protocol string
	
	// 内部状态
	rootNode *html.Node
	curNodes []*html.Node
	err      error
	text     *string
}

func NewResponse(
	URL string,
	Status int,
	Headers map[string]string,
	Body []byte,
	Request Request,
	Protocol string,
) *Response {
	return &Response{
		URL:      URL,
		Status:   Status,
		Headers:  Headers,
		Body:     Body,
		Request:  Request,
		Protocol: Protocol,
	}
}

// ParseHTML 解析HTML，返回链式调用的开始
func (r *Response) ParseHTML() *Response {
	if r.rootNode == nil && r.err == nil {
		root, err := htmlquery.Parse(bytes.NewReader(r.Body))
		if err != nil {
			r.err = err
			return r
		}
		r.rootNode = root
		r.curNodes = []*html.Node{root}
	}
	return r
}

// XPath 使用XPath表达式查询
func (r *Response) XPath(expr string) *Response {
	if r.err != nil {
		return r
	}
	
	r.ParseHTML() // 确保已解析HTML
	
	if r.curNodes == nil {
		r.curNodes = []*html.Node{r.rootNode}
	}
	
	var results []*html.Node
	for _, node := range r.curNodes {
		found := htmlquery.Find(node, expr)
		results = append(results, found...)
	}
	r.curNodes = results
	return r
}

// CSS 使用CSS选择器查询
func (r *Response) CSS(selector string) *Response {
	if r.err != nil {
		return r
	}
	
	r.ParseHTML() // 确保已解析HTML
	
	sel, err := cascadia.Compile(selector)
	if err != nil {
		r.err = fmt.Errorf("invalid CSS selector: %v", err)
		return r
	}
	
	if r.curNodes == nil {
		r.curNodes = []*html.Node{r.rootNode}
	}
	
	var results []*html.Node
	for _, node := range r.curNodes {
		found := cascadia.QueryAll(node, sel)
		results = append(results, found...)
	}
	r.curNodes = results
	return r
}

// First 获取第一个匹配的元素
func (r *Response) First() *Response {
	if r.err != nil || len(r.curNodes) == 0 {
		return r
	}
	r.curNodes = []*html.Node{r.curNodes[0]}
	return r
}

// Last 获取最后一个匹配的元素
func (r *Response) Last() *Response {
	if r.err != nil || len(r.curNodes) == 0 {
		return r
	}
	r.curNodes = []*html.Node{r.curNodes[len(r.curNodes)-1]}
	return r
}

// Eq 获取指定位置的元素（从0开始）

// Filter 过滤元素
func (r *Response) Filter(fn func(*html.Node) bool) *Response {
	if r.err != nil {
		return r
	}
	
	var filtered []*html.Node
	for _, node := range r.curNodes {
		if fn(node) {
			filtered = append(filtered, node)
		}
	}
	r.curNodes = filtered
	return r
}

// Attr 获取属性值（第一个匹配元素的）
func (r *Response) Attr(name string) string {
	if r.err != nil || len(r.curNodes) == 0 {
		return ""
	}
	
	for _, attr := range r.curNodes[0].Attr {
		if attr.Key == name {
			return attr.Val
		}
	}
	return ""
}

// Text 获取文本内容
func (r *Response) Text() (string, error) {
	if r.err != nil {
		return "", r.err
	}
	
	if len(r.curNodes) == 0 {
		return "", nil
	}
	
	var result string
	if len(r.curNodes) == 1 {
		result = htmlquery.InnerText(r.curNodes[0])
	} else {
		var texts []string
		for _, node := range r.curNodes {
			texts = append(texts, htmlquery.InnerText(node))
		}
		result = strings.Join(texts, "\n")
	}
	
	return strings.TrimSpace(result), nil
}

// MustText 获取文本内容（忽略错误）
func (r *Response) MustText() string {
	text, _ := r.Text()
	return text
}

// HTML 获取HTML内容
func (r *Response) HTML() (string, error) {
	if r.err != nil {
		return "", r.err
	}
	
	if len(r.curNodes) == 0 {
		return "", nil
	}
	
	var buf bytes.Buffer
	if len(r.curNodes) == 1 {
		if err := html.Render(&buf, r.curNodes[0]); err != nil {
			return "", err
		}
	} else {
		for _, node := range r.curNodes {
			if err := html.Render(&buf, node); err != nil {
				return "", err
			}
			buf.WriteString("\n")
		}
	}
	return buf.String(), nil
}

// MustHTML 获取HTML内容（忽略错误）
func (r *Response) MustHTML() string {
	html, _ := r.HTML()
	return html
}

// JSON 解析JSON
func (r *Response) JSON(v interface{}) error {
	if r.err != nil {
		return r.err
	}
	return json.Unmarshal(r.Body, v)
}

// Bytes 获取原始字节
func (r *Response) Bytes() []byte {
	return r.Body
}

// String 获取响应体字符串
func (r *Response) String() string {
	// 尝试自动检测字符集
	contentType, ok := r.Headers["Content-Type"]
	if ok && strings.Contains(strings.ToLower(contentType), "charset=") {
		reader, err := charset.NewReader(bytes.NewReader(r.Body), contentType)
		if err == nil {
			if converted, err := io.ReadAll(reader); err == nil {
				return string(converted)
			}
		}
	}
	return string(r.Body)
}

// Regex 使用正则表达式匹配
func (r *Response) Regex(pattern string) ([]string, error) {
	if r.err != nil {
		return nil, r.err
	}
	
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	
	text := r.String()
	return re.FindAllString(text, -1), nil
}

// MustRegex 使用正则表达式匹配（忽略错误）
func (r *Response) MustRegex(pattern string) []string {
	matches, _ := r.Regex(pattern)
	return matches
}

// Find 查找元素（使用CSS选择器）
func (r *Response) Find(selector string) *Response {
	return r.CSS(selector)
}

// Children 获取子元素
func (r *Response) Children() *Response {
	if r.err != nil || len(r.curNodes) == 0 {
		return r
	}
	
	var children []*html.Node
	for _, node := range r.curNodes {
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if child.Type == html.ElementNode {
				children = append(children, child)
			}
		}
	}
	r.curNodes = children
	return r
}

// Parent 获取父元素
func (r *Response) Parent() *Response {
	if r.err != nil || len(r.curNodes) == 0 {
		return r
	}
	
	var parents []*html.Node
	seen := make(map[*html.Node]bool)
	for _, node := range r.curNodes {
		if node.Parent != nil && node.Parent.Type == html.ElementNode {
			if !seen[node.Parent] {
				seen[node.Parent] = true
				parents = append(parents, node.Parent)
			}
		}
	}
	r.curNodes = parents
	return r
}

// Next 获取下一个兄弟元素
func (r *Response) Next() *Response {
	if r.err != nil || len(r.curNodes) == 0 {
		return r
	}
	
	var nexts []*html.Node
	for _, node := range r.curNodes {
		for n := node.NextSibling; n != nil; n = n.NextSibling {
			if n.Type == html.ElementNode {
				nexts = append(nexts, n)
				break
			}
		}
	}
	r.curNodes = nexts
	return r
}

// Prev 获取上一个兄弟元素
func (r *Response) Prev() *Response {
	if r.err != nil || len(r.curNodes) == 0 {
		return r
	}
	
	var prevs []*html.Node
	for _, node := range r.curNodes {
		for n := node.PrevSibling; n != nil; n = n.PrevSibling {
			if n.Type == html.ElementNode {
				prevs = append(prevs, n)
				break
			}
		}
	}
	r.curNodes = prevs
	return r
}

// Each 遍历所有匹配元素
func (r *Response) Each(fn func(int, *html.Node)) *Response {
	if r.err != nil {
		return r
	}
	
	for i, node := range r.curNodes {
		fn(i, node)
	}
	return r
}

// Map 将元素映射为其他值
func (r *Response) Map(fn func(*html.Node) string) []string {
	if r.err != nil || len(r.curNodes) == 0 {
		return nil
	}
	
	result := make([]string, len(r.curNodes))
	for i, node := range r.curNodes {
		result[i] = fn(node)
	}
	return result
}

// Length 获取匹配元素的数量
func (r *Response) Length() int {
	if r.err != nil {
		return 0
	}
	return len(r.curNodes)
}

// Error 获取错误
func (r *Response) Error() error {
	return r.err
}

// Reset 重置选择器状态
func (r *Response) Reset() *Response {
	r.curNodes = nil
	r.err = nil
	return r
}

// Clone 克隆响应（用于链式调用的分支）
func (r *Response) Clone() *Response {
	return &Response{
		URL:      r.URL,
		Status:   r.Status,
		Headers:  r.Headers,
		Body:     r.Body,
		Request:  r.Request,
		Protocol: r.Protocol,
		rootNode: r.rootNode,
		curNodes: r.curNodes,
		err:      r.err,
		text:     r.text,
	}
}
