package service

import (
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
)

type ProviderService struct {
	repo *repository.ProviderRepository
}

func NewProviderService(repo *repository.ProviderRepository) *ProviderService {
	return &ProviderService{repo: repo}
}

// CreateProvider 创建Provider
func (s *ProviderService) CreateProvider(provider *model.ProviderConfig) error {
	return s.repo.Create(provider)
}

// GetProvider 获取单个Provider
func (s *ProviderService) GetProvider(id uint) (*model.ProviderConfig, error) {
	return s.repo.GetByID(id)
}

// GetAllProviders 获取所有Provider
func (s *ProviderService) GetAllProviders() ([]model.ProviderConfig, error) {
	return s.repo.GetAll()
}

// GetActiveProviders 获取所有启用的Provider
func (s *ProviderService) GetActiveProviders() ([]model.ProviderConfig, error) {
	return s.repo.GetActive()
}

// UpdateProvider 更新Provider
func (s *ProviderService) UpdateProvider(provider *model.ProviderConfig) error {
	return s.repo.Update(provider)
}

// DeleteProvider 删除Provider
func (s *ProviderService) DeleteProvider(id uint) error {
	return s.repo.Delete(id)
}

// ToggleProviderStatus 切换Provider状态
func (s *ProviderService) ToggleProviderStatus(id uint, isActive bool) error {
	return s.repo.ToggleActive(id, isActive)
}
