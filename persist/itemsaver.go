package persist

import (
	"context"
	"fmt"
	"go-spider/zhenai/model"
	"log"
	"strconv"
	"time"

	"github.com/olivere/elastic"
)

var Type = "zhenai"

// 批量配置
const (
	batchSize    = 50              // 50条触发批量提交
	flushTimeout = 1 * time.Second // 1秒兜底超时
)

func ItemSaver(index string) (chan interface{}, error) {
	out := make(chan interface{})
	// 判断索引是否存在并创建索引
	client, err := elastic.NewClient(
		elastic.SetURL("http://127.0.0.1:9200"),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false),
		elastic.SetBasicAuth("elastic", "elastic"),
	)
	if err != nil {
		log.Println("❌ ES客户端初始化失败:", err)
		return nil, err
	}
	exists, err := client.IndexExists(index).Do(context.Background())
	if err != nil {
		// Handle error
		log.Println("es exists query fail:", err)
		return nil, err
		//return "",nil
	}
	if !exists {
		_, err := client.CreateIndex(index).
			Do(context.Background())
		if err != nil {
			log.Println("es CreateIndex  fail:", err)
			// Handle error
			return nil, err
		}
	}

	go func() {
		itemCount := 0
		for {
			item := <-out
			id, err := save(client, index, item)
			if err != nil {
				log.Printf("item save :error saving item %v : %v\n", item, err)
			}
			fmt.Printf("es save client count:%d id %v ,item : %v\n", itemCount, id, item)
			itemCount++
		}
	}()
	return out, nil
}

// region 批量处理数据
func batchSaveLoop(client *elastic.Client, index string, itemChan <-chan interface{}) {
	var buffer []interface{}
	ticker := time.NewTicker(flushTimeout)
	defer ticker.Stop()

	// ✅ 统一创建 ctx（用于批量提交）
	ctx := context.Background()

	for {
		select {
		case item, ok := <-itemChan:
			if !ok {
				// channel关闭，提交剩余数据
				flushBatch(ctx, client, index, buffer)
				return
			}

			buffer = append(buffer, item)

			// 达到数量 → 立即提交
			if len(buffer) >= batchSize {
				flushBatch(ctx, client, index, buffer)
				buffer = nil
			}

		case <-ticker.C:
			// 超时 → 提交缓冲区数据
			if len(buffer) > 0 {
				flushBatch(ctx, client, index, buffer)
				buffer = nil
			}
		}
	}
}

// flushBatch 真正执行批量写入ES
func flushBatch(ctx context.Context, client *elastic.Client, index string, items []interface{}) {
	if len(items) == 0 {
		return
	}

	bulkRequest := client.Bulk().Index(index).Type(Type)

	for _, item := range items {
		member, ok := item.(model.Member)
		if !ok {
			log.Printf("⚠️  非Member类型，跳过: %v", item)
			continue
		}

		id := strconv.Itoa(member.MemberID)

		// 构造ES批量操作：存在则更新，不存在则创建（upsert）
		req := elastic.NewBulkUpdateRequest().
			Id(id).
			Doc(item).
			Upsert(item) // 没有就创建，有就更新

		bulkRequest.Add(req)
	}
	// 执行批量请求
	res, err := bulkRequest.Do(ctx)
	if err != nil {
		log.Printf("❌ 批量提交失败: %v", err)
		return
	}
	// 日志
	log.Printf(
		"✅ 批量提交成功 | 总数:%d | 成功:%d | 失败:%d",
		len(items),
		len(res.Succeeded()),
		len(res.Failed()),
	)

	// 打印失败项（调试用）
	if len(res.Failed()) > 0 {
		for _, fail := range res.Failed() {
			log.Printf("❌ 失败ID:%s Error:%s", fail.Id, fail.Error)
		}
	}
}

// endregion
// region 数据单个处理
func save(client *elastic.Client, index string, item interface{}) (id string, err error) {
	//data := model.Member(item)
	data := item.(model.Member)
	IdString := strconv.Itoa(data.MemberID)
	// 执行ES请求需要提供一个上下文对象
	ctx := context.Background()
	exist, err := client.Exists().
		Index(index).
		Type(Type).
		Id(IdString).
		Do(ctx)
	if err != nil {
		return "", err
	}
	if exist {
		resp, err := client.Update().
			Index(index).
			Type(Type).
			Id(IdString).
			Doc(item).
			Do(ctx)
		if err != nil {
			return "", err
		}
		return resp.Id, nil
	} else {
		resp, err := client.Index().
			Index(index).
			Type(Type).
			Id(IdString).
			BodyJson(item).
			Do(ctx)
		if err != nil {
			return "", err
		}
		return resp.Id, nil
	}

}

// endregion
