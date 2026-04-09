package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadEnablesTelegramOnly(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("INBOUND_EVENTS_SECRET", "secret")
	t.Setenv("TELEGRAM_BOT_TOKEN", "token")
	t.Setenv("TELEGRAM_CHAT_IDS", "123,456")
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_USER", "")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("EMAIL_FROM", "")
	t.Setenv("EMAIL_TO", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Server.BindAddr != "0.0.0.0" {
		t.Fatalf("expected bind addr 0.0.0.0, got %s", cfg.Server.BindAddr)
	}

	if cfg.Server.Port != "9090" {
		t.Fatalf("expected port 9090, got %s", cfg.Server.Port)
	}

	if !cfg.Telegram.Enabled {
		t.Fatalf("expected telegram to be enabled")
	}

	if len(cfg.Telegram.ChatIDs) != 2 {
		t.Fatalf("expected 2 telegram chat ids, got %d", len(cfg.Telegram.ChatIDs))
	}

	if cfg.Email.Enabled {
		t.Fatalf("expected email to be disabled")
	}
}

func TestLoadFailsWhenAllChannelsDisabled(t *testing.T) {
	t.Setenv("INBOUND_EVENTS_SECRET", "secret")
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_CHAT_IDS", "")
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_USER", "")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("EMAIL_FROM", "")
	t.Setenv("EMAIL_TO", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when all channels are disabled")
	}

	if !strings.Contains(err.Error(), "at least one delivery channel") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadFailsForPartialEmailConfig(t *testing.T) {
	t.Setenv("INBOUND_EVENTS_SECRET", "secret")
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_CHAT_IDS", "")
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_USER", "user")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("EMAIL_FROM", "from@example.com")
	t.Setenv("EMAIL_TO", "to@example.com")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for partial email config")
	}

	if !strings.Contains(err.Error(), "email config must include") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadParsesMultipleEmailRecipients(t *testing.T) {
	t.Setenv("INBOUND_EVENTS_SECRET", "secret")
	t.Setenv("TELEGRAM_BOT_TOKEN", "token")
	t.Setenv("TELEGRAM_CHAT_IDS", "123")
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_USER", "user")
	t.Setenv("SMTP_PASSWORD", "password")
	t.Setenv("EMAIL_FROM", "from@example.com")
	t.Setenv("EMAIL_TO", "one@example.com, two@example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if len(cfg.Email.To) != 2 {
		t.Fatalf("expected 2 email recipients, got %d", len(cfg.Email.To))
	}
}

func TestMain(m *testing.M) {
	clearEnv()
	os.Exit(m.Run())
}

func clearEnv() {
	for _, key := range []string{
		"PORT",
		"BIND_ADDR",
		"INBOUND_EVENTS_SECRET",
		"TELEGRAM_BOT_TOKEN",
		"TELEGRAM_CHAT_IDS",
		"SMTP_HOST",
		"SMTP_PORT",
		"SMTP_USER",
		"SMTP_PASSWORD",
		"EMAIL_FROM",
		"EMAIL_TO",
	} {
		_ = os.Unsetenv(key)
	}
}
