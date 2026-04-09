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
	t.Setenv("TELEGRAM_CHAT_ID", "123")
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

	if cfg.Server.Port != "9090" {
		t.Fatalf("expected port 9090, got %s", cfg.Server.Port)
	}

	if !cfg.Telegram.Enabled {
		t.Fatalf("expected telegram to be enabled")
	}

	if cfg.Email.Enabled {
		t.Fatalf("expected email to be disabled")
	}
}

func TestLoadFailsWhenAllChannelsDisabled(t *testing.T) {
	t.Setenv("INBOUND_EVENTS_SECRET", "secret")
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_CHAT_ID", "")
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
	t.Setenv("TELEGRAM_CHAT_ID", "")
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

func TestMain(m *testing.M) {
	clearEnv()
	os.Exit(m.Run())
}

func clearEnv() {
	for _, key := range []string{
		"PORT",
		"INBOUND_EVENTS_SECRET",
		"TELEGRAM_BOT_TOKEN",
		"TELEGRAM_CHAT_ID",
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
