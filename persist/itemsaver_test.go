package persist

import (
	"go-spider/zhenai/model"
	"testing"

	"github.com/gookit/goutil/dump"
)

func TestItemSaver(t *testing.T) {
	merber := model.Member{
		NickName:         "雪梨",
		AvatarURL:        "https://photo.zastatic.com/images/photo/25073/100290572/2483971534708801550.jpg",
		Car:              "未填写",
		Children:         "有孩子但不在身边",
		Education:        "大专",
		Height:           165,
		House:            "已购房",
		IsRecommend:      0,
		IntroduceContent: "一个暖男家务一起做家庭观念强来一场结婚为目的的恋爱本人有点点内向，社恐，但认识久了你发现这货就是一个 2货哈哈还有最重要的一点，你觉得原生家庭很重要吗？我希望你是一个小话痨，话比我多，我是一个不错的聆听者。100度的水也会有0度的时候，双向奔赴的爱才是最想要的，让我携手同行。",
		LastModTime:      "2026-04-01 15:01:53",
		Marriage:         "离异",
		MemberID:         1802070444,
		ObjectAge:        "43-50岁",
		ObjectHight:      "165-170cm",
		ObjectMarriage:   "离异",
		ObjectSalary:     "8001-12000元",
		Sex:              0,
		WorkCity:         "广东广州",
		Age:              43,
		Salary:           "8001-12000元",
	}
	saver, err := ItemSaver("dating_profile")
	if err != nil {
		return
	}
	dump.P(saver)
	dump.P(merber)

}
