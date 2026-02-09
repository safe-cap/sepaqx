package server

import (
	"sync"
	"time"
)

type ipLimiter struct {
	mu      sync.Mutex
	rate    float64
	burst   float64
	ttl     time.Duration
	gcEvery time.Duration
	lastGC  time.Time
	byIP    map[string]*bucket
}

type bucket struct {
	tokens float64
	last   time.Time
}

func newIPLimiter(rate float64, burst int, ttl time.Duration) *ipLimiter {
	if rate <= 0 {
		rate = 10
	}
	if burst < 1 {
		burst = 1
	}
	gcEvery := ttl / 2
	if ttl > 0 && gcEvery < time.Second {
		gcEvery = time.Second
	}
	return &ipLimiter{
		rate:    rate,
		burst:   float64(burst),
		ttl:     ttl,
		gcEvery: gcEvery,
		byIP:    make(map[string]*bucket),
	}
}

func (l *ipLimiter) Allow(ip string) bool {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	b := l.byIP[ip]
	if b == nil {
		l.byIP[ip] = &bucket{tokens: l.burst - 1, last: now}
		l.maybeGC(now)
		return true
	}

	// Refill tokens based on elapsed time.
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * l.rate
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now

	if b.tokens < 1 {
		l.maybeGC(now)
		return false
	}
	b.tokens -= 1
	l.maybeGC(now)
	return true
}

func (l *ipLimiter) maybeGC(now time.Time) {
	if l.ttl <= 0 || l.gcEvery <= 0 {
		return
	}
	if l.lastGC.IsZero() || now.Sub(l.lastGC) >= l.gcEvery {
		l.lastGC = now
		l.gc(now)
	}
}

func (l *ipLimiter) gc(now time.Time) {
	if l.ttl <= 0 {
		return
	}
	for ip, b := range l.byIP {
		if now.Sub(b.last) > l.ttl {
			delete(l.byIP, ip)
		}
	}
}
