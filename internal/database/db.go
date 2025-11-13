// Package database handles database connections and initialization
package database

import (
	"fmt"
	"log"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	// Use pure-Go SQLite driver
	sqlite "github.com/glebarez/sqlite"
)

// DB is the global database instance
var DB *gorm.DB

// Initialize sets up the database connection and runs migrations
func Initialize(dbPath string, debug bool) error {
	var err error

	// Configure GORM logger
	var gormLogger logger.Interface
	if debug {
		gormLogger = logger.Default.LogMode(logger.Info)
	} else {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}

	// Open database connection
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run auto migrations
	if err := runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database initialized successfully")
	return nil
}

// runMigrations runs database migrations
func runMigrations() error {
	return DB.AutoMigrate(
		&User{},
		&Service{},
		&Booking{},
		&TimeSlot{},
		&Discount{},
		&WorkSchedule{},
		&BlockedDate{},
	)
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}

// Close closes the database connection
func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	return sqlDB.Close()
}
