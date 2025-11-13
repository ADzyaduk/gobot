// Package services contains business logic for the application
package services

import (
	"context"
	"fmt"
	"time"

	"gobot/internal/database"
)

// BookingService handles booking-related operations
type BookingService struct{}

// NewBookingService creates a new booking service instance
func NewBookingService() *BookingService {
	return &BookingService{}
}

// CreateBooking creates a new booking
func (s *BookingService) CreateBooking(ctx context.Context, userID int64, serviceID uint, date time.Time, timeSlot string) (*database.Booking, error) {
	booking := &database.Booking{
		UserID:    userID,
		ServiceID: serviceID,
		Date:      date,
		Time:      timeSlot,
		Status:    database.BookingStatusPending,
	}

	if err := database.DB.WithContext(ctx).Create(booking).Error; err != nil {
		return nil, fmt.Errorf("failed to create booking: %w", err)
	}

	// Load relations
	if err := database.DB.WithContext(ctx).Preload("Service").Preload("User").First(booking, booking.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to load booking relations: %w", err)
	}

	return booking, nil
}

// GetUserBookings retrieves all bookings for a user
func (s *BookingService) GetUserBookings(ctx context.Context, userID int64) ([]database.Booking, error) {
	var bookings []database.Booking
	err := database.DB.WithContext(ctx).
		Preload("Service").
		Where("user_id = ? AND status != ?", userID, database.BookingStatusCancelled).
		Order("date DESC, time DESC").
		Find(&bookings).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get user bookings: %w", err)
	}

	return bookings, nil
}

// CancelBooking cancels a booking
func (s *BookingService) CancelBooking(ctx context.Context, bookingID uint, userID int64) error {
	var booking database.Booking
	if err := database.DB.WithContext(ctx).First(&booking, bookingID).Error; err != nil {
		return fmt.Errorf("booking not found: %w", err)
	}

	if booking.UserID != userID {
		return fmt.Errorf("unauthorized: booking belongs to another user")
	}

	booking.Status = database.BookingStatusCancelled
	if err := database.DB.WithContext(ctx).Save(&booking).Error; err != nil {
		return fmt.Errorf("failed to cancel booking: %w", err)
	}

	return nil
}

// GetAvailableServices retrieves all active services
func (s *BookingService) GetAvailableServices(ctx context.Context) ([]database.Service, error) {
	var services []database.Service
	err := database.DB.WithContext(ctx).
		Where("is_active = ?", true).
		Find(&services).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}

	return services, nil
}
