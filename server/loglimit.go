package server

import (
	"log"
	"sync"
	"time"
)

type logLimiter struct {
	mu       sync.Mutex
	interval time.Duration
	last     map[string]time.Time
}

func newLogLimiter(interval time.Duration) *logLimiter {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	return &logLimiter{
		interval: interval,
		last:     make(map[string]time.Time),
	}
}

func (l *logLimiter) Logf(key, format string, args ...any) {
	if l == nil {
		log.Printf(format, args...)
		return
	}
	now := time.Now()
	l.mu.Lock()
	last := l.last[key]
	if now.Sub(last) < l.interval {
		l.mu.Unlock()
		return
	}
	l.last[key] = now
	l.mu.Unlock()

	log.Printf(format, args...)
}
