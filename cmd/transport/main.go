package main

import (
	"context"
	"log"
	"net/http"

	"proletarka_transport/internal/ai"
	"proletarka_transport/internal/backend"
	"proletarka_transport/internal/channels"
	"proletarka_transport/internal/config"
	"proletarka_transport/internal/events"
	httptransport "proletarka_transport/internal/http"
	"proletarka_transport/internal/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	appLogger := logger.New()
	var importTopicsProvider channels.ImportTopicsProvider
	if cfg.API.Enabled {
		client, err := backend.NewClient(cfg.API, nil)
		if err != nil {
			log.Fatalf("api config error: %v", err)
		}
		importTopicsProvider = client
	}

	var personDraftGenerator channels.PersonDraftGenerator
	if cfg.AI.Enabled {
		models := make([]ai.ModelConfig, 0, len(cfg.AI.Models))
		for _, model := range cfg.AI.Models {
			models = append(models, ai.ModelConfig{
				ID:            model.ID,
				Provider:      model.Provider,
				Name:          model.Name,
				APIKey:        model.APIKey,
				BaseURL:       model.BaseURL,
				Timeout:       model.Timeout,
				MaxInputChars: model.MaxInputChars,
			})
		}
		personDraftGenerator = ai.NewService(
			cfg.AI.Enabled,
			ai.NewRouter(cfg.AI.Enabled, cfg.AI.DefaultModel, models),
			ai.NewTemplatePrompter(ai.DefaultPromptTemplates()),
			ai.NewHTTPTransport(nil),
		)
	}

	var telegramChannel channels.Channel
	var telegramBot *channels.TelegramChannel
	if cfg.Telegram.Enabled {
		channel, err := channels.NewTelegramChannel(cfg.Telegram, importTopicsProvider, personDraftGenerator)
		if err != nil {
			log.Fatalf("telegram config error: %v", err)
		}
		telegramChannel = channel
		telegramBot = channel
	}

	var emailChannel channels.Channel
	if cfg.Email.Enabled {
		emailChannel = channels.NewEmailChannel(cfg.Email)
	}

	dispatcher := events.NewDispatcher(appLogger, telegramChannel, emailChannel)
	handler := httptransport.NewEventsHandler(cfg, appLogger, dispatcher)
	server := &http.Server{
		Addr:    cfg.Server.BindAddr + ":" + cfg.Server.Port,
		Handler: handler,
	}

	if telegramBot != nil {
		telegramBot.StartCommands(context.Background(), appLogger)
	}

	appLogger.Info("transport service starting", "bind_addr", cfg.Server.BindAddr, "port", cfg.Server.Port, "telegram_enabled", cfg.Telegram.Enabled, "email_enabled", cfg.Email.Enabled)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
