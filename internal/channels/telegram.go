package channels

import (
	"context"
	"fmt"
	"strings"
	"strconv"

	"github.com/go-telegram/bot"

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
