package model

import "strconv"

type Member struct {
	Age              int
	AvatarURL        string
	Car              string
	Children         string
	Constellation    string
	Education        string
	Height           int
	House            string
	IntroduceContent string
	IsRecommend      int
	LastModTime      string
	Marriage         string
	MemberID         int
	NickName         string
	ObjectAge        string
	ObjectHight      string
	ObjectMarriage   string
	ObjectSalary     string
	Occupation       string
	Salary           string
	Sex              int
	WorkCity         string
}

func (m Member) ID() string {
	return strconv.Itoa(m.MemberID)
}

func (m Member) IsPersistable() bool {
	return true
}
