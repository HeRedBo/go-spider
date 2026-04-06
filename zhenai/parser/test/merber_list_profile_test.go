package test

import (
	"fmt"
	"go-spider/zhenai/parser"
	"os"
	"testing"

	"github.com/gookit/goutil/dump"
)

func TestParseMemberListProfile(t *testing.T) {
	bytes, err := os.ReadFile("merber_list.json")
	if err != nil {
		fmt.Println("读取 json 文件报错", err)
		return
	}
	resp := parser.ParseMemberListProfile(bytes, "http://www.zhenai.com/zhenghun/akesu/1")
	dump.P(resp)
}
