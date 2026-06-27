package services

import (
	"errors"

	"auction-backend/internal/models"
	"auction-backend/internal/repository"
	"auction-backend/pkg/utils"
)

type AuthService struct {
	userRepo *repository.UserRepository
}

func NewAuthService(userRepo *repository.UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

func (s *AuthService) Register(email, password, fullName string, userType models.UserType) (*models.User, error) {
	if !utils.ValidateEmail(email) {
		return nil, errors.New("invalid email format")
	}
	if !utils.ValidatePassword(password) {
		return nil, errors.New("password must be at least 8 characters")
	}
	if _, err := s.userRepo.GetByEmail(email); err == nil {
		return nil, errors.New("email already registered")
	}

	hash, err := utils.HashPassword(password)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	user := &models.User{
		Email:        email,
		FullName:     fullName,
		PasswordHash: hash,
		UserType:     userType,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, errors.New("failed to create user")
	}

	return user, nil
}

func (s *AuthService) Login(email, password string) (string, *models.User, error) {
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	if !utils.CheckPassword(password, user.PasswordHash) {
		return "", nil, errors.New("invalid credentials")
	}

	token, err := utils.GenerateToken(user.ID, string(user.UserType))
	if err != nil {
		return "", nil, errors.New("failed to generate token")
	}

	return token, user, nil
}
