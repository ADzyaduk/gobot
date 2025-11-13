// Package config handles application configuration loading and validation
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	BotToken     string
	AdminUserIDs []int64
	DBPath       string
	Timezone     string
	Debug        bool
	ChannelID    string // Optional: Telegram channel ID for promotions (format: @channelname or -1001234567890)
}

// Load reads configuration from environment variables
// It returns an error if required configuration is missing or invalid
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if file doesn't exist)
	_ = godotenv.Load()

	cfg := &Config{
		BotToken:  os.Getenv("BOT_TOKEN"),
		DBPath:    os.Getenv("DB_PATH"),
		Timezone:  os.Getenv("TIMEZONE"),
		Debug:     os.Getenv("BOT_DEBUG") == "true",
		ChannelID: os.Getenv("CHANNEL_ID"), // Optional channel for promotions
	}

	// Validate required fields
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN is required")
	}

	if cfg.DBPath == "" {
		cfg.DBPath = "./bot.db" // Default value
	}

	if cfg.Timezone == "" {
		cfg.Timezone = "UTC" // Default value
	}

	// Parse admin user IDs
	adminIDsStr := os.Getenv("ADMIN_USER_IDS")
	if adminIDsStr != "" {
		ids := strings.Split(adminIDsStr, ",")
		cfg.AdminUserIDs = make([]int64, 0, len(ids))
		for _, idStr := range ids {
			id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid admin user ID: %s: %w", idStr, err)
			}
			cfg.AdminUserIDs = append(cfg.AdminUserIDs, id)
		}
	}

	return cfg, nil
}

// IsAdmin checks if the given user ID is an admin
func (c *Config) IsAdmin(userID int64) bool {
	for _, adminID := range c.AdminUserIDs {
		if adminID == userID {
			return true
		}
	}
	return false
}
