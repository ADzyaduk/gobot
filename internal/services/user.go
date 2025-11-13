// Package services contains business logic for the application
package services

import (
	"context"
	"fmt"

	"gobot/internal/database"

	tele "gopkg.in/telebot.v3"
)

// UserService handles user-related operations
type UserService struct{}

// NewUserService creates a new user service instance
func NewUserService() *UserService {
	return &UserService{}
}

// GetOrCreateUser retrieves or creates a user from Telegram user data
func (s *UserService) GetOrCreateUser(ctx context.Context, tgUser *tele.User) (*database.User, error) {
	var user database.User

	// Try to find existing user
	err := database.DB.WithContext(ctx).Where("id = ?", tgUser.ID).First(&user).Error
	if err == nil {
		// User exists, update info
		user.Username = tgUser.Username
		user.FirstName = tgUser.FirstName
		user.LastName = tgUser.LastName
		if err := database.DB.WithContext(ctx).Save(&user).Error; err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
		return &user, nil
	}

	// Create new user
	user = database.User{
		ID:        tgUser.ID,
		Username:  tgUser.Username,
		FirstName: tgUser.FirstName,
		LastName:  tgUser.LastName,
		IsAdmin:   false,
	}

	if err := database.DB.WithContext(ctx).Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

// IsAdmin checks if a user is an admin
func (s *UserService) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	var user database.User
	err := database.DB.WithContext(ctx).Where("id = ?", userID).First(&user).Error
	if err != nil {
		return false, fmt.Errorf("failed to get user: %w", err)
	}
	return user.IsAdmin, nil
}

// SetAdmin sets or removes admin status for a user
func (s *UserService) SetAdmin(ctx context.Context, userID int64, isAdmin bool) error {
	result := database.DB.WithContext(ctx).Model(&database.User{}).
		Where("id = ?", userID).
		Update("is_admin", isAdmin)

	if result.Error != nil {
		return fmt.Errorf("failed to update admin status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

