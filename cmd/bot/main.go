// Package main is the entry point for the Telegram bot application
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"gobot/internal/bot"
	"gobot/internal/config"
	"gobot/internal/database"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Println("Configuration loaded successfully")

	// Initialize database
	if err := database.Initialize(cfg.DBPath, cfg.Debug); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Create initial data if needed
	if err := seedDatabase(); err != nil {
		log.Printf("Warning: Failed to seed database: %v", err)
	}

	// Create and start bot
	telegramBot, err := bot.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down bot...")
		telegramBot.Stop()
		os.Exit(0)
	}()

	// Start bot
	log.Println("Starting bot...")
	telegramBot.Start()
}

// seedDatabase creates initial services if they don't exist
func seedDatabase() error {
	var count int64
	database.DB.Model(&database.Service{}).Count(&count)

	if count > 0 {
		log.Println("Database already seeded")
		return nil
	}

	log.Println("Seeding database with initial services...")

	services := []database.Service{
		{
			Name:        "Классический массаж",
			Duration:    60,
			Price:       300000, // 3000 руб. в копейках
			Description: "Общеукрепляющий массаж всего тела",
			IsActive:    true,
		},
		{
			Name:        "Расслабляющий массаж",
			Duration:    90,
			Price:       400000, // 4000 руб.
			Description: "Глубокий расслабляющий массаж",
			IsActive:    true,
		},
		{
			Name:        "Спортивный массаж",
			Duration:    60,
			Price:       350000, // 3500 руб.
			Description: "Массаж для спортсменов и активных людей",
			IsActive:    true,
		},
		{
			Name:        "Депиляция ног",
			Duration:    45,
			Price:       250000, // 2500 руб.
			Description: "Полная депиляция ног",
			IsActive:    true,
		},
		{
			Name:        "Депиляция рук",
			Duration:    30,
			Price:       150000, // 1500 руб.
			Description: "Полная депиляция рук",
			IsActive:    true,
		},
		{
			Name:        "Депиляция зоны бикини",
			Duration:    30,
			Price:       200000, // 2000 руб.
			Description: "Депиляция классическая бикини",
			IsActive:    true,
		},
	}

	for _, service := range services {
		if err := database.DB.Create(&service).Error; err != nil {
			return err
		}
	}

	log.Println("Database seeded successfully")
	return nil
}

