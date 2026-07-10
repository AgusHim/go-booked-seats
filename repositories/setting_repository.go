package repositories

import (
	"errors"
	"go-ticketing/models"

	"gorm.io/gorm"
)

type SettingRepository interface {
	Get(key string) (*models.Setting, error)
	Upsert(key string, value string) error
}

type settingRepository struct {
	db *gorm.DB
}

func NewSettingRepository(db *gorm.DB) SettingRepository {
	return &settingRepository{db}
}

func (r *settingRepository) Get(key string) (*models.Setting, error) {
	var setting models.Setting
	err := r.db.First(&setting, "key = ?", key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

func (r *settingRepository) Upsert(key string, value string) error {
	setting, err := r.Get(key)
	if err != nil {
		return err
	}
	if setting == nil {
		return r.db.Create(&models.Setting{Key: key, Value: value}).Error
	}
	return r.db.Model(&models.Setting{}).Where("key = ?", key).Update("value", value).Error
}
