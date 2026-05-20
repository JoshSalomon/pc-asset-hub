package export

import (
	"fmt"
	"sync"
	"time"
)

type PreviewCache interface {
	Store(token string, entry PreviewCacheEntry, ttl time.Duration) error
	Retrieve(token string) (*PreviewCacheEntry, error)
	Delete(token string)
}

type PreviewCacheEntry struct {
	CatalogName      string
	Artifacts        map[string][]K8sArtifact
	BindingResults   []BindingRunResult
	CatalogUpdatedAt time.Time
	ExpiresAt        time.Time
}

type InMemoryPreviewCache struct {
	mu              sync.RWMutex
	cache           map[string]*PreviewCacheEntry
	cleanupInterval time.Duration
	stopCh          chan struct{}
}

func NewInMemoryPreviewCache() *InMemoryPreviewCache {
	return NewInMemoryPreviewCacheWithInterval(1 * time.Minute)
}

func NewInMemoryPreviewCacheWithInterval(interval time.Duration) *InMemoryPreviewCache {
	c := &InMemoryPreviewCache{
		cache:           make(map[string]*PreviewCacheEntry),
		cleanupInterval: interval,
		stopCh:          make(chan struct{}),
	}
	go c.cleanupLoop()
	return c
}

func (c *InMemoryPreviewCache) Stop() {
	close(c.stopCh)
}

func (c *InMemoryPreviewCache) Store(token string, entry PreviewCacheEntry, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry.ExpiresAt = time.Now().Add(ttl)
	c.cache[token] = &entry
	return nil
}

func (c *InMemoryPreviewCache) Retrieve(token string) (*PreviewCacheEntry, error) {
	c.mu.RLock()
	entry, ok := c.cache[token]
	if !ok {
		c.mu.RUnlock()
		return nil, fmt.Errorf("preview session not found or expired")
	}
	expired := time.Now().After(entry.ExpiresAt)
	result := *entry
	c.mu.RUnlock()
	if expired {
		c.Delete(token)
		return nil, fmt.Errorf("preview session expired")
	}
	return &result, nil
}

func (c *InMemoryPreviewCache) Delete(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, token)
}

func (c *InMemoryPreviewCache) cleanupLoop() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
		}
		c.mu.Lock()
		now := time.Now()
		for token, entry := range c.cache {
			if now.After(entry.ExpiresAt) {
				delete(c.cache, token)
			}
		}
		c.mu.Unlock()
	}
}
