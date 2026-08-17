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

type UserListRowExtended struct {
	ID           uint            `json:"id"`
	Email        string          `json:"email"`
	FullName     string          `json:"full_name"`
	UserType     models.UserType `json:"user_type"`
	IsSuspicious bool            `json:"is_suspicious"`
	CreatedAt    string          `json:"created_at"`
}

type UserListFilterInput struct {
	Registered24h bool
	IsSuspicious  bool
	Search        string
}

type UserProfileResponse struct {
	ID                 uint            `json:"id"`
	FullName           string          `json:"full_name"`
	Email              string          `json:"email"`
	Phone              string          `json:"phone"`
	Address            string          `json:"address"`
	UserType           models.UserType `json:"user_type"`
	Balance            float64         `json:"balance"`
	ParticipatingCount int64           `json:"participating_count"`
	WonCount           int64           `json:"won_count"`
	MyAuctionCount     int64           `json:"my_auction_count"`
}

type UpdateProfileInput struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Address  string `json:"address"`
}

func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) ListUsersFiltered(input UserListFilterInput) ([]UserListRowExtended, error) {
	users, err := s.userRepo.ListUsersFiltered(input.Registered24h, input.IsSuspicious, input.Search)
	if err != nil {
		return nil, err
	}

	rows := make([]UserListRowExtended, 0, len(users))
	for _, user := range users {
		rows = append(rows, UserListRowExtended{
			ID:           user.ID,
			Email:        user.Email,
			FullName:     user.FullName,
			UserType:     user.UserType,
			IsSuspicious: user.IsSuspicious,
			CreatedAt:    user.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return rows, nil
}

func (s *UserService) GetUserSummary(userID uint) (*models.UserSummaryDTO, error) {
	return s.userRepo.GetUserSummary(userID)
}

func (s *UserService) UpdateUserRole(id uint, role string) (*models.User, error) {
	normalized := strings.TrimSpace(strings.ToLower(role))
	var targetRole models.UserType
	switch normalized {
	case "admin":
		targetRole = models.UserTypeAdmin
	case "seller":
		targetRole = models.UserTypeSeller
	case "bidder", "user":
		targetRole = models.UserTypeBidder
	default:
		return nil, errors.New("vai trò không hợp lệ")
	}

	currentUser, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, errors.New("không tìm thấy người dùng")
	}

	if currentUser.UserType == targetRole {
		return currentUser, nil
	}

	adminCount, err := s.userRepo.CountByRole(models.UserTypeAdmin)
	if err != nil {
		return nil, err
	}

	if targetRole == models.UserTypeAdmin && currentUser.UserType != models.UserTypeAdmin {
		if adminCount >= 1 {
			return nil, errors.New("hệ thống đã có Admin, không thể bổ nhiệm thêm")
		}
	}

	if currentUser.UserType == models.UserTypeAdmin && targetRole != models.UserTypeAdmin {
		if adminCount <= 1 {
			return nil, errors.New("không thể hạ cấp Admin duy nhất của hệ thống")
		}
	}

	return s.userRepo.UpdateRole(id, targetRole)
}

func (s *UserService) GetProfile(userID uint) (*UserProfileResponse, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	partCount, wonCount, myAuctionCount, err := s.userRepo.GetProfileStats(userID)
	if err != nil {
		return nil, err
	}

	return &UserProfileResponse{
		ID:                 user.ID,
		FullName:           user.FullName,
		Email:              user.Email,
		Phone:              user.Phone,
		Address:            user.ShippingAddress,
		UserType:           user.UserType,
		Balance:            user.WalletBalance,
		ParticipatingCount: partCount,
		WonCount:           wonCount,
		MyAuctionCount:     myAuctionCount,
	}, nil
}

func (s *UserService) UpdateProfile(userID uint, input UpdateProfileInput) (*models.User, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	if input.FullName != "" {
		user.FullName = input.FullName
	}
	user.Phone = input.Phone
	user.ShippingAddress = input.Address

	if err := s.userRepo.UpdateProfile(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) GetMyBids(userID uint) ([]models.UserBidItemDTO, error) {
	return s.userRepo.GetUserBids(userID)
}

func (s *UserService) GetWonAuctions(userID uint) ([]models.Auction, error) {
	return s.userRepo.GetWonAuctions(userID)
}
