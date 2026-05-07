package repository

import (
	"llm-proxy/internal/model"
	"log/slog"

	"gorm.io/gorm"
)

type ProviderRepository struct {
	db *gorm.DB
}

func NewProviderRepository(db *gorm.DB) *ProviderRepository {
	return &ProviderRepository{db: db}
}

// Create 创建Provider配置
func (r *ProviderRepository) Create(provider *model.ProviderConfig) error {
	return r.db.Create(provider).Error
}

// GetByID 根据ID获取Provider
func (r *ProviderRepository) GetByID(id uint) (*model.ProviderConfig, error) {
	var provider model.ProviderConfig
	err := r.db.First(&provider, id).Error
	if err != nil {
		slog.Error("根据ID获取Provider失败", "id", id, "error", err)
		return nil, err
	}
	return &provider, nil
}

// GetAll 获取所有Provider
func (r *ProviderRepository) GetAll() ([]model.ProviderConfig, error) {
	var providers []model.ProviderConfig
	err := r.db.Order("id desc").Find(&providers).Error
	return providers, err
}

// Update 更新Provider
func (r *ProviderRepository) Update(provider *model.ProviderConfig) error {
	return r.db.Save(provider).Error
}

// Delete 删除Provider（先将关联日志的ProviderID置为999，再删除Provider）
func (r *ProviderRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 将关联的请求日志的ProviderID置为999
		if err := tx.Model(&model.RequestLog{}).Where("provider_id = ?", id).Update("provider_id", 999).Error; err != nil {
			return err
		}
		return tx.Delete(&model.ProviderConfig{}, id).Error
	})
}

// DeleteAll 删除所有Provider配置
func (r *ProviderRepository) DeleteAll() error {
	return r.db.Where("1 = 1").Delete(&model.ProviderConfig{}).Error
}

// ImportAll 批量导入Provider配置
func (r *ProviderRepository) ImportAll(providers []model.ProviderConfig) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 将所有请求日志的ProviderID置为999
		if err := tx.Model(&model.RequestLog{}).Where("provider_id != ?", 999).Update("provider_id", 999).Error; err != nil {
			return err
		}
		// 清空现有数据
		if err := tx.Where("1 = 1").Delete(&model.ProviderConfig{}).Error; err != nil {
			return err
		}
		// 批量插入新数据
		if len(providers) > 0 {
			return tx.Create(&providers).Error
		}
		return nil
	})
}
