package cache

import (
	"context"
	"sync"
	"time"

	"github.com/suguer/go-whois/internal/model"
)

// CacheManager 定义缓存管理接口
type CacheManager interface {
	// Get 获取缓存
	Get(ctx context.Context, key string) (*model.DomainInfo, error)

	// Set 设置缓存
	Set(ctx context.Context, key string, value *model.DomainInfo, ttl time.Duration) error

	// Delete 删除缓存
	Delete(ctx context.Context, key string) error

	// Clear 清空缓存
	Clear(ctx context.Context) error

	// Stats 获取缓存统计
	Stats() CacheStats
}

// CacheStats 缓存统计信息
type CacheStats struct {
	Hits    int64   `json:"hits"`
	Misses  int64   `json:"misses"`
	Size    int     `json:"size"`
	HitRate float64 `json:"hit_rate"`
}

// GenerateCacheKey 生成缓存键
func GenerateCacheKey(protocol, domain string) string {
	return protocol + ":" + domain
}

// MemoryCache 表示内存缓存
type MemoryCache struct {
	mu      sync.RWMutex
	items   map[string]*cacheItem
	maxSize int
	hits    int64
	misses  int64
}

type cacheItem struct {
	value     *model.DomainInfo
	expiresAt time.Time
}

// NewMemoryCache 创建新的内存缓存
func NewMemoryCache(maxSize int) *MemoryCache {
	cache := &MemoryCache{
		items:   make(map[string]*cacheItem),
		maxSize: maxSize,
	}

	// 启动清理协程
	go cache.cleanup()

	return cache
}

// Get 获取缓存
func (c *MemoryCache) Get(ctx context.Context, key string) (*model.DomainInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		c.misses++
		return nil, ErrCacheMiss
	}

	// 检查是否过期
	if time.Now().After(item.expiresAt) {
		c.misses++
		return nil, ErrCacheMiss
	}

	c.hits++
	return item.value, nil
}

// Set 设置缓存
func (c *MemoryCache) Set(ctx context.Context, key string, value *model.DomainInfo, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查缓存大小
	if len(c.items) >= c.maxSize {
		c.evict()
	}

	c.items[key] = &cacheItem{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

// Delete 删除缓存
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
	return nil
}

// Clear 清空缓存
func (c *MemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*cacheItem)
	return nil
}

// Stats 获取缓存统计
func (c *MemoryCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.hits + c.misses
	var hitRate float64
	if total > 0 {
		hitRate = float64(c.hits) / float64(total)
	}

	return CacheStats{
		Hits:    c.hits,
		Misses:  c.misses,
		Size:    len(c.items),
		HitRate: hitRate,
	}
}

// evict 淘汰最旧的缓存项
func (c *MemoryCache) evict() {
	var oldestKey string
	var oldestTime time.Time

	for key, item := range c.items {
		if oldestKey == "" || item.expiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.expiresAt
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}

// cleanup 定期清理过期缓存
func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.expiresAt) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// 错误定义
var (
	ErrCacheMiss = &CacheError{Message: "缓存未命中"}
)

// CacheError 表示缓存错误
type CacheError struct {
	Message string
}

// Error 实现 error 接口
func (e *CacheError) Error() string {
	return e.Message
}
