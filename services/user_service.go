package services

import (
	"go-ticketing/models"
	"go-ticketing/repositories"
	"go-ticketing/utils"

	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	GetAll() ([]models.User, error)
	GetByID(id string) (*models.User, error)
	Create(user *models.User) error
	Update(id string, user *models.User) error
	Delete(id string) error
	Register(user *models.User) error
	Login(email, password string) (*models.User, string, error)
}

type userService struct {
	userRepo repositories.UserRepository
}

func NewUserService(userRepo repositories.UserRepository) UserService {
	return &userService{userRepo}
}

func (s *userService) Register(user *models.User) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hash)
	return s.userRepo.Create(user)
}

func (s *userService) Login(email, password string) (*models.User, string, error) {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return nil, "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, "", err
	}

	token, err := utils.GenerateJWT(user.ID)
	if err != nil {
		return nil, "", err
	}
	user.Password = ""
	return user, token, nil
}

func (s *userService) GetAll() ([]models.User, error) {
	return s.userRepo.FindAll()
}

func (s *userService) GetByID(id string) (*models.User, error) {
	return s.userRepo.FindByID(id)
}

func (s *userService) Create(user *models.User) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hash)
	return s.userRepo.Create(user)
}

func (s *userService) Update(id string, user *models.User) error {
	existing, err := s.userRepo.FindByID(id)
	if err != nil {
		return err
	}
	existing.Name = user.Name
	existing.Email = user.Email
	existing.Role = user.Role
	return s.userRepo.Update(existing)
}

func (s *userService) Delete(id string) error {
	return s.userRepo.Delete(id)
}
