package ai

import (
	"context"
	"errors"
	"time"
)

var (
	ErrDisabled      = errors.New("ai is disabled")
	ErrNotConfigured = errors.New("ai is not configured")
	ErrUnknownModel  = errors.New("ai model is not configured")
	ErrInputTooLong  = errors.New("ai input is too long")
	ErrEmptyInput    = errors.New("ai input is empty")
	ErrUnknownTask   = errors.New("ai task is not supported")
)

type Task string

const TaskGeneralDraft Task = "general_draft"

type Message struct {
	Role    string
	Content string
}

type Prompt struct {
	Messages []Message
}

type Request struct {
	Task    Task
	Input   string
	ModelID string
}

type Response struct {
	Text     string
	ModelID  string
	Provider string
}

type ModelConfig struct {
	ID            string
	Provider      string
	Name          string
	APIKey        string
	BaseURL       string
	Timeout       time.Duration
	MaxInputChars int
}

type Prompter interface {
	Build(task Task, input string) (Prompt, error)
}

type Transport interface {
	Complete(ctx context.Context, model ModelConfig, prompt Prompt) (Response, error)
}
