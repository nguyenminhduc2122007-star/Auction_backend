package services

import (
	"errors"
	"strings"

	"auction-backend/internal/models"
	"auction-backend/internal/repository"

	"gorm.io/gorm"
)

const defaultOwnerID uint = 1

var (
	ErrItemNotFound = errors.New("item not found")
)

type ItemService struct {
	itemRepo *repository.ItemRepository
	userRepo *repository.UserRepository
}

type ItemListQuery struct {
	Q     string
	Page  int
	Limit int
}

type ItemListResponse struct {
	Items      []ItemListRow `json:"items"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	Limit      int           `json:"limit"`
	TotalPages int           `json:"total_pages"`
}

type ItemListRow struct {
	ID          uint    `json:"id"`
	Title       string  `json:"title"`
	Type        string  `json:"type"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}

type DashboardStats struct {
	TotalUsers     int64   `json:"total_users"`
	TotalAuctions  int64   `json:"total_auctions"`
	ActiveAuctions int64   `json:"active_auctions"`
	TotalRevenue   float64 `json:"total_revenue"`
}

type CreateItemInput struct {
	Title        string
	Type         string
	Status       string
	Description  string
	Price        float64
	StartPrice   float64
	CurrentPrice float64
}

func NewItemService(itemRepo *repository.ItemRepository, userRepo *repository.UserRepository) *ItemService {
	return &ItemService{itemRepo: itemRepo, userRepo: userRepo}
}

func (s *ItemService) CreateItem(input CreateItemInput) (*models.Item, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Type = strings.TrimSpace(input.Type)
	input.Status = strings.TrimSpace(input.Status)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return nil, errors.New("title is required")
	}
	// allow creating an item when start_price is provided even if price==0
	if input.Price <= 0 {
		if input.StartPrice > 0 {
			input.Price = input.StartPrice
		} else {
			return nil, errors.New("price must be greater than 0 or start_price must be provided")
		}
	}
	if input.Type == "" {
		input.Type = "Product"
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if input.StartPrice <= 0 {
		input.StartPrice = input.Price
	}
	if input.CurrentPrice <= 0 {
		input.CurrentPrice = input.Price
	}

	item := &models.Item{
		Title:        input.Title,
		Category:     input.Type,
		Status:       input.Status,
		Description:  input.Description,
		Price:        input.Price,
		StartPrice:   input.StartPrice,
		CurrentPrice: input.CurrentPrice,
		BidStep:      0,
		SellerID:     defaultOwnerID,
		UserID:       defaultOwnerID,
	}

	if err := s.itemRepo.Create(item); err != nil {
		return nil, errors.New("failed to create item")
	}

	return item, nil
}

// CreateItemForSeller creates an auction for a specific seller. The created
// auction will default to 'pending' status unless overridden.
func (s *ItemService) CreateItemForSeller(sellerID uint, input CreateItemInput) (*models.Item, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Type = strings.TrimSpace(input.Type)
	input.Status = strings.TrimSpace(input.Status)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return nil, errors.New("title is required")
	}
	// if start_price not supplied, derive it from price
	if input.StartPrice <= 0 {
		if input.Price > 0 {
			input.StartPrice = input.Price
		} else {
			return nil, errors.New("price or start_price must be greater than 0")
		}
	}
	if input.Type == "" {
		input.Type = "General"
	}
	if input.Status == "" {
		input.Status = "pending"
	}
	if input.CurrentPrice <= 0 {
		input.CurrentPrice = input.StartPrice
	}

	item := &models.Item{
		Title:        input.Title,
		Category:     input.Type,
		Status:       input.Status,
		Description:  input.Description,
		Price:        input.Price,
		StartPrice:   input.StartPrice,
		CurrentPrice: input.CurrentPrice,
		BidStep:      0,
		SellerID:     sellerID,
		UserID:       sellerID,
	}

	if err := s.itemRepo.Create(item); err != nil {
		return nil, errors.New("failed to create item")
	}

	return item, nil
}

func (s *ItemService) GetItemByID(id uint) (*models.Item, error) {
	item, err := s.itemRepo.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrItemNotFound
	}
	return item, err
}

func (s *ItemService) GetAllItems() ([]models.Item, error) {
	return s.itemRepo.GetAll()
}

func (s *ItemService) GetAllItemsBySeller(userID uint) ([]models.Item, error) {
	if userID == 0 {
		return nil, errors.New("user_id must be greater than 0")
	}
	return s.itemRepo.GetBySellerID(userID)
}

func (s *ItemService) ListItems(query ItemListQuery) (*ItemListResponse, error) {
	query.Q = strings.TrimSpace(query.Q)
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 {
		query.Limit = 10
	}
	if query.Limit > 100 {
		query.Limit = 100
	}

	items, total, err := s.itemRepo.List(query.Q, query.Page, query.Limit, "")
	if err != nil {
		return nil, err
	}

	rows := make([]ItemListRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, ItemListRow{
			ID:          item.ID,
			Title:       item.Title,
			Type:        item.Category,
			Price:       item.CurrentPrice,
			Description: item.Description,
		})
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(query.Limit) - 1) / int64(query.Limit))
	}

	return &ItemListResponse{
		Items:      rows,
		Total:      total,
		Page:       query.Page,
		Limit:      query.Limit,
		TotalPages: totalPages,
	}, nil
}

// AdminListItems lists items with optional status filter
func (s *ItemService) AdminListItems(query ItemListQuery, status string) (*ItemListResponse, error) {
	query.Q = strings.TrimSpace(query.Q)
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 {
		query.Limit = 10
	}
	if query.Limit > 100 {
		query.Limit = 100
	}

	items, total, err := s.itemRepo.List(query.Q, query.Page, query.Limit, status)
	if err != nil {
		return nil, err
	}

	rows := make([]ItemListRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, ItemListRow{
			ID:          item.ID,
			Title:       item.Title,
			Type:        item.Category,
			Price:       item.CurrentPrice,
			Description: item.Description,
		})
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(query.Limit) - 1) / int64(query.Limit))
	}

	return &ItemListResponse{
		Items:      rows,
		Total:      total,
		Page:       query.Page,
		Limit:      query.Limit,
		TotalPages: totalPages,
	}, nil
}

// UpdateItemBySeller updates an item when requested by its seller and only when there are no bids
func (s *ItemService) UpdateItemBySeller(sellerID, itemID uint, title, description string, price float64) (*models.Item, error) {
	item, err := s.GetItemByID(itemID)
	if err != nil {
		return nil, err
	}
	if item.SellerID != sellerID {
		return nil, errors.New("not authorized to update this item")
	}
	// simplistic "no bids" check: current price equals start price
	if item.CurrentPrice > item.StartPrice {
		return nil, errors.New("cannot update item: bids already placed")
	}
	if title == "" {
		return nil, errors.New("title is required")
	}
	if price <= 0 {
		return nil, errors.New("price must be greater than 0")
	}

	item.Title = title
	item.Description = description
	// Update price fields to reflect seller's requested price when no bids
	item.Price = price
	item.StartPrice = price
	item.CurrentPrice = price

	if err := s.itemRepo.Update(item); err != nil {
		return nil, errors.New("failed to update item")
	}
	return item, nil
}

// AdminUpdateItemStatus changes the status of an item (e.g., active, rejected, suspended)
func (s *ItemService) AdminUpdateItemStatus(itemID uint, status string) error {
	item, err := s.GetItemByID(itemID)
	if err != nil {
		return err
	}
	status = strings.TrimSpace(status)
	if status == "" {
		return errors.New("status is required")
	}
	// restrict to known actions
	allowed := map[string]bool{"active": true, "rejected": true, "suspended": true}
	if !allowed[status] {
		return errors.New("invalid status")
	}
	item.Status = status
	return s.itemRepo.Update(item)
}

func (s *ItemService) GetDashboardStats() (*DashboardStats, error) {
	totalUsers, err := s.userRepo.CountAll()
	if err != nil {
		return nil, err
	}

	totalAuctions, err := s.itemRepo.CountAll()
	if err != nil {
		return nil, err
	}

	activeAuctions, err := s.itemRepo.CountByStatus("active")
	if err != nil {
		return nil, err
	}

	totalRevenue, err := s.itemRepo.SumEndedRevenue()
	if err != nil {
		return nil, err
	}

	return &DashboardStats{
		TotalUsers:     totalUsers,
		TotalAuctions:  totalAuctions,
		ActiveAuctions: activeAuctions,
		TotalRevenue:   totalRevenue,
	}, nil
}

func (s *ItemService) UpdateItem(itemID uint, title, description string, price float64) (*models.Item, error) {
	item, err := s.GetItemByID(itemID)
	if err != nil {
		return nil, err
	}
	if title == "" {
		return nil, errors.New("title is required")
	}
	if price <= 0 {
		return nil, errors.New("price must be greater than 0")
	}

	item.Title = title
	item.Description = description
	// Update price fields to reflect the new price when admin updates
	item.Price = price
	if item.StartPrice == 0 {
		item.StartPrice = price
	} else {
		item.StartPrice = price
	}
	// Admin can also reset current price when explicitly updating price
	item.CurrentPrice = price

	if err := s.itemRepo.Update(item); err != nil {
		return nil, errors.New("failed to update item")
	}
	return item, nil
}

func (s *ItemService) DeleteItem(itemID uint) error {
	if _, err := s.GetItemByID(itemID); err != nil {
		return err
	}
	return s.itemRepo.Delete(itemID)
}
