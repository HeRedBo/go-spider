package parser

import (
	"encoding/json"
	"fmt"
	"go-spider/types"
	"go-spider/zhenai/model"
)

func ParseMemberListProfile(contents []byte, url string) types.ParseResult {
	data := model.CityDetailList{}
	result := types.ParseResult{}
	if err := json.Unmarshal(contents, &data); err == nil {
		member_list := data.MemberListData.MemberList
		for _, item := range member_list {
			member := model.Member(item)
			result.Items = append(result.Items, member)
		}
	} else {
		fmt.Println("error:" + url)
		panic(err)
	}
	return result
}
