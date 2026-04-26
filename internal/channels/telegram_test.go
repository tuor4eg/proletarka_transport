package channels

import (
	"testing"

	"github.com/go-telegram/bot/models"
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

func messageUpdate(text string) *models.Update {
	return &models.Update{
		Message: &models.Message{
			Text: text,
		},
	}
}
