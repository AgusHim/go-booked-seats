package services

import "go-ticketing/repositories"

const darisiniCookieSettingKey = "darisini_cookie"

type SettingService interface {
	GetDarisiniCookie() (string, error)
	UpdateDarisiniCookie(cookie string) error
}

type settingService struct {
	repo repositories.SettingRepository
}

func NewSettingService(repo repositories.SettingRepository) SettingService {
	return &settingService{repo: repo}
}

func (s *settingService) GetDarisiniCookie() (string, error) {
	setting, err := s.repo.Get(darisiniCookieSettingKey)
	if err != nil || setting == nil {
		return "", err
	}
	return setting.Value, nil
}

func (s *settingService) UpdateDarisiniCookie(cookie string) error {
	return s.repo.Upsert(darisiniCookieSettingKey, cookie)
}
