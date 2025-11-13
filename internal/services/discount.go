// Package services contains discount logic
package services

import (
	"context"
	"fmt"
	"time"

	"gobot/internal/database"
)

// DiscountService handles discount/promotion operations
type DiscountService struct{}

// NewDiscountService creates a new discount service
func NewDiscountService() *DiscountService {
	return &DiscountService{}
}

// CreateDiscount creates a new discount/promotion
func (s *DiscountService) CreateDiscount(ctx context.Context, serviceID uint, name string, percentage int, startDate, endDate time.Time) (*database.Discount, error) {
	discount := &database.Discount{
		ServiceID:  serviceID,
		Name:       name,
		Percentage: percentage,
		StartDate:  startDate,
		EndDate:    endDate,
		IsActive:   true,
	}

	if err := database.DB.WithContext(ctx).Create(discount).Error; err != nil {
		return nil, fmt.Errorf("failed to create discount: %w", err)
	}

	// Load relations
	if err := database.DB.WithContext(ctx).Preload("Service").First(discount, discount.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to load discount relations: %w", err)
	}

	return discount, nil
}

// GetActiveDiscounts retrieves all active discounts
func (s *DiscountService) GetActiveDiscounts(ctx context.Context) ([]database.Discount, error) {
	var discounts []database.Discount
	now := time.Now()

	err := database.DB.WithContext(ctx).
		Preload("Service").
		Where("is_active = ? AND start_date <= ? AND end_date >= ?", true, now, now).
		Find(&discounts).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get active discounts: %w", err)
	}

	return discounts, nil
}

// GetDiscountsByService retrieves discounts for a specific service
func (s *DiscountService) GetDiscountsByService(ctx context.Context, serviceID uint) ([]database.Discount, error) {
	var discounts []database.Discount

	err := database.DB.WithContext(ctx).
		Where("service_id = ?", serviceID).
		Order("created_at DESC").
		Find(&discounts).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get discounts: %w", err)
	}

	return discounts, nil
}

// GetServiceWithDiscount calculates the discounted price for a service
func (s *DiscountService) GetServiceWithDiscount(ctx context.Context, serviceID uint) (*database.Service, int, error) {
	var service database.Service
	if err := database.DB.WithContext(ctx).First(&service, serviceID).Error; err != nil {
		return nil, 0, fmt.Errorf("service not found: %w", err)
	}

	// Find active discount
	discounts, err := s.GetActiveDiscounts(ctx)
	if err != nil {
		return &service, service.Price, nil
	}

	for _, discount := range discounts {
		if discount.ServiceID == serviceID {
			discountedPrice := service.Price - (service.Price * discount.Percentage / 100)
			return &service, discountedPrice, nil
		}
	}

	return &service, service.Price, nil
}

// ToggleDiscountStatus activates or deactivates a discount
func (s *DiscountService) ToggleDiscountStatus(ctx context.Context, discountID uint) error {
	var discount database.Discount
	if err := database.DB.WithContext(ctx).First(&discount, discountID).Error; err != nil {
		return fmt.Errorf("discount not found: %w", err)
	}

	discount.IsActive = !discount.IsActive
	if err := database.DB.WithContext(ctx).Save(&discount).Error; err != nil {
		return fmt.Errorf("failed to toggle discount status: %w", err)
	}

	return nil
}

// DeleteDiscount deletes a discount
func (s *DiscountService) DeleteDiscount(ctx context.Context, discountID uint) error {
	result := database.DB.WithContext(ctx).Delete(&database.Discount{}, discountID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete discount: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("discount not found")
	}

	return nil
}

// GetAllDiscounts retrieves all discounts with their services
func (s *DiscountService) GetAllDiscounts(ctx context.Context) ([]database.Discount, error) {
	var discounts []database.Discount

	err := database.DB.WithContext(ctx).
		Preload("Service").
		Order("created_at DESC").
		Find(&discounts).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get discounts: %w", err)
	}

	return discounts, nil
}

// GetDiscountByID retrieves a discount by ID
func (s *DiscountService) GetDiscountByID(ctx context.Context, discountID uint) (*database.Discount, error) {
	var discount database.Discount
	err := database.DB.WithContext(ctx).
		Preload("Service").
		First(&discount, discountID).Error
	if err != nil {
		return nil, fmt.Errorf("discount not found: %w", err)
	}
	return &discount, nil
}
