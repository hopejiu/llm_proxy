package service

import (
	"github.com/wanglejiu/llm-proxy/internal/model"
	"github.com/wanglejiu/llm-proxy/internal/repository"
	"strings"
)

// CacheInvalidator 缓存失效接口，用于解耦 ProviderService 与 ProviderCache
type CacheInvalidator interface {
	Invalidate()
}

type ProviderService struct {
	repo     *repository.ProviderRepository
	cacheInv CacheInvalidator
}

func NewProviderService(repo *repository.ProviderRepository, cacheInv CacheInvalidator) *ProviderService {
	return &ProviderService{
		repo:     repo,
		cacheInv: cacheInv,
	}
}

// ProviderCacheInvalidator 使 ProviderCache 实现 CacheInvalidator 接口
func (c *ProviderCache) InvalidateCache() { c.Invalidate() }

// CreateProvider 创建Provider
func (s *ProviderService) CreateProvider(provider *model.ProviderConfig) error {
	err := s.repo.Create(provider)
	if err == nil && s.cacheInv != nil {
		s.cacheInv.Invalidate()
	}
	return err
}

// GetProvider 获取单个Provider
func (s *ProviderService) GetProvider(id uint) (*model.ProviderConfig, error) {
	return s.repo.GetByID(id)
}

// GetAllProviders 获取所有Provider
func (s *ProviderService) GetAllProviders() ([]model.ProviderConfig, error) {
	return s.repo.GetAll()
}

// GetProvidersByIDs 批量获取 Provider 名称（用于消除绑定服务层 N+1）
func (s *ProviderService) GetProvidersByIDs(ids []uint) (map[uint]string, error) {
	providers, err := s.repo.GetByIDs(ids)
	if err != nil {
		return nil, err
	}
	result := make(map[uint]string, len(providers))
	for id, p := range providers {
		result[id] = p.Name
	}
	return result, nil
}

// UpdateProvider 更新Provider
func (s *ProviderService) UpdateProvider(provider *model.ProviderConfig) error {
	err := s.repo.Update(provider)
	if err == nil && s.cacheInv != nil {
		s.cacheInv.Invalidate()
	}
	return err
}

// DeleteProvider 删除Provider
func (s *ProviderService) DeleteProvider(id uint) error {
	err := s.repo.Delete(id)
	if err == nil && s.cacheInv != nil {
		s.cacheInv.Invalidate()
	}
	return err
}

// ImportAll 批量导入Provider配置
func (s *ProviderService) ImportAll(providers []model.ProviderConfig) error {
	err := s.repo.ImportAll(providers)
	if err == nil && s.cacheInv != nil {
		s.cacheInv.Invalidate()
	}
	return err
}

// PreserveAPIKey 如果新 API Key 包含脱敏标记（****），保留原有密钥
func (s *ProviderService) PreserveAPIKey(id uint, newKey string) (string, error) {
	if !strings.Contains(newKey, "****") {
		return newKey, nil
	}
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return "", err
	}
	return existing.APIKey, nil
}
