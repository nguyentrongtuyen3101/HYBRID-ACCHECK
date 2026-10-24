package security

import (
	"sync"
	"time"
)

// LoginLimiter chặn brute-force: quá maxFailures lần sai trong window thì khoá
// key (ip|email) đến hết window. Lưu in-memory, đủ cho 1 instance; chạy nhiều
// instance thì chuyển sang Redis.
type LoginLimiter struct {
	mu          sync.Mutex
	maxFailures int
	window      time.Duration
	entries     map[string]*limiterEntry
}

type limiterEntry struct {
	failures int
	first    time.Time
}

func NewLoginLimiter(maxFailures int, window time.Duration) *LoginLimiter {
	return &LoginLimiter{maxFailures: maxFailures, window: window, entries: map[string]*limiterEntry{}}
}

func (l *LoginLimiter) Blocked(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[key]
	if !ok {
		return false
	}
	if time.Since(e.first) > l.window {
		delete(l.entries, key)
		return false
	}
	return e.failures >= l.maxFailures
}

func (l *LoginLimiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	e, ok := l.entries[key]
	if !ok || now.Sub(e.first) > l.window {
		l.entries[key] = &limiterEntry{failures: 1, first: now}
	} else {
		e.failures++
	}
	// Dọn entry hết hạn để map không phình mãi
	if len(l.entries) > 10000 {
		for k, v := range l.entries {
			if now.Sub(v.first) > l.window {
				delete(l.entries, k)
			}
		}
	}
}

func (l *LoginLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}
