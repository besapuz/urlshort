package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type LoadTestConfig struct {
	BaseURL       string
	TotalRequests int
	Concurrency   int
	Duration      time.Duration
	Method        string
	Endpoint      string
	Body          []byte
}

type LoadTestResult struct {
	TotalRequests     int64
	Successful        int64
	Failed            int64
	TotalDuration     time.Duration
	RequestsPerSecond float64
	AverageLatency    time.Duration
	MinLatency        time.Duration
	MaxLatency        time.Duration
	ErrorRate         float64
}

func runLoadTest(cfg LoadTestConfig) LoadTestResult {
	var (
		successful int64
		failed     int64
		totalTime  int64
		minLatency = time.Hour
		maxLatency time.Duration
		start      = time.Now()
		wg         sync.WaitGroup
		latencies  = make(chan time.Duration, cfg.TotalRequests)
	)

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        cfg.Concurrency * 2,
			MaxIdleConnsPerHost: cfg.Concurrency * 2,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// Семафор для ограничения параллелизма
	sem := make(chan struct{}, cfg.Concurrency)

	for i := 0; i < cfg.TotalRequests; i++ {
		wg.Add(1)
		go func(reqNum int) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			reqStart := time.Now()

			url := cfg.BaseURL + cfg.Endpoint
			req, err := http.NewRequest(cfg.Method, url, bytes.NewReader(cfg.Body))
			if err != nil {
				atomic.AddInt64(&failed, 1)
				return
			}

			if cfg.Method == "POST" {
				req.Header.Set("Content-Type", "application/json")
			}

			resp, err := client.Do(req)
			latency := time.Since(reqStart)
			latencies <- latency

			if err != nil {
				atomic.AddInt64(&failed, 1)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				atomic.AddInt64(&successful, 1)
			} else {
				atomic.AddInt64(&failed, 1)
			}

			// Обновляем минимальную и максимальную задержку
			if latency < minLatency {
				minLatency = latency
			}
			if latency > maxLatency {
				maxLatency = latency
			}
			atomic.AddInt64(&totalTime, latency.Nanoseconds())
		}(i)

		// Небольшая пауза между запуском горутин
		if i%100 == 0 {
			time.Sleep(1 * time.Millisecond)
		}
	}

	wg.Wait()
	close(latencies)

	totalDuration := time.Since(start)

	// Вычисляем среднюю задержку
	var totalLatency time.Duration
	count := 0
	for latency := range latencies {
		totalLatency += latency
		count++
	}

	var avgLatency time.Duration
	if count > 0 {
		avgLatency = totalLatency / time.Duration(count)
	}

	total := successful + failed
	errorRate := 0.0
	if total > 0 {
		errorRate = float64(failed) / float64(total) * 100
	}

	return LoadTestResult{
		TotalRequests:     total,
		Successful:        successful,
		Failed:            failed,
		TotalDuration:     totalDuration,
		RequestsPerSecond: float64(total) / totalDuration.Seconds(),
		AverageLatency:    avgLatency,
		MinLatency:        minLatency,
		MaxLatency:        maxLatency,
		ErrorRate:         errorRate,
	}
}

func main() {
	cfg := LoadTestConfig{
		BaseURL:       "http://localhost:8080",
		TotalRequests: 10000,
		Concurrency:   100,
		Duration:      30 * time.Second,
	}

	// Тест 1: POST /api/shorten
	fmt.Println("=== Тест 1: POST /api/shorten ===")
	shortenReq := map[string]string{
		"url": "https://example.com/" + time.Now().String(),
	}
	body, _ := json.Marshal(shortenReq)

	cfg.Method = "POST"
	cfg.Endpoint = "/api/shorten"
	cfg.Body = body

	result1 := runLoadTest(cfg)
	printResult(result1)

	// Тест 2: GET /
	fmt.Println("\n=== Тест 2: GET / ===")
	cfg.Method = "GET"
	cfg.Endpoint = "/"
	cfg.Body = nil

	result2 := runLoadTest(cfg)
	printResult(result2)
}

func printResult(r LoadTestResult) {
	fmt.Printf("Всего запросов: %d\n", r.TotalRequests)
	fmt.Printf("Успешных: %d\n", r.Successful)
	fmt.Printf("Неудачных: %d\n", r.Failed)
	fmt.Printf("Общее время: %v\n", r.TotalDuration)
	fmt.Printf("Запросов в секунду: %.2f\n", r.RequestsPerSecond)
	fmt.Printf("Средняя задержка: %v\n", r.AverageLatency)
	fmt.Printf("Минимальная задержка: %v\n", r.MinLatency)
	fmt.Printf("Максимальная задержка: %v\n", r.MaxLatency)
	fmt.Printf("Процент ошибок: %.2f%%\n", r.ErrorRate)
}
