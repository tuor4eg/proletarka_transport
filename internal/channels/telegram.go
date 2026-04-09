package channels

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-telegram/bot"

	"proletarka_transport/internal/config"
)

type TelegramChannel struct {
	bot    *bot.Bot
	chatID int64
}

func NewTelegramChannel(cfg config.TelegramConfig) (*TelegramChannel, error) {
	chatID, err := strconv.ParseInt(cfg.ChatID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid TELEGRAM_CHAT_ID: %w", err)
	}

	client, err := bot.New(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("create telegram bot client: %w", err)
	}

	return &TelegramChannel{
		bot:    client,
		chatID: chatID,
	}, nil
}

func (c *TelegramChannel) Name() string {
	return "telegram"
}

func (c *TelegramChannel) Send(ctx context.Context, message Message) error {
	_, err := c.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: c.chatID,
		Text:   message.Text,
	})
	if err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}

	return nil
}
