package channels

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/go-telegram/bot/models"

	"proletarka_transport/internal/backend"
)

func TestIsPlainText(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "plain text", text: "что-то непонятное", want: true},
		{name: "trimmed plain text", text: "  что-то непонятное  ", want: true},
		{name: "command", text: "/unknown", want: false},
		{name: "trimmed command", text: "  /unknown  ", want: false},
		{name: "empty", text: "   ", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPlainText(tt.text); got != tt.want {
				t.Fatalf("isPlainText(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestIsCommandMessage(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "command", text: "/unknown", want: true},
		{name: "trimmed command", text: "  /unknown  ", want: true},
		{name: "plain text", text: "что-то непонятное", want: false},
		{name: "empty", text: "   ", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := messageUpdate(tt.text)
			if got := isCommandMessage(update); got != tt.want {
				t.Fatalf("isCommandMessage(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestAddPersonHandlerReturnsConfiguredAPIMessageWhenProviderNil(t *testing.T) {
	got, err := addPersonHandler(nil)(context.Background())
	if err != nil {
		t.Fatalf("addPersonHandler() returned error: %v", err)
	}
	if !strings.Contains(got, "API основного backend не настроен") {
		t.Fatalf("message = %q, want API disabled explanation", got)
	}
}

func TestAddPersonHandlerFormatsProviderTopics(t *testing.T) {
	handler := addPersonHandler(fakeImportTopicsProvider{
		topics: []backend.ImportTopic{
			{Code: "war", Title: "Война"},
		},
	})

	got, err := handler(context.Background())
	if err != nil {
		t.Fatalf("addPersonHandler() returned error: %v", err)
	}
	if !strings.Contains(got, "- Война (war)") {
		t.Fatalf("message = %q, want formatted topic", got)
	}
}

func TestAddPersonHandlerHidesProviderError(t *testing.T) {
	handler := addPersonHandler(fakeImportTopicsProvider{
		err: fmt.Errorf("backend secret raw error"),
	})

	got, err := handler(context.Background())
	if err != nil {
		t.Fatalf("addPersonHandler() returned error: %v", err)
	}
	if got != importTopicsUnavailableMessage {
		t.Fatalf("message = %q, want %q", got, importTopicsUnavailableMessage)
	}
	if strings.Contains(got, "backend secret raw error") {
		t.Fatalf("message exposes raw error: %q", got)
	}
}

func messageUpdate(text string) *models.Update {
	return &models.Update{
		Message: &models.Message{
			Text: text,
		},
	}
}

type fakeImportTopicsProvider struct {
	topics []backend.ImportTopic
	err    error
}

func (p fakeImportTopicsProvider) FetchImportTopics(ctx context.Context) ([]backend.ImportTopic, error) {
	return p.topics, p.err
}
