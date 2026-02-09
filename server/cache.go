package server

import (
	"container/list"
	"image"
	"sync"
	"time"
)

// pngCache is safe for concurrent use via its internal mutex.
type pngCache struct {
	mu       sync.Mutex
	maxBytes int64
	curBytes int64
	ttl      time.Duration
	ll       *list.List
	items    map[string]*list.Element
}

type pngEntry struct {
	key     string
	value   []byte
	size    int64
	expires time.Time
}

func newPNGCache(maxBytes int64, ttl time.Duration) *pngCache {
	if maxBytes <= 0 {
		return &pngCache{maxBytes: 0}
	}
	return &pngCache{
		maxBytes: maxBytes,
		ttl:      ttl,
		ll:       list.New(),
		items:    make(map[string]*list.Element),
	}
}

func (c *pngCache) Get(key string) ([]byte, bool) {
	if c == nil || c.maxBytes <= 0 {
		return nil, false
	}
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if ele, ok := c.items[key]; ok {
		ent := ele.Value.(*pngEntry)
		if c.ttl > 0 && now.After(ent.expires) {
			c.removeElement(ele)
			return nil, false
		}
		c.ll.MoveToFront(ele)
		return ent.value, true
	}
	return nil, false
}

func (c *pngCache) Set(key string, value []byte) {
	if c == nil || c.maxBytes <= 0 {
		return
	}
	size := int64(len(value))
	if size <= 0 || size > c.maxBytes {
		return
	}
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()

	if ele, ok := c.items[key]; ok {
		ent := ele.Value.(*pngEntry)
		c.curBytes -= ent.size
		ent.value = value
		ent.size = size
		if c.ttl > 0 {
			ent.expires = now.Add(c.ttl)
		}
		c.curBytes += size
		c.ll.MoveToFront(ele)
		c.evict()
		return
	}

	ent := &pngEntry{key: key, value: value, size: size}
	if c.ttl > 0 {
		ent.expires = now.Add(c.ttl)
	}
	ele := c.ll.PushFront(ent)
	c.items[key] = ele
	c.curBytes += size
	c.evict()
}

func (c *pngCache) evict() {
	for c.curBytes > c.maxBytes {
		ele := c.ll.Back()
		if ele == nil {
			return
		}
		c.removeElement(ele)
	}
}

func (c *pngCache) removeElement(ele *list.Element) {
	ent := ele.Value.(*pngEntry)
	delete(c.items, ent.key)
	c.curBytes -= ent.size
	c.ll.Remove(ele)
}

// logoCache is safe for concurrent use via its internal mutex.
type logoCache struct {
	mu       sync.Mutex
	maxBytes int64
	curBytes int64
	ll       *list.List
	items    map[string]*list.Element
}

type logoEntry struct {
	key   string
	value image.Image
	size  int64
}

func newLogoCache(maxBytes int64) *logoCache {
	if maxBytes <= 0 {
		return &logoCache{maxBytes: 0}
	}
	return &logoCache{
		maxBytes: maxBytes,
		ll:       list.New(),
		items:    make(map[string]*list.Element),
	}
}

func (c *logoCache) Get(key string) (image.Image, bool) {
	if c == nil || c.maxBytes <= 0 {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if ele, ok := c.items[key]; ok {
		c.ll.MoveToFront(ele)
		return ele.Value.(*logoEntry).value, true
	}
	return nil, false
}

func (c *logoCache) Set(key string, value image.Image) {
	if c == nil || c.maxBytes <= 0 || value == nil {
		return
	}
	size := estimateImageBytes(value)
	if size <= 0 || size > c.maxBytes {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if ele, ok := c.items[key]; ok {
		ent := ele.Value.(*logoEntry)
		c.curBytes -= ent.size
		ent.value = value
		ent.size = size
		c.curBytes += size
		c.ll.MoveToFront(ele)
		c.evict()
		return
	}

	ent := &logoEntry{key: key, value: value, size: size}
	ele := c.ll.PushFront(ent)
	c.items[key] = ele
	c.curBytes += size
	c.evict()
}

func (c *logoCache) evict() {
	for c.curBytes > c.maxBytes {
		ele := c.ll.Back()
		if ele == nil {
			return
		}
		c.removeElement(ele)
	}
}

func (c *logoCache) removeElement(ele *list.Element) {
	ent := ele.Value.(*logoEntry)
	delete(c.items, ent.key)
	c.curBytes -= ent.size
	c.ll.Remove(ele)
}

func estimateImageBytes(img image.Image) int64 {
	b := img.Bounds()
	if b.Empty() {
		return 0
	}
	return int64(b.Dx()*b.Dy()) * 4
}
