package services

import (
	"errors"
	"strings"

	"auction-backend/internal/models"
	"auction-backend/internal/repository"
)

type UserService struct {
	userRepo *repository.UserRepository
}

type UserListRow struct {
	ID        uint            `json:"id"`
	Email     string          `json:"email"`
	FullName  string          `json:"full_name"`
	UserType  models.UserType `json:"user_type"`
	CreatedAt string          `json:"created_at"`
}

func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) ListUsers() ([]UserListRow, error) {
	users, err := s.userRepo.GetAll()
	if err != nil {
		return nil, err
	}

	rows := make([]UserListRow, 0, len(users))
	for _, user := range users {
		rows = append(rows, UserListRow{
			ID:        user.ID,
			Email:     user.Email,
			FullName:  user.FullName,
			UserType:  user.UserType,
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return rows, nil
}

func (s *UserService) UpdateUserRole(id uint, role string) (*models.User, error) {
	normalized := strings.TrimSpace(strings.ToLower(role))
	var userType models.UserType
	switch normalized {
	case "admin":
		userType = models.UserTypeAdmin
	case "seller":
		userType = models.UserTypeSeller
	case "bidder":
		userType = models.UserTypeBidder
	default:
		return nil, errors.New("invalid role")
	}

	return s.userRepo.UpdateRole(id, userType)
}
