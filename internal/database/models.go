// Package database contains database models and operations
package database

import (
	"time"

	"gorm.io/gorm"
)

// User represents a Telegram user in the system
type User struct {
	ID        int64  `gorm:"primaryKey"` // Telegram User ID
	Username  string `gorm:"index"`
	FirstName string
	LastName  string
	IsAdmin   bool `gorm:"default:false"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Relations
	Bookings []Booking `gorm:"foreignKey:UserID"`
}

// Service represents a service type (massage or depilation)
type Service struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"not null;index"`
	Duration    int    `gorm:"not null"` // Duration in minutes
	Price       int    `gorm:"not null"` // Price in cents or smallest currency unit
	Description string
	IsActive    bool `gorm:"default:true;index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	// Relations
	Bookings []Booking `gorm:"foreignKey:ServiceID"`
}

// BookingStatus represents the status of a booking
type BookingStatus string

const (
	BookingStatusPending   BookingStatus = "pending"
	BookingStatusConfirmed BookingStatus = "confirmed"
	BookingStatusCancelled BookingStatus = "cancelled"
	BookingStatusCompleted BookingStatus = "completed"
)

// Booking represents a service booking
type Booking struct {
	ID        uint          `gorm:"primaryKey"`
	UserID    int64         `gorm:"not null;index"`
	ServiceID uint          `gorm:"not null;index"`
	Date      time.Time     `gorm:"not null;index"`
	Time      string        `gorm:"not null"` // Format: "HH:MM"
	Status    BookingStatus `gorm:"not null;index;default:'pending'"`
	Notes     string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Relations
	User    User    `gorm:"foreignKey:UserID"`
	Service Service `gorm:"foreignKey:ServiceID"`
}

// TimeSlot represents an available time slot
type TimeSlot struct {
	ID          uint      `gorm:"primaryKey"`
	Date        time.Time `gorm:"not null;index"`
	Time        string    `gorm:"not null"` // Format: "HH:MM"
	IsAvailable bool      `gorm:"default:true;index"`
	BookingID   *uint     `gorm:"index"` // nullable
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	// Relations
	Booking *Booking `gorm:"foreignKey:BookingID"`
}

// Discount represents a discount/promotion for services
type Discount struct {
	ID         uint      `gorm:"primaryKey"`
	ServiceID  uint      `gorm:"not null;index"`
	Name       string    `gorm:"not null"` // e.g., "Summer Sale"
	Percentage int       `gorm:"not null"` // e.g., 20 for 20% off
	StartDate  time.Time `gorm:"not null;index"`
	EndDate    time.Time `gorm:"not null;index"`
	IsActive   bool      `gorm:"default:true;index"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`

	// Relations
	Service Service `gorm:"foreignKey:ServiceID"`
}

// WorkSchedule represents working hours configuration
type WorkSchedule struct {
	ID        uint   `gorm:"primaryKey"`
	DayOfWeek int    `gorm:"not null;index"` // 0=Sunday, 1=Monday, etc.
	StartTime string `gorm:"not null"`       // Format: "HH:MM"
	EndTime   string `gorm:"not null"`       // Format: "HH:MM"
	IsActive  bool   `gorm:"default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// BlockedDate represents dates when bookings are not allowed
type BlockedDate struct {
	ID        uint      `gorm:"primaryKey"`
	Date      time.Time `gorm:"not null;index"`
	Reason    string    // e.g., "Holiday", "Closed"
	CreatedAt time.Time
	UpdatedAt time.Time
}
