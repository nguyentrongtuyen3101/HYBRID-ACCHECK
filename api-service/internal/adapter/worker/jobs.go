// Package worker chạy job phân tích nền theo chu kỳ.
package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// JobProcessor là use case xử lý job (hiện thực: analysis.Service).
type JobProcessor interface {
	ProcessNextJob(ctx context.Context) (bool, error)
}

// RunJobs chạy n worker; mỗi worker lấy job đang chờ, hết job thì nghỉ interval.
func RunJobs(ctx context.Context, p JobProcessor, n int, interval time.Duration, log *slog.Logger) {
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for ctx.Err() == nil {
				processed, err := p.ProcessNextJob(ctx)
				if err != nil && ctx.Err() == nil { // lỗi do đang tắt service thì bỏ qua
					log.Error("analysis job error", "worker", id, "err", err)
				}
				if processed && err == nil {
					continue
				}
				select {
				case <-ctx.Done():
				case <-time.After(interval):
				}
			}
		}(i)
	}
	wg.Wait()
}
