package config

import (
	"os"
	"strings"
	"testing"
	"time"
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

func TestLoadAllowsEmptyAPIConfig(t *testing.T) {
	t.Setenv("INBOUND_EVENTS_SECRET", "secret")
	t.Setenv("TELEGRAM_BOT_TOKEN", "token")
	t.Setenv("TELEGRAM_CHAT_IDS", "123")
	t.Setenv("API_BASE_URL", "")
	t.Setenv("API_HEADER_KEY", "")
	t.Setenv("API_SECRET", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.API.Enabled {
		t.Fatal("expected API to be disabled")
	}
}

func TestLoadParsesFullAPIConfig(t *testing.T) {
	t.Setenv("INBOUND_EVENTS_SECRET", "secret")
	t.Setenv("TELEGRAM_BOT_TOKEN", "token")
	t.Setenv("TELEGRAM_CHAT_IDS", "123")
	t.Setenv("API_BASE_URL", "https://backend.example.com/api")
	t.Setenv("API_HEADER_KEY", "X-Backend-Secret")
	t.Setenv("API_SECRET", "backend-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if !cfg.API.Enabled {
		t.Fatal("expected API to be enabled")
	}
	if cfg.API.BaseURL != "https://backend.example.com/api" {
		t.Fatalf("unexpected API base url: %s", cfg.API.BaseURL)
	}
	if cfg.API.HeaderKey != "X-Backend-Secret" {
		t.Fatalf("unexpected API header key: %s", cfg.API.HeaderKey)
	}
	if cfg.API.Secret != "backend-secret" {
		t.Fatalf("unexpected API secret")
	}
}

func TestLoadFailsForPartialAPIConfig(t *testing.T) {
	t.Setenv("INBOUND_EVENTS_SECRET", "secret")
	t.Setenv("TELEGRAM_BOT_TOKEN", "token")
	t.Setenv("TELEGRAM_CHAT_IDS", "123")
	t.Setenv("API_BASE_URL", "https://backend.example.com")
	t.Setenv("API_HEADER_KEY", "")
	t.Setenv("API_SECRET", "backend-secret")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for partial API config")
	}
	if !strings.Contains(err.Error(), "api config must include") {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), "backend-secret") {
		t.Fatalf("error must not contain API secret: %v", err)
	}
}

func TestLoadFailsForInvalidAPIBaseURL(t *testing.T) {
	t.Setenv("INBOUND_EVENTS_SECRET", "secret")
	t.Setenv("TELEGRAM_BOT_TOKEN", "token")
	t.Setenv("TELEGRAM_CHAT_IDS", "123")
	t.Setenv("API_BASE_URL", "ftp://backend.example.com")
	t.Setenv("API_HEADER_KEY", "X-Backend-Secret")
	t.Setenv("API_SECRET", "backend-secret")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid API base URL")
	}
	if !strings.Contains(err.Error(), "API_BASE_URL must be an absolute http/https URL") {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), "backend-secret") {
		t.Fatalf("error must not contain API secret: %v", err)
	}
}

func TestLoadAllowsInvalidAIConfigWhenDisabled(t *testing.T) {
	t.Setenv("INBOUND_EVENTS_SECRET", "secret")
	t.Setenv("TELEGRAM_BOT_TOKEN", "token")
	t.Setenv("TELEGRAM_CHAT_IDS", "123")
	t.Setenv("AI_ENABLED", "false")
	t.Setenv("AI_DEFAULT_MODEL", "missing")
	t.Setenv("AI_MODELS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.AI.Enabled {
		t.Fatal("expected AI to be disabled")
	}
}

func TestLoadParsesEnabledAIConfig(t *testing.T) {
	t.Setenv("INBOUND_EVENTS_SECRET", "secret")
	t.Setenv("TELEGRAM_BOT_TOKEN", "token")
	t.Setenv("TELEGRAM_CHAT_IDS", "123")
	t.Setenv("AI_ENABLED", "true")
	t.Setenv("AI_DEFAULT_MODEL", "main")
	t.Setenv("AI_MODELS", "main,cheap")
	t.Setenv("AI_MODEL_MAIN_PROVIDER", "minimax")
	t.Setenv("AI_MODEL_MAIN_NAME", "MiniMax-M2.7")
	t.Setenv("AI_MODEL_MAIN_API_KEY", "main-key")
	t.Setenv("AI_MODEL_MAIN_BASE_URL", "https://api.minimax.io/v1")
	t.Setenv("AI_MODEL_MAIN_TIMEOUT_SEC", "15")
	t.Setenv("AI_MODEL_MAIN_MAX_INPUT_CHARS", "5000")
	t.Setenv("AI_MODEL_CHEAP_PROVIDER", "openai")
	t.Setenv("AI_MODEL_CHEAP_NAME", "gpt-4.1-mini")
	t.Setenv("AI_MODEL_CHEAP_API_KEY", "cheap-key")
	t.Setenv("AI_MODEL_CHEAP_BASE_URL", "https://api.openai.com/v1")
	t.Setenv("AI_MODEL_CHEAP_TIMEOUT_SEC", "0")
	t.Setenv("AI_MODEL_CHEAP_MAX_INPUT_CHARS", "-1")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if !cfg.AI.Enabled {
		t.Fatal("expected AI to be enabled")
	}
	if cfg.AI.DefaultModel != "main" {
		t.Fatalf("expected default AI model main, got %s", cfg.AI.DefaultModel)
	}
	if len(cfg.AI.Models) != 2 {
		t.Fatalf("expected 2 AI models, got %d", len(cfg.AI.Models))
	}
	if cfg.AI.Models[0].Timeout != 15*time.Second {
		t.Fatalf("expected custom timeout, got %s", cfg.AI.Models[0].Timeout)
	}
	if cfg.AI.Models[1].Timeout != 30*time.Second {
		t.Fatalf("expected default timeout, got %s", cfg.AI.Models[1].Timeout)
	}
	if cfg.AI.Models[1].MaxInputChars != 6000 {
		t.Fatalf("expected default max input chars, got %d", cfg.AI.Models[1].MaxInputChars)
	}
}

func TestLoadFailsWhenEnabledAIHasNoModels(t *testing.T) {
	t.Setenv("INBOUND_EVENTS_SECRET", "secret")
	t.Setenv("TELEGRAM_BOT_TOKEN", "token")
	t.Setenv("TELEGRAM_CHAT_IDS", "123")
	t.Setenv("AI_ENABLED", "true")
	t.Setenv("AI_DEFAULT_MODEL", "main")
	t.Setenv("AI_MODELS", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing AI_MODELS")
	}
	if !strings.Contains(err.Error(), "AI_MODELS is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadFailsWhenEnabledAIDefaultModelIsNotConfigured(t *testing.T) {
	t.Setenv("INBOUND_EVENTS_SECRET", "secret")
	t.Setenv("TELEGRAM_BOT_TOKEN", "token")
	t.Setenv("TELEGRAM_CHAT_IDS", "123")
	t.Setenv("AI_ENABLED", "true")
	t.Setenv("AI_DEFAULT_MODEL", "other")
	t.Setenv("AI_MODELS", "main")
	t.Setenv("AI_MODEL_MAIN_PROVIDER", "openai")
	t.Setenv("AI_MODEL_MAIN_NAME", "gpt-4.1-mini")
	t.Setenv("AI_MODEL_MAIN_API_KEY", "key")
	t.Setenv("AI_MODEL_MAIN_BASE_URL", "https://api.openai.com/v1")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing default model in AI_MODELS")
	}
	if !strings.Contains(err.Error(), "AI_DEFAULT_MODEL must be included") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadFailsWhenEnabledAIModelMissesRequiredFields(t *testing.T) {
	t.Setenv("INBOUND_EVENTS_SECRET", "secret")
	t.Setenv("TELEGRAM_BOT_TOKEN", "token")
	t.Setenv("TELEGRAM_CHAT_IDS", "123")
	t.Setenv("AI_ENABLED", "true")
	t.Setenv("AI_DEFAULT_MODEL", "main")
	t.Setenv("AI_MODELS", "main")
	t.Setenv("AI_MODEL_MAIN_PROVIDER", "openai")
	t.Setenv("AI_MODEL_MAIN_NAME", "")
	t.Setenv("AI_MODEL_MAIN_API_KEY", "key")
	t.Setenv("AI_MODEL_MAIN_BASE_URL", "https://api.openai.com/v1")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for incomplete AI model")
	}
	if !strings.Contains(err.Error(), "must include PROVIDER, NAME, API_KEY and BASE_URL") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadFailsWhenEnabledAIModelMissesBaseURL(t *testing.T) {
	t.Setenv("INBOUND_EVENTS_SECRET", "secret")
	t.Setenv("TELEGRAM_BOT_TOKEN", "token")
	t.Setenv("TELEGRAM_CHAT_IDS", "123")
	t.Setenv("AI_ENABLED", "true")
	t.Setenv("AI_DEFAULT_MODEL", "main")
	t.Setenv("AI_MODELS", "main")
	t.Setenv("AI_MODEL_MAIN_PROVIDER", "minimax")
	t.Setenv("AI_MODEL_MAIN_NAME", "model")
	t.Setenv("AI_MODEL_MAIN_API_KEY", "key")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing base url")
	}
	if !strings.Contains(err.Error(), "BASE_URL") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadAllowsEnabledAIModelWithCustomProviderAndBaseURL(t *testing.T) {
	t.Setenv("INBOUND_EVENTS_SECRET", "secret")
	t.Setenv("TELEGRAM_BOT_TOKEN", "token")
	t.Setenv("TELEGRAM_CHAT_IDS", "123")
	t.Setenv("AI_ENABLED", "true")
	t.Setenv("AI_DEFAULT_MODEL", "main")
	t.Setenv("AI_MODELS", "main")
	t.Setenv("AI_MODEL_MAIN_PROVIDER", "custom")
	t.Setenv("AI_MODEL_MAIN_NAME", "model")
	t.Setenv("AI_MODEL_MAIN_API_KEY", "key")
	t.Setenv("AI_MODEL_MAIN_BASE_URL", "https://ai.example.com/v1")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.AI.Models[0].BaseURL != "https://ai.example.com/v1" {
		t.Fatalf("unexpected base url: %s", cfg.AI.Models[0].BaseURL)
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
		"API_BASE_URL",
		"API_HEADER_KEY",
		"API_SECRET",
		"AI_ENABLED",
		"AI_DEFAULT_MODEL",
		"AI_MODELS",
		"AI_MODEL_MAIN_PROVIDER",
		"AI_MODEL_MAIN_NAME",
		"AI_MODEL_MAIN_API_KEY",
		"AI_MODEL_MAIN_BASE_URL",
		"AI_MODEL_MAIN_TIMEOUT_SEC",
		"AI_MODEL_MAIN_MAX_INPUT_CHARS",
		"AI_MODEL_CHEAP_PROVIDER",
		"AI_MODEL_CHEAP_NAME",
		"AI_MODEL_CHEAP_API_KEY",
		"AI_MODEL_CHEAP_BASE_URL",
		"AI_MODEL_CHEAP_TIMEOUT_SEC",
		"AI_MODEL_CHEAP_MAX_INPUT_CHARS",
	} {
		_ = os.Unsetenv(key)
	}
}
