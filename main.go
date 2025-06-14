package main

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"
)

type Result struct {
	Response string
	Err      error
	Index    int    // 请求顺序索引
	URL      string // 原始URL
	Status   string // 状态标识
	Duration time.Duration
}

func mockRequest(url string, index int, wg *sync.WaitGroup, resultChan chan<- Result) {
	defer wg.Done()

	//start := time.Now()
	delay := time.Duration(rand.Intn(1000)) * time.Millisecond
	time.Sleep(delay)

	result := Result{
		Index:    index,
		URL:      url,
		Duration: delay,
	}

	if rand.Intn(10) < 2 {
		result.Status = "失败"
		result.Err = fmt.Errorf("请求失败 [%s] (耗时: %v)", url, delay)
	} else {
		result.Status = "成功"
		result.Response = fmt.Sprintf("结果数据 [%s]", url[:7])
	}

	resultChan <- result
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// 1. 创建有序的URL列表(带序号)
	urls := []string{
		"https://api.service.com/user",
		"https://api.service.com/products",
		"https://api.service.com/orders",
		"https://api.service.com/inventory",
		"https://api.service.com/payments",
		"https://api.service.com/shipping",
		"https://api.service.com/reviews",
		"https://api.service.com/analytics",
		"https://api.service.com/notifications",
		"https://api.service.com/recommendations",
	}

	// 2. 创建带缓冲的结果通道(双倍容量)
	resultChan := make(chan Result, len(urls)*2)

	// 3. 使用WaitGroup确保所有请求完成
	var wg sync.WaitGroup
	totalStart := time.Now()

	// 4. 启动所有并发请求(携带索引序号)
	for i, url := range urls {
		wg.Add(1)
		go mockRequest(url, i, &wg, resultChan)
	}

	// 5. 后台聚合结果(使用索引确保顺序)
	results := make([]Result, len(urls))
	aggregateDone := make(chan struct{})

	go func() {
		defer close(aggregateDone)
		wg.Wait()
		close(resultChan) // 确保所有结果已发送
	}()

	// 6. 按完成顺序接收结果(立即显示)
	fmt.Println("开始并发请求...")
	fmt.Printf("%-5s %-12s %-8s %-45s %s\n", "序号", "耗时", "状态", "请求地址", "详情")
	fmt.Println("----------------------------------------------------------------------")

	// 创建临时存储和顺序跟踪器
	tempResults := make([]Result, 0, len(urls))
	nextIndex := 0

	// 实时处理和显示结果
	for result := range resultChan {
		// 临时存储结果
		tempResults = append(tempResults, result)

		// 按完成顺序显示
		fmt.Printf("%-5d %-12v %-8s %-45s %s\n",
			result.Index,
			result.Duration,
			result.Status,
			result.URL,
			result.Status+" (收到结果)")

		// 按请求顺序显示结果(当达到nextIndex时)
		sort.Slice(tempResults, func(i, j int) bool {
			return tempResults[i].Index < tempResults[j].Index
		})

		for len(tempResults) > 0 && tempResults[0].Index == nextIndex {
			r := tempResults[0]
			tempResults = tempResults[1:]
			results[nextIndex] = r
			nextIndex++

			if r.Err != nil {
				fmt.Printf("❌ [%d] 错误结果: %v\n", r.Index, r.Err)
			} else {
				fmt.Printf("✅ [%d] 有序结果: %s\n", r.Index, r.Response)
			}
		}
	}

	// 7. 确保所有结果都按顺序处理
	<-aggregateDone

	// 8. 打印最终汇总报告(按请求顺序)
	fmt.Println("\n======================= 最终结果(按请求顺序) =======================")
	fmt.Printf("%-5s %-12s %-8s %-45s %s\n", "序号", "耗时", "状态", "请求地址", "详情")
	fmt.Println("----------------------------------------------------------------------")

	successCount := 0
	for i, r := range results {
		if r.Err == nil {
			successCount++
		}

		fmt.Printf("%-5d %-12v %-8s %-45s ", i, r.Duration, r.Status, r.URL)
		if r.Err != nil {
			fmt.Printf("❌ %v\n", r.Err)
		} else {
			fmt.Printf("✅ %s\n", r.Response)
		}
	}

	// 9. 统计信息
	totalTime := time.Since(totalStart)
	fmt.Println("\n======================= 执行统计 =======================")
	fmt.Printf("总请求数: %d\n", len(urls))
	fmt.Printf("成功请求: %d\n", successCount)
	fmt.Printf("失败请求: %d\n", len(urls)-successCount)
	fmt.Printf("成功率: %.1f%%\n", float64(successCount)/float64(len(urls))*100)
	fmt.Printf("总执行时间: %v (%.1fms/请求)\n", totalTime,
		float64(totalTime.Microseconds())/1000/float64(len(urls)))

	// 10. 显示最快和最慢请求
	if len(results) > 0 {
		sort.Slice(results, func(i, j int) bool {
			return results[i].Duration < results[j].Duration
		})

		fastest := results[0]
		slowest := results[len(results)-1]

		fmt.Println("\n======================= 性能分析 =======================")
		fmt.Printf("最快请求: #%d %s (%v)\n", fastest.Index, fastest.URL, fastest.Duration)
		fmt.Printf("最慢请求: #%d %s (%v)\n", slowest.Index, slowest.URL, slowest.Duration)
		fmt.Printf("速度差距: %v (%.1f%%)\n",
			slowest.Duration-fastest.Duration,
			float64(slowest.Duration-fastest.Duration)/float64(fastest.Duration)*100)
	}
}
