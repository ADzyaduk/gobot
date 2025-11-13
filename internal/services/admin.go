// Package services contains business logic for admin operations
package services

import (
	"context"
	"fmt"

	"gobot/internal/database"
)

// AdminService handles admin-related operations
type AdminService struct{}

// NewAdminService creates a new admin service instance
func NewAdminService() *AdminService {
	return &AdminService{}
}

// GetAllBookings retrieves all bookings with pagination
func (s *AdminService) GetAllBookings(ctx context.Context, limit, offset int) ([]database.Booking, error) {
	var bookings []database.Booking
	err := database.DB.WithContext(ctx).
		Preload("Service").
		Preload("User").
		Order("date DESC, time DESC").
		Limit(limit).
		Offset(offset).
		Find(&bookings).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get bookings: %w", err)
	}

	return bookings, nil
}

// CreateService creates a new service
func (s *AdminService) CreateService(ctx context.Context, name, description string, duration, price int) (*database.Service, error) {
	service := &database.Service{
		Name:        name,
		Description: description,
		Duration:    duration,
		Price:       price,
		IsActive:    true,
	}

	if err := database.DB.WithContext(ctx).Create(service).Error; err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	return service, nil
}

// UpdateService updates an existing service
func (s *AdminService) UpdateService(ctx context.Context, serviceID uint, name, description string, duration, price int) error {
	updates := map[string]interface{}{
		"name":        name,
		"description": description,
		"duration":    duration,
		"price":       price,
	}

	result := database.DB.WithContext(ctx).
		Model(&database.Service{}).
		Where("id = ?", serviceID).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update service: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("service not found")
	}

	return nil
}

// UpdateServiceField updates a single field of a service
func (s *AdminService) UpdateServiceField(ctx context.Context, serviceID uint, field string, value interface{}) error {
	result := database.DB.WithContext(ctx).
		Model(&database.Service{}).
		Where("id = ?", serviceID).
		Update(field, value)

	if result.Error != nil {
		return fmt.Errorf("failed to update %s: %w", field, result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("service not found")
	}

	return nil
}

// ToggleServiceStatus activates or deactivates a service
func (s *AdminService) ToggleServiceStatus(ctx context.Context, serviceID uint) error {
	var service database.Service
	if err := database.DB.WithContext(ctx).First(&service, serviceID).Error; err != nil {
		return fmt.Errorf("service not found: %w", err)
	}

	service.IsActive = !service.IsActive
	if err := database.DB.WithContext(ctx).Save(&service).Error; err != nil {
		return fmt.Errorf("failed to toggle service status: %w", err)
	}

	return nil
}

// DeleteService soft deletes a service
func (s *AdminService) DeleteService(ctx context.Context, serviceID uint) error {
	result := database.DB.WithContext(ctx).Delete(&database.Service{}, serviceID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete service: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("service not found")
	}

	return nil
}

// GetAllServices retrieves all services including inactive ones
func (s *AdminService) GetAllServices(ctx context.Context) ([]database.Service, error) {
	var services []database.Service
	err := database.DB.WithContext(ctx).Find(&services).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}
	return services, nil
}

// GetServiceByID retrieves a service by ID
func (s *AdminService) GetServiceByID(ctx context.Context, serviceID uint) (*database.Service, error) {
	var service database.Service
	err := database.DB.WithContext(ctx).First(&service, serviceID).Error
	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}
	return &service, nil
}

// UpdateBookingStatus updates the status of a booking
func (s *AdminService) UpdateBookingStatus(ctx context.Context, bookingID uint, status database.BookingStatus) error {
	result := database.DB.WithContext(ctx).
		Model(&database.Booking{}).
		Where("id = ?", bookingID).
		Update("status", status)

	if result.Error != nil {
		return fmt.Errorf("failed to update booking status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("booking not found")
	}

	return nil
}

// GetStats retrieves system statistics
func (s *AdminService) GetStats(ctx context.Context) (map[string]int64, error) {
	stats := make(map[string]int64)

	var totalUsers int64
	var totalBookings int64
	var activeBookings int64
	var completedBookings int64
	var activeServices int64

	// Total users
	database.DB.Model(&database.User{}).Count(&totalUsers)
	stats["total_users"] = totalUsers

	// Total bookings
	database.DB.Model(&database.Booking{}).Count(&totalBookings)
	stats["total_bookings"] = totalBookings

	// Active bookings
	database.DB.Model(&database.Booking{}).
		Where("status IN ?", []database.BookingStatus{
			database.BookingStatusPending,
			database.BookingStatusConfirmed,
		}).Count(&activeBookings)
	stats["active_bookings"] = activeBookings

	// Completed bookings
	database.DB.Model(&database.Booking{}).
		Where("status = ?", database.BookingStatusCompleted).
		Count(&completedBookings)
	stats["completed_bookings"] = completedBookings

	// Total services
	database.DB.Model(&database.Service{}).Where("is_active = ?", true).Count(&activeServices)
	stats["active_services"] = activeServices

	return stats, nil
}
