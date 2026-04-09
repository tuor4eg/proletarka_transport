package channels

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"proletarka_transport/internal/config"
)

type TelegramChannel struct {
	bot     *bot.Bot
	chatIDs []int64
}

func NewTelegramChannel(cfg config.TelegramConfig) (*TelegramChannel, error) {
	chatIDs := make([]int64, 0, len(cfg.ChatIDs))
	for _, rawChatID := range cfg.ChatIDs {
		chatID, err := strconv.ParseInt(rawChatID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid TELEGRAM_CHAT_IDS value %q: %w", rawChatID, err)
		}
		chatIDs = append(chatIDs, chatID)
	}

	client, err := bot.New(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("create telegram bot client: %w", err)
	}

	return &TelegramChannel{
		bot:     client,
		chatIDs: chatIDs,
	}, nil
}

func (c *TelegramChannel) Name() string {
	return "telegram"
}

func (c *TelegramChannel) Send(ctx context.Context, message Message) error {
	var delivered int
	var failures []string

	for _, chatID := range c.chatIDs {
		_, err := c.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   message.Text,
		})
		if err != nil {
			failures = append(failures, fmt.Sprintf("%d: %v", chatID, err))
			continue
		}

		delivered++
	}

	if delivered > 0 {
		return nil
	}

	return fmt.Errorf("send telegram message to all chats failed: %s", strings.Join(failures, "; "))
}

func (c *TelegramChannel) StartCommands(ctx context.Context, logger *slog.Logger) {
	c.bot.RegisterHandler(bot.HandlerTypeMessageText, "ping", bot.MatchTypeCommand, func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if !c.isAllowed(update) {
			c.reply(ctx, update.Message.Chat.ID, "command is not available for this account")
			logger.Warn("telegram command rejected", "command", "ping", "user_id", update.Message.From.ID, "chat_id", update.Message.Chat.ID)
			return
		}

		c.reply(ctx, update.Message.Chat.ID, "pong")
		logger.Info("telegram command handled", "command", "ping", "user_id", update.Message.From.ID, "chat_id", update.Message.Chat.ID)
	})

	c.bot.RegisterHandler(bot.HandlerTypeMessageText, "", bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message == nil || !strings.HasPrefix(strings.TrimSpace(update.Message.Text), "/") {
			return
		}

		if !c.isAllowed(update) {
			c.reply(ctx, update.Message.Chat.ID, "command is not available for this account")
			logger.Warn("telegram command rejected", "command", update.Message.Text, "user_id", update.Message.From.ID, "chat_id", update.Message.Chat.ID)
			return
		}

		c.reply(ctx, update.Message.Chat.ID, "unknown command")
		logger.Info("telegram command not found", "command", update.Message.Text, "user_id", update.Message.From.ID, "chat_id", update.Message.Chat.ID)
	})

	go c.bot.Start(ctx)
	logger.Info("telegram commands started", "allowed_chat_ids", len(c.chatIDs))
}

func (c *TelegramChannel) isAllowed(update *models.Update) bool {
	if update == nil || update.Message == nil || update.Message.From == nil {
		return false
	}

	userID := update.Message.From.ID
	for _, chatID := range c.chatIDs {
		if chatID == userID {
			return true
		}
	}

	return false
}

func (c *TelegramChannel) reply(ctx context.Context, chatID int64, text string) {
	_, _ = c.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})
}
