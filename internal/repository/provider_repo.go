package repository

import (
	"log/slog"

	"github.com/wanglejiu/llm-proxy/internal/model"

	"gorm.io/gorm"
)

type ProviderRepository struct {
	dbManager *DBManager
}

func NewProviderRepository(dbManager *DBManager) *ProviderRepository {
	return &ProviderRepository{dbManager: dbManager}
}

// Create 创建Provider配置
func (r *ProviderRepository) Create(provider *model.ProviderConfig) error {
	return r.dbManager.GetDB().Create(provider).Error
}

// GetByID 根据ID获取Provider（记录不存在时返回"已删除"占位，不报错）
func (r *ProviderRepository) GetByID(id uint) (*model.ProviderConfig, error) {
	var provider model.ProviderConfig
	err := r.dbManager.GetDB().First(&provider, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &model.ProviderConfig{
				ID:   id,
				Name: "已删除",
			}, nil
		}
		slog.Error("根据ID获取Provider失败", "id", id, "error", err)
		return nil, err
	}
	return &provider, nil
}

// GetAll 获取所有Provider
func (r *ProviderRepository) GetAll() ([]model.ProviderConfig, error) {
	var providers []model.ProviderConfig
	err := r.dbManager.GetDB().Order("id desc").Find(&providers).Error
	return providers, err
}

// GetByIDs 批量获取 Provider（用于消除绑定服务层 N+1）
func (r *ProviderRepository) GetByIDs(ids []uint) (map[uint]model.ProviderConfig, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var providers []model.ProviderConfig
	if err := r.dbManager.GetDB().Where("id IN ?", ids).Find(&providers).Error; err != nil {
		return nil, err
	}
	result := make(map[uint]model.ProviderConfig, len(providers))
	for i := range providers {
		result[providers[i].ID] = providers[i]
	}
	return result, nil
}

// Update 更新Provider（只更新业务字段，不覆盖 created_at）
func (r *ProviderRepository) Update(provider *model.ProviderConfig) error {
	return r.dbManager.GetDB().Model(provider).Select("name", "auto_suffix", "url_suffix", "base_url", "api_key", "models", "enable_extra_params", "auto_fix_thinking", "updated_at").Updates(provider).Error
}

// Delete 删除Provider（先将关联日志的ProviderID置为DeletedProviderID，再删除Provider）
func (r *ProviderRepository) Delete(id uint) error {
	return r.dbManager.GetDB().Transaction(func(tx *gorm.DB) error {
		// 将关联的请求日志的ProviderID置为DeletedProviderID
		if err := tx.Model(&model.RequestLog{}).Where("provider_id = ?", id).Update("provider_id", model.DeletedProviderID).Error; err != nil {
			return err
		}
		return tx.Delete(&model.ProviderConfig{}, id).Error
	})
}

// ImportAll 批量导入Provider配置
func (r *ProviderRepository) ImportAll(providers []model.ProviderConfig) error {
	return r.dbManager.GetDB().Transaction(func(tx *gorm.DB) error {
		// 将所有请求日志的ProviderID置为DeletedProviderID
		if err := tx.Model(&model.RequestLog{}).Where("provider_id != ?", model.DeletedProviderID).Update("provider_id", model.DeletedProviderID).Error; err != nil {
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
