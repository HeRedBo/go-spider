package test

import (
	"fmt"
	"go-spider/zhenai/parser"
	"os"
	"testing"

	"github.com/gookit/goutil/dump"
)

func TestParseCityUserList(t *testing.T) {
	content, err := os.ReadFile("city_user_list.html")
	if err != nil {
		t.Fatal(err)
	}
	result := parser.ParseCityUserList(content, "")

	var jsonStr = result.Requests[0].Text

	filePath := "merber_list.json" // 相对于 fetcher 目录的路径
	err = os.WriteFile(filePath, []byte(jsonStr), 0644)
	if err != nil {
		panic(err)
	}
	fmt.Printf("内容已成功写入到 %s 文件\n", filePath)

	dump.P(result.Requests[0])

}
