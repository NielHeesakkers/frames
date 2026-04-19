// internal/auth/ratelimit.go
package auth

import (
	"sync"
	"time"
)

type LoginLimiter struct {
	mu      sync.Mutex
	max     int
	window  time.Duration
	buckets map[string]*bucket
}

type bucket struct {
	count    int
	resetsAt time.Time
}

func NewLoginLimiter(max int, window time.Duration) *LoginLimiter {
	return &LoginLimiter{
		max: max, window: window,
		buckets: map[string]*bucket{},
	}
}

func (l *LoginLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	b := l.buckets[key]
	if b == nil || now.After(b.resetsAt) {
		l.buckets[key] = &bucket{count: 1, resetsAt: now.Add(l.window)}
		return true
	}
	if b.count >= l.max {
		return false
	}
	b.count++
	return true
}
