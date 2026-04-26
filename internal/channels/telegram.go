package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"proletarka_transport/internal/ai"
	"proletarka_transport/internal/backend"
	"proletarka_transport/internal/botmenu"
	"proletarka_transport/internal/config"
)

const importTopicsUnavailableMessage = "Не удалось получить список тем. Попробуйте позже."
const addPersonPromptMessage = "Пришлите одним сообщением описание человека: имя, годы жизни, биографию, связь с заводом и важные события. Я подготовлю черновик для проверки."
const personDraftAcceptedMessage = "Текст принят. Готовлю черновик, это может занять немного времени."
const personDraftInProgressMessage = "Текст уже отправлен на анализ. Дождитесь результата, пожалуйста."
const personDraftUnavailableMessage = "Не удалось подготовить черновик. Попробуйте позже."
const personDraftCallbackPrefix = "person_draft:"
const personDraftConfirmCallback = "person_draft:confirm"
const personDraftCancelCallback = "person_draft:cancel"
const personDraftConfirmPlaceholderMessage = "Подтверждение пока в подготовке. Черновик не отправлен в архив."
const personDraftCancelMessage = "Черновик отменён. В архив ничего не отправлено."

type ImportTopicsProvider interface {
	FetchImportTopics(ctx context.Context) ([]backend.ImportTopic, error)
}

type PersonDraftGenerator interface {
	Generate(ctx context.Context, req ai.Request) (ai.Response, error)
}

type pendingAction string

const (
	waitingPersonText     pendingAction = "waiting_person_text"
	waitingPersonAnalysis pendingAction = "waiting_person_analysis"
	waitingPersonConfirm  pendingAction = "waiting_person_confirm"
)

type TelegramChannel struct {
	bot                  *bot.Bot
	chatIDs              []int64
	menu                 *botmenu.Menu
	importTopicsProvider ImportTopicsProvider
	personDraftGenerator PersonDraftGenerator
	pendingMu            sync.Mutex
	pending              map[int64]pendingAction
}

func NewTelegramChannel(cfg config.TelegramConfig, importTopicsProvider ImportTopicsProvider, personDraftGenerator PersonDraftGenerator) (*TelegramChannel, error) {
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
		bot:                  client,
		chatIDs:              chatIDs,
		menu:                 botmenu.New(),
		importTopicsProvider: importTopicsProvider,
		personDraftGenerator: personDraftGenerator,
		pending:              make(map[int64]pendingAction),
	}, nil
}

func addPersonHandler(provider ImportTopicsProvider) botmenu.AddPersonAction {
	return func(ctx context.Context) (string, error) {
		if provider == nil {
			return "API основного backend не настроен. Список тем сейчас недоступен.", nil
		}

		topics, err := provider.FetchImportTopics(ctx)
		if err != nil {
			return importTopicsUnavailableMessage, nil
		}

		return backend.FormatImportTopics(topics), nil
	}
}

func (c *TelegramChannel) startAddPerson(chatID int64) string {
	c.setPendingAction(chatID, waitingPersonText)
	return addPersonPromptMessage
}

func (c *TelegramChannel) handlePendingPersonText(ctx context.Context, chatID int64, source string, logger *slog.Logger) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return addPersonPromptMessage
	}

	if !c.transitionPendingAction(chatID, waitingPersonText, waitingPersonAnalysis) {
		return personDraftInProgressMessage
	}

	if c.importTopicsProvider == nil {
		c.clearPendingAction(chatID)
		return "API основного backend не настроен. Список тем сейчас недоступен."
	}
	if c.personDraftGenerator == nil {
		c.clearPendingAction(chatID)
		return "AI-разбор не настроен. Черновик сейчас нельзя подготовить."
	}

	topics, err := c.importTopicsProvider.FetchImportTopics(ctx)
	if err != nil {
		logTelegramWarn(logger, "import topics fetch failed", chatID, err)
		c.clearPendingAction(chatID)
		return importTopicsUnavailableMessage
	}

	topicsJSON, err := json.Marshal(topics)
	if err != nil {
		logTelegramWarn(logger, "import topics marshal failed", chatID, err)
		c.clearPendingAction(chatID)
		return personDraftUnavailableMessage
	}

	response, err := c.personDraftGenerator.Generate(ctx, ai.Request{
		Task:  ai.TaskPersonDraft,
		Input: ai.BuildPersonDraftInput(topicsJSON, source),
	})
	if err != nil {
		logTelegramWarn(logger, "person draft generation failed", chatID, err)
		c.clearPendingAction(chatID)
		return personDraftUnavailableMessage
	}

	draft, err := ai.ParsePersonDraft(response.Text)
	if err != nil {
		logTelegramWarn(logger, "person draft parse failed", chatID, err)
		c.clearPendingAction(chatID)
		return personDraftUnavailableMessage
	}

	c.setPendingAction(chatID, waitingPersonConfirm)
	return ai.FormatPersonDraft(draft, importTopicTitles(topics))
}

func (c *TelegramChannel) confirmPersonDraft(chatID int64) string {
	c.clearPendingAction(chatID)
	return personDraftConfirmPlaceholderMessage
}

func (c *TelegramChannel) cancelPersonDraft(chatID int64) string {
	c.clearPendingAction(chatID)
	return personDraftCancelMessage
}

func logTelegramWarn(logger *slog.Logger, message string, chatID int64, err error) {
	if logger == nil || err == nil {
		return
	}

	logger.Warn(message, "chat_id", chatID, "error", err.Error())
}

func importTopicTitles(topics []backend.ImportTopic) map[string]string {
	titles := make(map[string]string)
	var walk func(items []backend.ImportTopic)
	walk = func(items []backend.ImportTopic) {
		for _, topic := range items {
			code := strings.TrimSpace(topic.Code)
			title := strings.TrimSpace(topic.Title)
			if code != "" && title != "" {
				titles[code] = title
			}
			if len(topic.Children) > 0 {
				walk(topic.Children)
			}
		}
	}
	walk(topics)

	return titles
}

func (c *TelegramChannel) setPendingAction(chatID int64, action pendingAction) {
	if c == nil || chatID == 0 {
		return
	}

	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()

	if c.pending == nil {
		c.pending = make(map[int64]pendingAction)
	}
	c.pending[chatID] = action
}

func (c *TelegramChannel) pendingAction(chatID int64) pendingAction {
	if c == nil || chatID == 0 {
		return ""
	}

	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()

	return c.pending[chatID]
}

func (c *TelegramChannel) transitionPendingAction(chatID int64, from pendingAction, to pendingAction) bool {
	if c == nil || chatID == 0 {
		return false
	}

	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()

	if c.pending[chatID] != from {
		return false
	}
	c.pending[chatID] = to
	return true
}

func (c *TelegramChannel) clearPendingAction(chatID int64) {
	if c == nil || chatID == 0 {
		return
	}

	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()

	delete(c.pending, chatID)
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

		c.clearPendingAction(update.Message.Chat.ID)
		c.sendRootMenu(ctx, update.Message.Chat.ID)
		logger.Info("telegram command handled", "command", "start", "user_id", update.Message.From.ID, "chat_id", update.Message.Chat.ID)
	})

	c.bot.RegisterHandler(bot.HandlerTypeMessageText, "ping", bot.MatchTypeCommand, func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if !c.isAllowedMessage(update) {
			c.reply(ctx, update.Message.Chat.ID, "Команда недоступна для этого аккаунта.")
			logger.Warn("telegram command rejected", "command", "ping", "user_id", update.Message.From.ID, "chat_id", update.Message.Chat.ID)
			return
		}

		c.clearPendingAction(update.Message.Chat.ID)
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

	c.bot.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return isPlainTextMessage(update)
	}, func(ctx context.Context, b *bot.Bot, update *models.Update) {
		text := strings.TrimSpace(update.Message.Text)
		var userID int64
		if update.Message.From != nil {
			userID = update.Message.From.ID
		}

		if !c.isAllowedMessage(update) {
			c.reply(ctx, update.Message.Chat.ID, "Действие недоступно для этого аккаунта.")
			logger.Warn("telegram message rejected", "text", text, "user_id", userID, "chat_id", update.Message.Chat.ID)
			return
		}

		switch c.pendingAction(update.Message.Chat.ID) {
		case waitingPersonText:
			if text != "" {
				c.reply(ctx, update.Message.Chat.ID, personDraftAcceptedMessage)
			}
			result := c.handlePendingPersonText(ctx, update.Message.Chat.ID, text, logger)
			if c.pendingAction(update.Message.Chat.ID) == waitingPersonConfirm {
				c.replyWithPersonDraftActions(ctx, update.Message.Chat.ID, result)
			} else {
				c.replyWithRootMenu(ctx, update.Message.Chat.ID, result)
			}
			logger.Info("telegram pending person text handled", "user_id", userID, "chat_id", update.Message.Chat.ID)
			return
		case waitingPersonAnalysis:
			c.reply(ctx, update.Message.Chat.ID, personDraftInProgressMessage)
			logger.Info("telegram pending person analysis ignored text", "user_id", userID, "chat_id", update.Message.Chat.ID)
			return
		}

		c.replyWithRootMenu(ctx, update.Message.Chat.ID, "Неизвестное действие. Используйте меню ниже.")
		logger.Info("telegram text not found", "text", text, "user_id", userID, "chat_id", update.Message.Chat.ID)
	})

	c.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, personDraftCallbackPrefix, bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *models.Update) {
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
			logger.Warn("telegram person draft callback rejected", "callback", update.CallbackQuery.Data, "user_id", update.CallbackQuery.From.ID, "chat_id", chatID)
			return
		}

		c.handlePersonDraftCallback(ctx, chatID, update.CallbackQuery.Data)
		logger.Info("telegram person draft callback handled", "callback", update.CallbackQuery.Data, "user_id", update.CallbackQuery.From.ID, "chat_id", chatID)
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

	c.bot.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return isCommandMessage(update)
	}, func(ctx context.Context, b *bot.Bot, update *models.Update) {
		text := strings.TrimSpace(update.Message.Text)
		var userID int64
		if update.Message.From != nil {
			userID = update.Message.From.ID
		}

		if !c.isAllowedMessage(update) {
			c.reply(ctx, update.Message.Chat.ID, "Команда недоступна для этого аккаунта.")
			logger.Warn("telegram command rejected", "command", text, "user_id", userID, "chat_id", update.Message.Chat.ID)
			return
		}

		c.clearPendingAction(update.Message.Chat.ID)
		c.replyWithRootMenu(ctx, update.Message.Chat.ID, "Неизвестная команда. Используйте /start или /ping.")
		logger.Info("telegram command not found", "command", text, "user_id", userID, "chat_id", update.Message.Chat.ID)
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
	if !isPlainText(text) {
		return false
	}

	_, ok := c.menu.FindRootTitle(text)
	return ok
}

func isPlainTextMessage(update *models.Update) bool {
	if update == nil || update.Message == nil {
		return false
	}

	return isPlainText(update.Message.Text)
}

func isCommandMessage(update *models.Update) bool {
	if update == nil || update.Message == nil {
		return false
	}

	return strings.HasPrefix(strings.TrimSpace(update.Message.Text), "/")
}

func isPlainText(text string) bool {
	text = strings.TrimSpace(text)
	return text != "" && !strings.HasPrefix(text, "/")
}

func (c *TelegramChannel) isAllowedCallback(update *models.Update) bool {
	if update == nil || update.CallbackQuery == nil {
		return false
	}

	return c.isAllowedUser(update.CallbackQuery.From.ID)
}

func (c *TelegramChannel) handlePersonDraftCallback(ctx context.Context, chatID int64, data string) {
	if chatID == 0 {
		return
	}

	if c.pendingAction(chatID) != waitingPersonConfirm {
		c.replyWithRootMenu(ctx, chatID, "Черновик уже не ожидает подтверждения.")
		return
	}

	switch data {
	case personDraftConfirmCallback:
		c.replyWithRootMenu(ctx, chatID, c.confirmPersonDraft(chatID))
	case personDraftCancelCallback:
		c.replyWithRootMenu(ctx, chatID, c.cancelPersonDraft(chatID))
	default:
		c.replyWithRootMenu(ctx, chatID, "Неизвестное действие с черновиком.")
	}
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

	if item != nil && item.ID == "add_person" {
		c.replyWithRootMenu(ctx, chatID, c.startAddPerson(chatID))
		return
	}

	c.clearPendingAction(chatID)
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

func (c *TelegramChannel) replyWithPersonDraftActions(ctx context.Context, chatID int64, text string) {
	_, _ = c.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: c.personDraftActionsKeyboard(),
	})
}

func (c *TelegramChannel) reply(ctx context.Context, chatID int64, text string) {
	_, _ = c.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})
}

func (c *TelegramChannel) personDraftActionsKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{
					Text:         "Подтвердить",
					CallbackData: personDraftConfirmCallback,
				},
				{
					Text:         "Отмена",
					CallbackData: personDraftCancelCallback,
				},
			},
		},
	}
}
