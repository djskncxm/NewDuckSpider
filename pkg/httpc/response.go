// package httpc
//
// import (
//
//	"bytes"
//	"encoding/json"
//	"fmt"
//	"io"
//	"regexp"
//	"strings"
//
//	"github.com/andybalholm/cascadia"
//	"github.com/antchfx/htmlquery"
//	"golang.org/x/net/html"
//	"golang.org/x/net/html/charset"
//
// )
//
//	type Response struct {
//		URL        string
//		StatusCode int
//		Headers    map[string]string
//		Body       []byte
//		Request    Request
//		Protocol   string
//
//		// 内部状态
//		rootNode *html.Node
//		curNodes []*html.Node
//		err      error
//		text     *string
//	}
//
// func NewResponse(
//
//	URL string,
//	Status int,
//	Headers map[string]string,
//	Body []byte,
//	Request *Request,
//	Protocol string,
//
//	) *Response {
//		return &Response{
//			URL:        URL,
//			StatusCode: Status,
//			Headers:    Headers,
//			Body:       Body,
//			Request:    *Request,
//			Protocol:   Protocol,
//		}
//	}
//
// // ParseHTML 解析HTML，返回链式调用的开始
//
//	func (r *Response) ParseHTML() *Response {
//		if r.rootNode == nil && r.err == nil {
//			root, err := htmlquery.Parse(bytes.NewReader(r.Body))
//			if err != nil {
//				r.err = err
//				return r
//			}
//			r.rootNode = root
//			r.curNodes = []*html.Node{root}
//		}
//		return r
//	}
//
// // XPath 使用XPath表达式查询
//
//	func (r *Response) XPath(expr string) *Response {
//		if r.err != nil {
//			return r
//		}
//
//		r.ParseHTML() // 确保已解析HTML
//
//		if r.curNodes == nil {
//			r.curNodes = []*html.Node{r.rootNode}
//		}
//
//		var results []*html.Node
//		for _, node := range r.curNodes {
//			found := htmlquery.Find(node, expr)
//			results = append(results, found...)
//		}
//		r.curNodes = results
//		return r
//	}
//
// // CSS 使用CSS选择器查询
//
//	func (r *Response) CSS(selector string) *Response {
//		if r.err != nil {
//			return r
//		}
//
//		r.ParseHTML() // 确保已解析HTML
//
//		sel, err := cascadia.Compile(selector)
//		if err != nil {
//			r.err = fmt.Errorf("invalid CSS selector: %v", err)
//			return r
//		}
//
//		if r.curNodes == nil {
//			r.curNodes = []*html.Node{r.rootNode}
//		}
//
//		var results []*html.Node
//		for _, node := range r.curNodes {
//			found := cascadia.QueryAll(node, sel)
//			results = append(results, found...)
//		}
//		r.curNodes = results
//		return r
//	}
//
// // First 获取第一个匹配的元素
//
//	func (r *Response) First() *Response {
//		if r.err != nil || len(r.curNodes) == 0 {
//			return r
//		}
//		r.curNodes = []*html.Node{r.curNodes[0]}
//		return r
//	}
//
// // Last 获取最后一个匹配的元素
//
//	func (r *Response) Last() *Response {
//		if r.err != nil || len(r.curNodes) == 0 {
//			return r
//		}
//		r.curNodes = []*html.Node{r.curNodes[len(r.curNodes)-1]}
//		return r
//	}
//
// // Eq 获取指定位置的元素（从0开始）
//
// // Filter 过滤元素
//
//	func (r *Response) Filter(fn func(*html.Node) bool) *Response {
//		if r.err != nil {
//			return r
//		}
//
//		var filtered []*html.Node
//		for _, node := range r.curNodes {
//			if fn(node) {
//				filtered = append(filtered, node)
//			}
//		}
//		r.curNodes = filtered
//		return r
//	}
//
// // Attr 获取属性值（第一个匹配元素的）
//
//	func (r *Response) Attr(name string) string {
//		if r.err != nil || len(r.curNodes) == 0 {
//			return ""
//		}
//
//		for _, attr := range r.curNodes[0].Attr {
//			if attr.Key == name {
//				return attr.Val
//			}
//		}
//		return ""
//	}
//
// // Text 获取文本内容
//
//	func (r *Response) Text() (string, error) {
//		if r.err != nil {
//			return "", r.err
//		}
//
//		if len(r.curNodes) == 0 {
//			return "", nil
//		}
//
//		var result string
//		if len(r.curNodes) == 1 {
//			result = htmlquery.InnerText(r.curNodes[0])
//		} else {
//			var texts []string
//			for _, node := range r.curNodes {
//				texts = append(texts, htmlquery.InnerText(node))
//			}
//			result = strings.Join(texts, "\n")
//		}
//
//		return strings.TrimSpace(result), nil
//	}
//
// // MustText 获取文本内容（忽略错误）
//
//	func (r *Response) MustText() string {
//		text, _ := r.Text()
//		return text
//	}
//
// // HTML 获取HTML内容
//
//	func (r *Response) HTML() (string, error) {
//		if r.err != nil {
//			return "", r.err
//		}
//
//		if len(r.curNodes) == 0 {
//			return "", nil
//		}
//
//		var buf bytes.Buffer
//		if len(r.curNodes) == 1 {
//			if err := html.Render(&buf, r.curNodes[0]); err != nil {
//				return "", err
//			}
//		} else {
//			for _, node := range r.curNodes {
//				if err := html.Render(&buf, node); err != nil {
//					return "", err
//				}
//				buf.WriteString("\n")
//			}
//		}
//		return buf.String(), nil
//	}
//
// // MustHTML 获取HTML内容（忽略错误）
//
//	func (r *Response) MustHTML() string {
//		html, _ := r.HTML()
//		return html
//	}
//
// // JSON 解析JSON
//
//	func (r *Response) JSON(v interface{}) error {
//		if r.err != nil {
//			return r.err
//		}
//		return json.Unmarshal(r.Body, v)
//	}
//
// // Bytes 获取原始字节
//
//	func (r *Response) Bytes() []byte {
//		return r.Body
//	}
//
// // String 获取响应体字符串
//
//	func (r *Response) String() string {
//		// 尝试自动检测字符集
//		contentType, ok := r.Headers["Content-Type"]
//		if ok && strings.Contains(strings.ToLower(contentType), "charset=") {
//			reader, err := charset.NewReader(bytes.NewReader(r.Body), contentType)
//			if err == nil {
//				if converted, err := io.ReadAll(reader); err == nil {
//					return string(converted)
//				}
//			}
//		}
//		return string(r.Body)
//	}
//
// // Regex 使用正则表达式匹配
//
//	func (r *Response) Regex(pattern string) ([]string, error) {
//		if r.err != nil {
//			return nil, r.err
//		}
//
//		re, err := regexp.Compile(pattern)
//		if err != nil {
//			return nil, err
//		}
//
//		text := r.String()
//		return re.FindAllString(text, -1), nil
//	}
//
// // MustRegex 使用正则表达式匹配（忽略错误）
//
//	func (r *Response) MustRegex(pattern string) []string {
//		matches, _ := r.Regex(pattern)
//		return matches
//	}
//
// // Find 查找元素（使用CSS选择器）
//
//	func (r *Response) Find(selector string) *Response {
//		return r.CSS(selector)
//	}
//
// // Children 获取子元素
//
//	func (r *Response) Children() *Response {
//		if r.err != nil || len(r.curNodes) == 0 {
//			return r
//		}
//
//		var children []*html.Node
//		for _, node := range r.curNodes {
//			for child := node.FirstChild; child != nil; child = child.NextSibling {
//				if child.Type == html.ElementNode {
//					children = append(children, child)
//				}
//			}
//		}
//		r.curNodes = children
//		return r
//	}
//
// // Parent 获取父元素
//
//	func (r *Response) Parent() *Response {
//		if r.err != nil || len(r.curNodes) == 0 {
//			return r
//		}
//
//		var parents []*html.Node
//		seen := make(map[*html.Node]bool)
//		for _, node := range r.curNodes {
//			if node.Parent != nil && node.Parent.Type == html.ElementNode {
//				if !seen[node.Parent] {
//					seen[node.Parent] = true
//					parents = append(parents, node.Parent)
//				}
//			}
//		}
//		r.curNodes = parents
//		return r
//	}
//
// // Next 获取下一个兄弟元素
//
//	func (r *Response) Next() *Response {
//		if r.err != nil || len(r.curNodes) == 0 {
//			return r
//		}
//
//		var nexts []*html.Node
//		for _, node := range r.curNodes {
//			for n := node.NextSibling; n != nil; n = n.NextSibling {
//				if n.Type == html.ElementNode {
//					nexts = append(nexts, n)
//					break
//				}
//			}
//		}
//		r.curNodes = nexts
//		return r
//	}
//
// // Prev 获取上一个兄弟元素
//
//	func (r *Response) Prev() *Response {
//		if r.err != nil || len(r.curNodes) == 0 {
//			return r
//		}
//
//		var prevs []*html.Node
//		for _, node := range r.curNodes {
//			for n := node.PrevSibling; n != nil; n = n.PrevSibling {
//				if n.Type == html.ElementNode {
//					prevs = append(prevs, n)
//					break
//				}
//			}
//		}
//		r.curNodes = prevs
//		return r
//	}
//
// // Each 遍历所有匹配元素
//
//	func (r *Response) Each(fn func(int, *html.Node)) *Response {
//		if r.err != nil {
//			return r
//		}
//
//		for i, node := range r.curNodes {
//			fn(i, node)
//		}
//		return r
//	}
//
// // Map 将元素映射为其他值
//
//	func (r *Response) Map(fn func(*html.Node) string) []string {
//		if r.err != nil || len(r.curNodes) == 0 {
//			return nil
//		}
//
//		result := make([]string, len(r.curNodes))
//		for i, node := range r.curNodes {
//			result[i] = fn(node)
//		}
//		return result
//	}
//
// // Length 获取匹配元素的数量
//
//	func (r *Response) Length() int {
//		if r.err != nil {
//			return 0
//		}
//		return len(r.curNodes)
//	}
//
// // Error 获取错误
//
//	func (r *Response) Error() error {
//		return r.err
//	}
//
// // Reset 重置选择器状态
//
//	func (r *Response) Reset() *Response {
//		r.curNodes = nil
//		r.err = nil
//		return r
//	}
//
// // Clone 克隆响应（用于链式调用的分支）
//
//	func (r *Response) Clone() *Response {
//		return &Response{
//			URL:        r.URL,
//			StatusCode: r.StatusCode,
//			Headers:    r.Headers,
//			Body:       r.Body,
//			Request:    r.Request,
//			Protocol:   r.Protocol,
//			rootNode:   r.rootNode,
//			curNodes:   r.curNodes,
//			err:        r.err,
//			text:       r.text,
//		}
//	}
package httpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/andybalholm/cascadia"
	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

// Response 封装了 HTTP 响应的所有信息，并提供链式调用的 HTML 解析功能。
// 注意：本类型的方法不是并发安全的，建议在单个 goroutine 中使用。
type Response struct {
	URL        string
	StatusCode int
	Headers    map[string]string
	Body       []byte
	Request    Request
	Protocol   string

	rootNode *html.Node   // 解析后的 HTML 根节点
	curNodes []*html.Node // 当前选中的节点集合
	err      error        // 链式调用中遇到的第一个错误
}

// NewResponse 创建一个新的 Response 实例。
func NewResponse(
	URL string,
	Status int,
	Headers map[string]string,
	Body []byte,
	Request *Request,
	Protocol string,
) *Response {
	return &Response{
		URL:        URL,
		StatusCode: Status,
		Headers:    Headers,
		Body:       Body,
		Request:    *Request,
		Protocol:   Protocol,
	}
}

// check 如果已有错误则直接返回 true，供内部方法快速退出。
func (r *Response) check() bool {
	return r.err != nil
}

// ensureNodes 确保 curNodes 不为空：如果 curNodes 为空但 rootNode 存在，则将其作为当前节点。
// 返回当前 curNodes 是否非空（且无错误）。
func (r *Response) ensureNodes() bool {
	if r.check() {
		return false
	}
	if len(r.curNodes) == 0 && r.rootNode != nil {
		r.curNodes = []*html.Node{r.rootNode}
	}
	return len(r.curNodes) > 0
}

// ParseHTML 解析 HTML，为后续查询做准备。如果已经解析过，不会重复解析。
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

// XPath 使用 XPath 表达式查询，将当前节点集合替换为匹配到的所有节点。
func (r *Response) XPath(expr string) *Response {
	if r.check() {
		return r
	}
	r.ParseHTML() // 确保已解析
	if !r.ensureNodes() {
		return r
	}

	var results []*html.Node
	for _, node := range r.curNodes {
		found := htmlquery.Find(node, expr)
		results = append(results, found...)
	}
	r.curNodes = results
	return r
}

// CSS 使用 CSS 选择器查询，将当前节点集合替换为匹配到的所有节点。
func (r *Response) CSS(selector string) *Response {
	if r.check() {
		return r
	}
	r.ParseHTML()
	if !r.ensureNodes() {
		return r
	}

	sel, err := cascadia.Compile(selector)
	if err != nil {
		r.err = fmt.Errorf("invalid CSS selector: %v", err)
		return r
	}

	var results []*html.Node
	for _, node := range r.curNodes {
		found := cascadia.QueryAll(node, sel)
		results = append(results, found...)
	}
	r.curNodes = results
	return r
}

// Find 是 CSS 的别名，提供类似 jQuery 的语义。
func (r *Response) Find(selector string) *Response {
	return r.CSS(selector)
}

// First 将当前节点集合缩减为第一个元素。
func (r *Response) First() *Response {
	if r.check() || len(r.curNodes) == 0 {
		return r
	}
	r.curNodes = []*html.Node{r.curNodes[0]}
	return r
}

// Last 将当前节点集合缩减为最后一个元素。
func (r *Response) Last() *Response {
	if r.check() || len(r.curNodes) == 0 {
		return r
	}
	r.curNodes = []*html.Node{r.curNodes[len(r.curNodes)-1]}
	return r
}

// Eq 将当前节点集合缩减为索引 i 处的元素（从 0 开始）。如果索引越界，集合变为空。
func (r *Response) Eq(i int) *Response {
	if r.check() || len(r.curNodes) == 0 {
		return r
	}
	if i < 0 || i >= len(r.curNodes) {
		r.curNodes = []*html.Node{}
		return r
	}
	r.curNodes = []*html.Node{r.curNodes[i]}
	return r
}

// Filter 用自定义函数过滤当前节点集合，保留使 fn 返回 true 的节点。
func (r *Response) Filter(fn func(*html.Node) bool) *Response {
	if r.check() {
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

// Attr 返回当前节点集合中第一个节点的指定属性值。如果节点不存在或属性缺失，返回空字符串。
func (r *Response) Attr(name string) string {
	if r.check() || len(r.curNodes) == 0 {
		return ""
	}
	for _, attr := range r.curNodes[0].Attr {
		if attr.Key == name {
			return attr.Val
		}
	}
	return ""
}

// Text 返回当前节点集合中所有节点的文本内容拼接（多个节点用换行分隔），并去除首尾空白。
func (r *Response) Text() (string, error) {
	if r.check() {
		return "", r.err
	}
	if len(r.curNodes) == 0 {
		return "", nil
	}
	if len(r.curNodes) == 1 {
		return strings.TrimSpace(htmlquery.InnerText(r.curNodes[0])), nil
	}
	var texts []string
	for _, node := range r.curNodes {
		texts = append(texts, strings.TrimSpace(htmlquery.InnerText(node)))
	}
	return strings.Join(texts, "\n"), nil
}

// MustText 返回 Text 的结果，忽略错误。如果出错返回空字符串。
func (r *Response) MustText() string {
	s, _ := r.Text()
	return s
}

// HTML 返回当前节点集合中所有节点的 HTML 源码拼接（多个节点用换行分隔）。
func (r *Response) HTML() (string, error) {
	if r.check() {
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

// MustHTML 返回 HTML 的结果，忽略错误。
func (r *Response) MustHTML() string {
	s, _ := r.HTML()
	return s
}

// JSON 将响应体解析为 JSON 并存入 v。
func (r *Response) JSON(v interface{}) error {
	if r.err != nil {
		return r.err
	}
	return json.Unmarshal(r.Body, v)
}

// Bytes 返回原始响应体字节切片。
func (r *Response) Bytes() []byte {
	return r.Body
}

// String 返回响应体字符串，自动根据 Content-Type 中的 charset 进行字符集转换。
func (r *Response) String() string {
	if contentType, ok := r.Headers["Content-Type"]; ok {
		if strings.Contains(strings.ToLower(contentType), "charset=") {
			reader, err := charset.NewReader(bytes.NewReader(r.Body), contentType)
			if err == nil {
				converted, err := io.ReadAll(reader)
				if err == nil {
					return string(converted)
				}
			}
		}
	}
	return string(r.Body)
}

// Regex 在响应体字符串上执行正则匹配，返回所有匹配项。
func (r *Response) Regex(pattern string) ([]string, error) {
	if r.err != nil {
		return nil, r.err
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return re.FindAllString(r.String(), -1), nil
}

// MustRegex 返回 Regex 的结果，忽略错误。
func (r *Response) MustRegex(pattern string) []string {
	matches, _ := r.Regex(pattern)
	return matches
}

// Children 将当前节点集合替换为每个节点的子元素节点（仅元素节点）。
func (r *Response) Children() *Response {
	if r.check() {
		return r
	}
	if !r.ensureNodes() {
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

// Parent 将当前节点集合替换为每个节点的父元素节点（仅元素节点，去重）。
func (r *Response) Parent() *Response {
	if r.check() {
		return r
	}
	if !r.ensureNodes() {
		return r
	}
	parentsMap := make(map[*html.Node]bool)
	var parents []*html.Node
	for _, node := range r.curNodes {
		if node.Parent != nil && node.Parent.Type == html.ElementNode {
			if !parentsMap[node.Parent] {
				parentsMap[node.Parent] = true
				parents = append(parents, node.Parent)
			}
		}
	}
	r.curNodes = parents
	return r
}

// Next 将当前节点集合替换为每个节点的下一个兄弟元素节点（仅第一个）。
func (r *Response) Next() *Response {
	if r.check() {
		return r
	}
	if !r.ensureNodes() {
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

// Prev 将当前节点集合替换为每个节点的上一个兄弟元素节点（仅第一个）。
func (r *Response) Prev() *Response {
	if r.check() {
		return r
	}
	if !r.ensureNodes() {
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

// Each 遍历当前节点集合，对每个节点执行 fn。
func (r *Response) Each(fn func(int, *html.Node)) *Response {
	if r.check() {
		return r
	}
	for i, node := range r.curNodes {
		fn(i, node)
	}
	return r
}

// Map 将当前节点集合映射为字符串切片，每个元素由 fn 生成。
func (r *Response) Map(fn func(*html.Node) string) []string {
	if r.check() || len(r.curNodes) == 0 {
		return nil
	}
	result := make([]string, len(r.curNodes))
	for i, node := range r.curNodes {
		result[i] = fn(node)
	}
	return result
}

// Length 返回当前节点集合中的元素个数。
func (r *Response) Length() int {
	if r.check() {
		return 0
	}
	return len(r.curNodes)
}

// Error 返回链式调用过程中遇到的第一个错误，如果没有错误则返回 nil。
func (r *Response) Error() error {
	return r.err
}

// Reset 重置当前节点集合为空，并清空错误，但保留已解析的根节点（可用于开始新的查询）。
func (r *Response) Reset() *Response {
	r.curNodes = nil
	r.err = nil
	return r
}

// Clone 返回一个当前 Response 的浅拷贝，可用于分支链式调用。
// 克隆后的 Response 共享相同的根节点和当前节点集合（但 curNodes 切片是独立的），
// 因此对一个副本的查询不会影响另一个。
func (r *Response) Clone() *Response {
	clone := *r
	// 复制 curNodes 切片，避免相互影响
	if r.curNodes != nil {
		clone.curNodes = make([]*html.Node, len(r.curNodes))
		copy(clone.curNodes, r.curNodes)
	}
	return &clone
}
