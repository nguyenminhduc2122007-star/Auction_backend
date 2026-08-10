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
	// 1. Chuẩn hóa & Validate role đầu vào
	normalized := strings.TrimSpace(strings.ToLower(role))
	var targetRole models.UserType
	switch normalized {
	case "admin":
		targetRole = models.UserTypeAdmin
	case "seller":
		targetRole = models.UserTypeSeller
	case "bidder":
		targetRole = models.UserTypeBidder
	default:
		return nil, errors.New("vai trò không hợp lệ")
	}

	// 2. Lấy thông tin user hiện tại trong DB
	currentUser, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, errors.New("không tìm thấy người dùng")
	}

	// Nếu role không thay đổi, trả về kết quả luôn để tiết kiệm DB query
	if currentUser.UserType == targetRole {
		return currentUser, nil
	}

	// 3. Đếm số lượng Admin hiện tại trong hệ thống
	adminCount, err := s.userRepo.CountByRole(models.UserTypeAdmin)
	if err != nil {
		return nil, err
	}

	// RULE 1: Không cho phép bổ nhiệm thêm ADMIN nếu đã có ADMIN trong hệ thống
	if targetRole == models.UserTypeAdmin && currentUser.UserType != models.UserTypeAdmin {
		if adminCount >= 1 {
			return nil, errors.New("hệ thống đã có Admin, không thể bổ nhiệm thêm")
		}
	}

	// RULE 2: Không cho phép hạ cấp ADMIN duy nhất xuống Seller/Bidder
	if currentUser.UserType == models.UserTypeAdmin && targetRole != models.UserTypeAdmin {
		if adminCount <= 1 {
			return nil, errors.New("không thể hạ cấp Admin duy nhất của hệ thống")
		}
	}

	// 4. Gọi Repository để cập nhật role
	return s.userRepo.UpdateRole(id, targetRole)
}