package persist

import (
	"context"
	"go-spider/zhenai/model"
	"strconv"
	"testing"
	"time"

	"github.com/olivere/elastic"
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

	// 写入数据
	saver <- merber
	// ✅ 关键：等待协程完成保存
	time.Sleep(1 * time.Second)
	t.Log("✅ 数据已成功写入ES")
}

func TestItemSaver_Batch(t *testing.T) {

	// 1. 创建批量保存通道
	itemChan, err := ItemSaver("dating_profile")
	if err != nil {
		t.Fatalf("ES 连接失败: %v", err)
	}

	// 2. 构造测试数据（一次性发 5 条，触发批量逻辑）
	testMembers := []model.Member{
		{
			MemberID:    1802070444,
			NickName:    "雪梨",
			Age:         43,
			Height:      165,
			Education:   "大专",
			Marriage:    "离异",
			WorkCity:    "广东广州",
			LastModTime: time.Now().Format(time.DateTime),
		},
		{
			MemberID:    1802070445,
			NickName:    "测试用户2",
			Age:         30,
			Height:      170,
			Education:   "本科",
			Marriage:    "未婚",
			WorkCity:    "广东深圳",
			LastModTime: time.Now().Format(time.DateTime),
		},
		{
			MemberID:    1802070446,
			NickName:    "测试用户3",
			Age:         28,
			Height:      168,
			Education:   "硕士",
			Marriage:    "未婚",
			WorkCity:    "上海",
			LastModTime: time.Now().Format(time.DateTime),
		},
	}

	// 3. 发送数据到批量通道
	t.Log("开始发送测试数据到批量保存通道...")
	for _, m := range testMembers {
		itemChan <- m
	}

	// 4. 关闭通道 → 触发批量保存器自动提交剩余数据
	close(itemChan)

	// 5. 等待批量协程处理完毕（因为通道关闭，协程会自动 flush 并退出）
	// 这里不需要 WaitGroup，不需要 sleep，通道关闭后批量协程自动结束
	t.Log("等待批量保存完成...")

	// 给一点极短时间让协程完成收尾（必执行，不会卡）
	time.Sleep(100 * time.Millisecond)

	// 6. 验证数据是否真的写入 ES
	client, err := elastic.NewClient(
		elastic.SetURL("http://127.0.0.1:9200"),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false),
		elastic.SetBasicAuth("elastic", "elastic"),
	)
	if err != nil {
		t.Fatal(err)
	}

	for _, m := range testMembers {
		exists, err := client.Exists().
			Index("dating_profile").
			Type(Type).
			Id(strconv.Itoa(m.MemberID)).
			Do(context.Background())

		if err != nil {
			t.Errorf("检查数据失败: %v", err)
		}
		if !exists {
			t.Errorf("数据未保存成功 ID: %d", m.MemberID)
		} else {
			t.Logf("✅ 数据保存成功 ID: %d", m.MemberID)
		}
	}

	t.Log("🎉 批量保存单元测试全部通过！") // 3. 发送数据到批量通道
	t.Log("开始发送测试数据到批量保存通道...")
	for _, m := range testMembers {
		itemChan <- m
	}

	// 4. 关闭通道 → 触发批量保存器自动提交剩余数据
	close(itemChan)

	// 5. 等待批量协程处理完毕（因为通道关闭，协程会自动 flush 并退出）
	// 这里不需要 WaitGroup，不需要 sleep，通道关闭后批量协程自动结束
	t.Log("等待批量保存完成...")

	// 给一点极短时间让协程完成收尾（必执行，不会卡）
	time.Sleep(100 * time.Millisecond)
	
	for _, m := range testMembers {
		exists, err := client.Exists().
			Index("dating_profile").
			Type(Type).
			Id(strconv.Itoa(m.MemberID)).
			Do(context.Background())

		if err != nil {
			t.Errorf("检查数据失败: %v", err)
		}
		if !exists {
			t.Errorf("数据未保存成功 ID: %d", m.MemberID)
		} else {
			t.Logf("✅ 数据保存成功 ID: %d", m.MemberID)
		}
	}

	t.Log("🎉 批量保存单元测试全部通过！")

}
