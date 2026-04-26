package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server   ServerConfig
	Inbound  InboundConfig
	Telegram TelegramConfig
	Email    EmailConfig
	AI       AIConfig
	API      APIConfig
}

type ServerConfig struct {
	BindAddr string
	Port     string
}

type InboundConfig struct {
	EventsSecret string
}

type TelegramConfig struct {
	BotToken string
	ChatIDs  []string
	Enabled  bool
}

type EmailConfig struct {
	Host     string
	Port     int
	PortRaw  string
	User     string
	Password string
	From     string
	To       []string
	Enabled  bool
}

type APIConfig struct {
	BaseURL   string
	HeaderKey string
	Secret    string
	Enabled   bool
}

type AIConfig struct {
	Enabled      bool
	DefaultModel string
	Models       []AIModelConfig
}

type AIModelConfig struct {
	ID            string
	Provider      string
	Name          string
	APIKey        string
	BaseURL       string
	Timeout       time.Duration
	MaxInputChars int
}

const (
	defaultAITimeout       = 30 * time.Second
	defaultAIMaxInputChars = 6000
)

func Load() (Config, error) {
	cfg := Config{
		Server: ServerConfig{
			BindAddr: envOrDefault("BIND_ADDR", "0.0.0.0"),
			Port:     envOrDefault("PORT", "8080"),
		},
		Inbound: InboundConfig{
			EventsSecret: os.Getenv("INBOUND_EVENTS_SECRET"),
		},
		Telegram: TelegramConfig{
			BotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
			ChatIDs:  parseCommaSeparatedEnv("TELEGRAM_CHAT_IDS"),
		},
		Email: EmailConfig{
			Host:     os.Getenv("SMTP_HOST"),
			PortRaw:  os.Getenv("SMTP_PORT"),
			User:     os.Getenv("SMTP_USER"),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     os.Getenv("EMAIL_FROM"),
			To:       parseCommaSeparatedEnv("EMAIL_TO"),
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

	apiCfg, err := loadAPIConfig()
	if err != nil {
		return Config{}, err
	}
	cfg.API = apiCfg

	aiCfg, err := loadAIConfig()
	if err != nil {
		return Config{}, err
	}
	cfg.AI = aiCfg

	if !cfg.Telegram.Enabled && !cfg.Email.Enabled {
		return Config{}, fmt.Errorf("at least one delivery channel must be configured")
	}

	return cfg, nil
}

func validateTelegram(cfg *TelegramConfig) error {
	hasToken := cfg.BotToken != ""
	hasChatIDs := len(cfg.ChatIDs) > 0

	if !hasToken && !hasChatIDs {
		cfg.Enabled = false
		return nil
	}

	if !hasToken || !hasChatIDs {
		return fmt.Errorf("telegram config must include both TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_IDS")
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
		"EMAIL_TO":      strings.Join(cfg.To, ","),
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

func loadAPIConfig() (APIConfig, error) {
	cfg := APIConfig{
		BaseURL:   strings.TrimSpace(os.Getenv("API_BASE_URL")),
		HeaderKey: strings.TrimSpace(os.Getenv("API_HEADER_KEY")),
		Secret:    os.Getenv("API_SECRET"),
	}

	values := []string{cfg.BaseURL, cfg.HeaderKey, cfg.Secret}
	filled := 0
	for _, value := range values {
		if value != "" {
			filled++
		}
	}

	if filled == 0 {
		return cfg, nil
	}

	if filled != len(values) {
		return APIConfig{}, fmt.Errorf("api config must include API_BASE_URL, API_HEADER_KEY and API_SECRET")
	}

	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return APIConfig{}, fmt.Errorf("API_BASE_URL must be an absolute http/https URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return APIConfig{}, fmt.Errorf("API_BASE_URL must be an absolute http/https URL")
	}

	cfg.Enabled = true
	return cfg, nil
}

func loadAIConfig() (AIConfig, error) {
	cfg := AIConfig{
		Enabled:      parseBoolEnv("AI_ENABLED"),
		DefaultModel: strings.TrimSpace(os.Getenv("AI_DEFAULT_MODEL")),
	}

	modelIDs := parseCommaSeparatedEnv("AI_MODELS")
	cfg.Models = make([]AIModelConfig, 0, len(modelIDs))
	for _, id := range modelIDs {
		envID := aiModelEnvID(id)
		cfg.Models = append(cfg.Models, AIModelConfig{
			ID:            id,
			Provider:      strings.TrimSpace(os.Getenv("AI_MODEL_" + envID + "_PROVIDER")),
			Name:          strings.TrimSpace(os.Getenv("AI_MODEL_" + envID + "_NAME")),
			APIKey:        os.Getenv("AI_MODEL_" + envID + "_API_KEY"),
			BaseURL:       strings.TrimSpace(os.Getenv("AI_MODEL_" + envID + "_BASE_URL")),
			Timeout:       parsePositiveDurationEnv("AI_MODEL_"+envID+"_TIMEOUT_SEC", defaultAITimeout),
			MaxInputChars: parsePositiveIntEnv("AI_MODEL_"+envID+"_MAX_INPUT_CHARS", defaultAIMaxInputChars),
		})
	}

	if !cfg.Enabled {
		return cfg, nil
	}

	if len(cfg.Models) == 0 {
		return AIConfig{}, fmt.Errorf("AI_MODELS is required when AI_ENABLED=true")
	}

	if cfg.DefaultModel == "" {
		return AIConfig{}, fmt.Errorf("AI_DEFAULT_MODEL is required when AI_ENABLED=true")
	}

	hasDefault := false
	seen := make(map[string]struct{}, len(cfg.Models))
	for _, model := range cfg.Models {
		if _, ok := seen[model.ID]; ok {
			return AIConfig{}, fmt.Errorf("AI_MODELS contains duplicate model %q", model.ID)
		}
		seen[model.ID] = struct{}{}

		if model.ID == cfg.DefaultModel {
			hasDefault = true
		}

		if model.Provider == "" || model.Name == "" || model.APIKey == "" || model.BaseURL == "" {
			return AIConfig{}, fmt.Errorf("AI model %q must include PROVIDER, NAME, API_KEY and BASE_URL", model.ID)
		}
	}

	if !hasDefault {
		return AIConfig{}, fmt.Errorf("AI_DEFAULT_MODEL must be included in AI_MODELS")
	}

	return cfg, nil
}

func envOrDefault(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	return value
}

func parseCommaSeparatedEnv(name string) []string {
	raw := os.Getenv(name)
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

func parseBoolEnv(name string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	return value == "true" || value == "1" || value == "yes"
}

func parsePositiveIntEnv(name string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	if err != nil || value <= 0 {
		return fallback
	}

	return value
}

func parsePositiveDurationEnv(name string, fallback time.Duration) time.Duration {
	seconds := parsePositiveIntEnv(name, int(fallback/time.Second))
	return time.Duration(seconds) * time.Second
}

func aiModelEnvID(id string) string {
	id = strings.TrimSpace(id)
	replacer := strings.NewReplacer("-", "_", ".", "_")
	return strings.ToUpper(replacer.Replace(id))
}
