package service

import (
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
)

type ProviderService struct {
	repo         *repository.ProviderRepository
	proxyService *ProxyService
}

func NewProviderService(repo *repository.ProviderRepository, proxyService *ProxyService) *ProviderService {
	return &ProviderService{repo: repo, proxyService: proxyService}
}

// CreateProvider 创建Provider
func (s *ProviderService) CreateProvider(provider *model.ProviderConfig) error {
	err := s.repo.Create(provider)
	if err == nil && s.proxyService != nil {
		s.proxyService.InvalidateCache()
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

// UpdateProvider 更新Provider
func (s *ProviderService) UpdateProvider(provider *model.ProviderConfig) error {
	err := s.repo.Update(provider)
	if err == nil && s.proxyService != nil {
		s.proxyService.InvalidateCache()
	}
	return err
}

// DeleteProvider 删除Provider
func (s *ProviderService) DeleteProvider(id uint) error {
	err := s.repo.Delete(id)
	if err == nil && s.proxyService != nil {
		s.proxyService.InvalidateCache()
	}
	return err
}

// ImportAll 批量导入Provider配置
func (s *ProviderService) ImportAll(providers []model.ProviderConfig) error {
	err := s.repo.ImportAll(providers)
	if err == nil && s.proxyService != nil {
		s.proxyService.InvalidateCache()
	}
	return err
}
