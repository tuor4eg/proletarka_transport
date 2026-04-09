package main

import (
	"log"
	"net/http"

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
	var telegramChannel channels.Channel
	if cfg.Telegram.Enabled {
		channel, err := channels.NewTelegramChannel(cfg.Telegram)
		if err != nil {
			log.Fatalf("telegram config error: %v", err)
		}
		telegramChannel = channel
	}

	var emailChannel channels.Channel
	if cfg.Email.Enabled {
		emailChannel = channels.NewEmailChannel(cfg.Email)
	}

	dispatcher := events.NewDispatcher(appLogger, telegramChannel, emailChannel)
	handler := httptransport.NewEventsHandler(cfg, appLogger, dispatcher)
	server := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: handler,
	}

	appLogger.Info("transport service starting", "port", cfg.Server.Port, "telegram_enabled", cfg.Telegram.Enabled, "email_enabled", cfg.Email.Enabled)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
