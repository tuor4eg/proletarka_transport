package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Inbound  InboundConfig
	Telegram TelegramConfig
	Email    EmailConfig
}

type ServerConfig struct {
	Port string
}

type InboundConfig struct {
	EventsSecret string
}

type TelegramConfig struct {
	BotToken string
	ChatID   string
	Enabled  bool
}

type EmailConfig struct {
	Host     string
	Port     int
	PortRaw  string
	User     string
	Password string
	From     string
	To       string
	Enabled  bool
}

func Load() (Config, error) {
	cfg := Config{
		Server: ServerConfig{
			Port: envOrDefault("PORT", "8080"),
		},
		Inbound: InboundConfig{
			EventsSecret: os.Getenv("INBOUND_EVENTS_SECRET"),
		},
		Telegram: TelegramConfig{
			BotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
			ChatID:   os.Getenv("TELEGRAM_CHAT_ID"),
		},
		Email: EmailConfig{
			Host:     os.Getenv("SMTP_HOST"),
			PortRaw:  os.Getenv("SMTP_PORT"),
			User:     os.Getenv("SMTP_USER"),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     os.Getenv("EMAIL_FROM"),
			To:       os.Getenv("EMAIL_TO"),
		},
	}

	if cfg.Inbound.EventsSecret == "" {
		return Config{}, fmt.Errorf("INBOUND_EVENTS_SECRET is required")
	}

	if err := validateTelegram(&cfg.Telegram); err != nil {
		return Config{}, err
	}

	if err := validateEmail(&cfg.Email); err != nil {
		return Config{}, err
	}

	if !cfg.Telegram.Enabled && !cfg.Email.Enabled {
		return Config{}, fmt.Errorf("at least one delivery channel must be configured")
	}

	return cfg, nil
}

func validateTelegram(cfg *TelegramConfig) error {
	hasToken := cfg.BotToken != ""
	hasChatID := cfg.ChatID != ""

	if !hasToken && !hasChatID {
		cfg.Enabled = false
		return nil
	}

	if !hasToken || !hasChatID {
		return fmt.Errorf("telegram config must include both TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID")
	}

	cfg.Enabled = true
	return nil
}

func validateEmail(cfg *EmailConfig) error {
	values := map[string]string{
		"SMTP_HOST":     cfg.Host,
		"SMTP_PORT":     cfg.PortRaw,
		"SMTP_USER":     cfg.User,
		"SMTP_PASSWORD": cfg.Password,
		"EMAIL_FROM":    cfg.From,
		"EMAIL_TO":      cfg.To,
	}

	filled := 0
	for _, value := range values {
		if value != "" {
			filled++
		}
	}

	if filled == 0 {
		cfg.Enabled = false
		return nil
	}

	if filled != len(values) {
		return fmt.Errorf("email config must include SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASSWORD, EMAIL_FROM and EMAIL_TO")
	}

	port, err := strconv.Atoi(cfg.PortRaw)
	if err != nil || port <= 0 {
		return fmt.Errorf("SMTP_PORT must be a positive integer")
	}

	cfg.Port = port
	cfg.Enabled = true
	return nil
}

func envOrDefault(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	return value
}
