package service

import (
	"sync"
	"time"

	"github.com/wanglejiu/llm-proxy/internal/model"
	"github.com/wanglejiu/llm-proxy/internal/repository"
)

// CacheTTLProvider 缓存 TTL 提供者接口，使 ProviderCache 不依赖 config.Config
type CacheTTLProvider interface {
	GetProviderCacheTTL() time.Duration
}

// ProviderCache 线程安全的 Provider 列表缓存，带 TTL 过期和双重检查锁定
type ProviderCache struct {
	repo          *repository.ProviderRepository
	ttlProvider   CacheTTLProvider
	mu            sync.RWMutex
	providerCache []model.ProviderConfig
	cacheExpiry   time.Time
}

// NewProviderCache 创建 ProviderCache
func NewProviderCache(repo *repository.ProviderRepository, ttlProvider CacheTTLProvider) *ProviderCache {
	return &ProviderCache{
		repo:        repo,
		ttlProvider: ttlProvider,
	}
}

// GetAll 获取所有 Provider（优先读缓存，缓存过期时自动刷新）
// 深拷贝返回，防止调用者修改影响缓存
func (c *ProviderCache) GetAll() ([]model.ProviderConfig, error) {
	// 快速路径：读缓存
	c.mu.RLock()
	if time.Now().Before(c.cacheExpiry) {
		result := make([]model.ProviderConfig, len(c.providerCache))
		copy(result, c.providerCache)
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	// 缓存过期，加写锁
	c.mu.Lock()
	defer c.mu.Unlock()

	// 双重检查：另一个 goroutine 可能已经刷新了缓存
	if time.Now().Before(c.cacheExpiry) {
		result := make([]model.ProviderConfig, len(c.providerCache))
		copy(result, c.providerCache)
		return result, nil
	}

	providers, err := c.repo.GetAll()
	if err != nil {
		return nil, err
	}

	c.providerCache = providers
	c.cacheExpiry = time.Now().Add(c.ttlProvider.GetProviderCacheTTL())

	result := make([]model.ProviderConfig, len(providers))
	copy(result, providers)
	return result, nil
}

// Invalidate 主动失效缓存
func (c *ProviderCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cacheExpiry = time.Time{}
}
