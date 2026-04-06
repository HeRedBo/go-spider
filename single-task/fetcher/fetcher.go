package fetcher

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/avast/retry-go"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

var client = &http.Client{
	Timeout: 10 * time.Second, // 重要：防止请求卡死
}

// 随机UA列表
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Mobile Safari/537.36",
}

func randomUA() string {
	return userAgents[rand.Intn(len(userAgents))]
}

// 抓取网页，返回 UTF-8 编码的内容
func Fetch(url, method string) ([]byte, error) {

	// 构造请求
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	// 添加随机 UA
	req.Header.Set("User-Agent", randomUA())
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// 判断状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status error: %d", resp.StatusCode)
	}
	// 编码识别 + 转 UTF-8
	bodyReader := bufio.NewReader(resp.Body)
	encoding := determineEncoding(bodyReader)
	utf8Reader := transform.NewReader(bodyReader, encoding.NewDecoder())
	return io.ReadAll(utf8Reader)
}

func FetchWithRetry(url, method string) ([]byte, error) {
	var body []byte

	// 使用 retry 库做重试
	err := retry.Do(
		func() error {
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				return err
			}

			// 随机UA
			req.Header.Set("User-Agent", randomUA())

			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return fmt.Errorf("http status: %d", resp.StatusCode)
			}

			// 编码识别 + 转 UTF-8
			br := bufio.NewReader(resp.Body)
			enc := determineEncoding(br)
			utf8Reader := transform.NewReader(br, enc.NewDecoder())

			body, err = io.ReadAll(utf8Reader)
			return err
		},
		retry.Attempts(3),          // 最多重试3次
		retry.Delay(1*time.Second), // 间隔1s
		retry.MaxDelay(3*time.Second),
	)

	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	return body, nil
}

// determineEncoding 自动检测网页编码
func determineEncoding(r *bufio.Reader) encoding.Encoding {
	// 偷看前1024字节判断编码（不消耗数据流）
	bytes, err := r.Peek(1024)
	if err != nil {
		return unicode.UTF8
	}
	e, _, _ := charset.DetermineEncoding(bytes, "")
	return e
}
