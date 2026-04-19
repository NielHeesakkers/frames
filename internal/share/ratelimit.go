// internal/share/ratelimit.go
package share

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu   sync.Mutex
	max  int
	win  time.Duration
	bkts map[string]*bkt
}

type bkt struct {
	count   int
	resetAt time.Time
}

func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{max: max, win: window, bkts: map[string]*bkt{}}
}

func (l *RateLimiter) Allow(token string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	b := l.bkts[token]
	if b == nil || now.After(b.resetAt) {
		l.bkts[token] = &bkt{count: 1, resetAt: now.Add(l.win)}
		return true
	}
	if b.count >= l.max {
		return false
	}
	b.count++
	return true
}
