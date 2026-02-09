package server

import (
	"image"
	"sync"
	"testing"
	"time"
)

func TestPNGCacheConcurrent(t *testing.T) {
	c := newPNGCache(1<<20, 50*time.Millisecond)
	if c == nil {
		t.Fatalf("cache is nil")
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "k" + string(rune('A'+(i%26)))
			c.Set(key, []byte("value"))
			_, _ = c.Get(key)
		}(i)
	}
	wg.Wait()
}

func TestLogoCacheConcurrent(t *testing.T) {
	c := newLogoCache(1 << 20)
	if c == nil {
		t.Fatalf("cache is nil")
	}
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "k" + string(rune('A'+(i%26)))
			c.Set(key, img)
			_, _ = c.Get(key)
		}(i)
	}
	wg.Wait()
}
