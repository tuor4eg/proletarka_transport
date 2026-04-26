package channels

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"proletarka_transport/internal/botmenu"
	"proletarka_transport/internal/config"
)

type TelegramChannel struct {
	bot     *bot.Bot
	chatIDs []int64
	menu    *botmenu.Menu
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
		menu:    botmenu.New(),
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
			ChatID:      chatID,
			Text:        message.Text,
			ReplyMarkup: c.rootKeyboard(),
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
	c.bot.RegisterHandler(bot.HandlerTypeMessageText, "start", bot.MatchTypeCommand, func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if !c.isAllowedMessage(update) {
			c.reply(ctx, update.Message.Chat.ID, "Команда недоступна для этого аккаунта.")
			logger.Warn("telegram command rejected", "command", "start", "user_id", update.Message.From.ID, "chat_id", update.Message.Chat.ID)
			return
		}

		c.sendRootMenu(ctx, update.Message.Chat.ID)
		logger.Info("telegram command handled", "command", "start", "user_id", update.Message.From.ID, "chat_id", update.Message.Chat.ID)
	})

	c.bot.RegisterHandler(bot.HandlerTypeMessageText, "ping", bot.MatchTypeCommand, func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if !c.isAllowedMessage(update) {
			c.reply(ctx, update.Message.Chat.ID, "Команда недоступна для этого аккаунта.")
			logger.Warn("telegram command rejected", "command", "ping", "user_id", update.Message.From.ID, "chat_id", update.Message.Chat.ID)
			return
		}

		c.runMenuAction(ctx, update.Message.Chat.ID, "ping")
		logger.Info("telegram command handled", "command", "ping", "user_id", update.Message.From.ID, "chat_id", update.Message.Chat.ID)
	})

	c.bot.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return c.isRootMenuMessage(update)
	}, func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if !c.isAllowedMessage(update) {
			c.reply(ctx, update.Message.Chat.ID, "Команда недоступна для этого аккаунта.")
			logger.Warn("telegram root menu rejected", "text", update.Message.Text, "user_id", update.Message.From.ID, "chat_id", update.Message.Chat.ID)
			return
		}

		item, _ := c.menu.FindRootTitle(strings.TrimSpace(update.Message.Text))

		if item.IsAction() {
			c.runMenuItem(ctx, update.Message.Chat.ID, item)
			logger.Info("telegram root menu action handled", "item_id", item.ID, "user_id", update.Message.From.ID, "chat_id", update.Message.Chat.ID)
			return
		}

		c.sendMenu(ctx, update.Message.Chat.ID, item)
		logger.Info("telegram root menu submenu handled", "item_id", item.ID, "user_id", update.Message.From.ID, "chat_id", update.Message.Chat.ID)
	})

	c.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, botmenu.CallbackPrefix, bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.CallbackQuery == nil {
			return
		}

		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			ShowAlert:       false,
		})

		chatID := c.callbackChatID(update)
		if !c.isAllowedCallback(update) {
			if chatID != 0 {
				c.reply(ctx, chatID, "Команда недоступна для этого аккаунта.")
			}
			logger.Warn("telegram callback rejected", "callback", update.CallbackQuery.Data, "user_id", update.CallbackQuery.From.ID, "chat_id", chatID)
			return
		}

		item, ok := c.menu.FindCallback(update.CallbackQuery.Data)
		if !ok {
			if chatID != 0 {
				c.replyWithRootMenu(ctx, chatID, "Неизвестное действие меню.")
			}
			logger.Warn("telegram callback not found", "callback", update.CallbackQuery.Data, "user_id", update.CallbackQuery.From.ID, "chat_id", chatID)
			return
		}

		if item.IsAction() {
			c.runMenuItem(ctx, chatID, item)
			logger.Info("telegram callback action handled", "callback", update.CallbackQuery.Data, "user_id", update.CallbackQuery.From.ID, "chat_id", chatID)
			return
		}

		c.showCallbackMenu(ctx, update, item)
		logger.Info("telegram callback submenu handled", "callback", update.CallbackQuery.Data, "user_id", update.CallbackQuery.From.ID, "chat_id", chatID)
	})

	c.bot.RegisterHandler(bot.HandlerTypeMessageText, "", bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message == nil || !strings.HasPrefix(strings.TrimSpace(update.Message.Text), "/") {
			return
		}

		if !c.isAllowedMessage(update) {
			c.reply(ctx, update.Message.Chat.ID, "Команда недоступна для этого аккаунта.")
			logger.Warn("telegram command rejected", "command", update.Message.Text, "user_id", update.Message.From.ID, "chat_id", update.Message.Chat.ID)
			return
		}

		c.replyWithRootMenu(ctx, update.Message.Chat.ID, "Неизвестная команда. Используйте /start или /ping.")
		logger.Info("telegram command not found", "command", update.Message.Text, "user_id", update.Message.From.ID, "chat_id", update.Message.Chat.ID)
	})

	go c.bot.Start(ctx)
	logger.Info("telegram commands started", "allowed_chat_ids", len(c.chatIDs))
}

func (c *TelegramChannel) isAllowedMessage(update *models.Update) bool {
	if update == nil || update.Message == nil || update.Message.From == nil {
		return false
	}

	return c.isAllowedUser(update.Message.From.ID)
}

func (c *TelegramChannel) isRootMenuMessage(update *models.Update) bool {
	if update == nil || update.Message == nil {
		return false
	}

	text := strings.TrimSpace(update.Message.Text)
	if text == "" || strings.HasPrefix(text, "/") {
		return false
	}

	_, ok := c.menu.FindRootTitle(text)
	return ok
}

func (c *TelegramChannel) isAllowedCallback(update *models.Update) bool {
	if update == nil || update.CallbackQuery == nil {
		return false
	}

	return c.isAllowedUser(update.CallbackQuery.From.ID)
}

func (c *TelegramChannel) isAllowedUser(userID int64) bool {
	for _, chatID := range c.chatIDs {
		if chatID == userID {
			return true
		}
	}

	return false
}

func (c *TelegramChannel) runMenuAction(ctx context.Context, chatID int64, id string) {
	item, ok := c.menu.Find(id)
	if !ok {
		c.replyWithRootMenu(ctx, chatID, "Неизвестное действие меню.")
		return
	}

	c.runMenuItem(ctx, chatID, item)
}

func (c *TelegramChannel) runMenuItem(ctx context.Context, chatID int64, item *botmenu.Item) {
	if chatID == 0 {
		return
	}

	result, err := botmenu.Run(ctx, item)
	if err != nil {
		c.replyWithRootMenu(ctx, chatID, "Не удалось выполнить действие.")
		return
	}

	c.replyWithRootMenu(ctx, chatID, result)
}

func (c *TelegramChannel) sendRootMenu(ctx context.Context, chatID int64) {
	c.replyWithRootMenu(ctx, chatID, menuText(c.menu.Root()))
}

func (c *TelegramChannel) sendMenu(ctx context.Context, chatID int64, item *botmenu.Item) {
	replyMarkup := models.ReplyMarkup(c.inlineKeyboard(item))
	if item == nil || item.ID == "root" {
		replyMarkup = c.rootKeyboard()
	}

	_, _ = c.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        menuText(item),
		ReplyMarkup: replyMarkup,
	})
}

func (c *TelegramChannel) showCallbackMenu(ctx context.Context, update *models.Update, item *botmenu.Item) {
	if update.CallbackQuery.Message.Message != nil {
		_, _ = c.bot.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
			MessageID:   update.CallbackQuery.Message.Message.ID,
			Text:        menuText(item),
			ReplyMarkup: c.inlineKeyboard(item),
		})
		return
	}

	if chatID := c.callbackChatID(update); chatID != 0 {
		c.sendMenu(ctx, chatID, item)
	}
}

func (c *TelegramChannel) callbackChatID(update *models.Update) int64 {
	if update == nil || update.CallbackQuery == nil {
		return 0
	}
	if update.CallbackQuery.Message.Message != nil {
		return update.CallbackQuery.Message.Message.Chat.ID
	}

	return update.CallbackQuery.From.ID
}

func (c *TelegramChannel) inlineKeyboard(item *botmenu.Item) *models.InlineKeyboardMarkup {
	if item == nil || len(item.Children) == 0 {
		return nil
	}

	keyboard := make([][]models.InlineKeyboardButton, 0, len(item.Children))
	for _, child := range item.Children {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			{
				Text:         child.Title,
				CallbackData: botmenu.CallbackKey(child.ID),
			},
		})
	}
	if parent, ok := c.menu.Parent(item.ID); ok {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			{
				Text:         "Назад",
				CallbackData: botmenu.CallbackKey(parent.ID),
			},
		})
	}

	return &models.InlineKeyboardMarkup{InlineKeyboard: keyboard}
}

func (c *TelegramChannel) rootKeyboard() *models.ReplyKeyboardMarkup {
	if c == nil || c.menu == nil {
		return nil
	}

	root := c.menu.Root()
	if root == nil || len(root.Children) == 0 {
		return nil
	}

	keyboard := make([][]models.KeyboardButton, 0, len(root.Children))
	for _, child := range root.Children {
		keyboard = append(keyboard, []models.KeyboardButton{
			{Text: child.Title},
		})
	}

	return &models.ReplyKeyboardMarkup{
		Keyboard:       keyboard,
		IsPersistent:   true,
		ResizeKeyboard: true,
	}
}

func menuText(item *botmenu.Item) string {
	if item == nil || item.ID == "root" {
		return "Здравствуйте! Я transport-бот Proletarka.\n\nВыберите действие:"
	}

	return item.Title
}

func (c *TelegramChannel) replyWithRootMenu(ctx context.Context, chatID int64, text string) {
	_, _ = c.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: c.rootKeyboard(),
	})
}

func (c *TelegramChannel) reply(ctx context.Context, chatID int64, text string) {
	_, _ = c.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})
}
