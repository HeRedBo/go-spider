package fetcher

import (
	"fmt"
	"os"
	"testing"

	"github.com/gookit/goutil/dump"
)

//fetch 测试用例

func TestFetch(t *testing.T) {
	// s, err := Fetch("http://www.zhenai.com/zhenghun", "GET")
	s, err := FetchWithRetry("http://www.zhenai.com/zhenghun/akesu/1", "GET")
	if err != nil {
		panic(err)
	}
	//fmt.Printf("%s\n", s)
	dump.P(string(s))

	// 写入内容到文件
	filePath := "city_user_list.html" // 相对于 fetcher 目录的路径
	err = os.WriteFile(filePath, s, 0644)
	if err != nil {
		panic(err)
	}
	fmt.Printf("内容已成功写入到 %s 文件\n", filePath)
}
