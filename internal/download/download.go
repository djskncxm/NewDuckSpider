package download

import (
	"fmt"
	"github.com/djskncxm/NewDuckSpider/pkg/httpc"
	"io"
	"net/http"
)

type Download struct{}

func InitDownload() Download {
	return Download{}
}

func (d *Download) Fetch(request *httpc.Request) *httpc.Response {
	resp, err := http.Get(request.URL)
	if err != nil {
		fmt.Println("请求失败:", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取响应失败:", err)
		return nil
	}
	fmt.Println(string(body))

	return nil
}
