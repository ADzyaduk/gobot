// Package bot contains the Telegram bot implementation
package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"gobot/internal/config"
	"gobot/internal/database"
	"gobot/internal/services"

	tele "gopkg.in/telebot.v3"
)

// Bot represents the Telegram bot instance
type Bot struct {
	tg                  *tele.Bot
	config              *config.Config
	bookingService      *services.BookingService
	userService         *services.UserService
	adminService        *services.AdminService
	notificationService *services.NotificationService
	userStates          map[int64]*UserState
}

// UserState holds the current state of user interaction
type UserState struct {
	CurrentStep string
	ServiceID   uint
	Date        time.Time
	Time        string
	BookingID   uint

	// Admin editing states
	EditMode        string // "service_name", "service_price", etc.
	EditServiceID   uint
	TempServiceData map[string]interface{} // Temporary storage for editing
}

// New creates a new bot instance
func New(cfg *config.Config) (*Bot, error) {
	pref := tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	tg, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	bot := &Bot{
		tg:                  tg,
		config:              cfg,
		bookingService:      services.NewBookingService(),
		userService:         services.NewUserService(),
		adminService:        services.NewAdminService(),
		notificationService: services.NewNotificationService(tg, cfg.AdminUserIDs, cfg.ChannelID),
		userStates:          make(map[int64]*UserState),
	}

	bot.setupHandlers()

	// Start reminder worker in background
	go bot.notificationService.StartReminderWorker(context.Background())

	return bot, nil
}

// setupHandlers registers all command and callback handlers
func (b *Bot) setupHandlers() {
	// Command handlers
	b.tg.Handle("/start", b.handleStart)
	b.tg.Handle("/help", b.handleHelp)
	b.tg.Handle("/book", b.handleBook)
	b.tg.Handle("/my_bookings", b.handleMyBookings)
	b.tg.Handle("/cancel", b.handleCancelStart)
	b.tg.Handle("/admin", b.handleAdmin)

	// Callback handlers
	b.tg.Handle(tele.OnCallback, b.handleCallback)

	// Text message handler for admin edits
	b.tg.Handle(tele.OnText, b.handleTextInput)
}

// Start starts the bot
func (b *Bot) Start() {
	log.Println("Bot started successfully")
	b.tg.Start()
}

// Stop stops the bot gracefully
func (b *Bot) Stop() {
	b.tg.Stop()
	log.Println("Bot stopped")
}

// getUserState retrieves or creates user state
func (b *Bot) getUserState(userID int64) *UserState {
	if state, exists := b.userStates[userID]; exists {
		return state
	}
	state := &UserState{}
	b.userStates[userID] = state
	return state
}

// clearUserState clears user state
func (b *Bot) clearUserState(userID int64) {
	delete(b.userStates, userID)
}

// isAdmin checks if user is admin
func (b *Bot) isAdmin(userID int64) bool {
	return b.config.IsAdmin(userID)
}

// ensureUser ensures user exists in database
func (b *Bot) ensureUser(ctx context.Context, tgUser *tele.User) (*database.User, error) {
	return b.userService.GetOrCreateUser(ctx, tgUser)
}
