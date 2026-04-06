package test

import (
	"go-spider/single-task/zhenai/parser"
	"os"
	"testing"

	"github.com/gookit/goutil/dump"
)

func TestParseCityList(t *testing.T) {
	contents, err := os.ReadFile("city_list.html")
	if err != nil {
		panic(err)
	}
	result := parser.ParseCityList(contents)
	dump.P(result)
}
