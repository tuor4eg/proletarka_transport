package events

import (
	"strings"
	"testing"

	"proletarka_transport/internal/domain"
)

func TestParseCommentCreatedPayloadWithNestedFields(t *testing.T) {
	payload, err := parseCommentCreatedPayload(map[string]any{
		"comment": map[string]any{
			"text":       "Looks good",
			"authorName": "Alice",
		},
		"target": map[string]any{
			"type":  "post",
			"title": "Release Notes",
		},
		"urls": map[string]any{
			"public": "https://example.com/public",
			"admin":  "https://example.com/admin",
		},
	})
	if err != nil {
		t.Fatalf("parseCommentCreatedPayload() returned error: %v", err)
	}

	if payload.CommentText != "Looks good" {
		t.Fatalf("expected comment text, got %q", payload.CommentText)
	}

	if payload.CommentAuthorName != "Alice" {
		t.Fatalf("expected author name, got %q", payload.CommentAuthorName)
	}

	if payload.TargetTitle != "Release Notes" {
		t.Fatalf("expected target title, got %q", payload.TargetTitle)
	}

	if payload.PublicURL != "https://example.com/public" {
		t.Fatalf("expected public url, got %q", payload.PublicURL)
	}
}

func TestParseCommentCreatedPayloadRequiresCommentText(t *testing.T) {
	_, err := parseCommentCreatedPayload(map[string]any{
		"comment": map[string]any{},
	})
	if err == nil {
		t.Fatal("expected error when comment text is missing")
	}

	if !strings.Contains(err.Error(), "comment text") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildCommentCreatedMessage(t *testing.T) {
	event := domain.Event{
		Event:    "comment.created",
		Severity: "normal",
		Resource: domain.EventResource{
			Kind: "comment",
			ID:   "comment_123",
		},
	}

	message := buildCommentCreatedMessage(event, commentCreatedPayload{
		CommentText:       "Looks good",
		CommentAuthorName: "Alice",
		TargetType:        "post",
		TargetTitle:       "Release Notes",
		PublicURL:         "https://example.com/public",
		AdminURL:          "https://example.com/admin",
	})

	if message.Subject != "Новый комментарий: Release Notes" {
		t.Fatalf("unexpected subject: %q", message.Subject)
	}

	for _, expected := range []string{
		"Новый комментарий",
		"Автор: Alice",
		"Тип: объект",
		"Материал: Release Notes",
		"Текст комментария:",
		"Looks good",
		"Публичная ссылка: https://example.com/public",
		"Админка: https://example.com/admin",
	} {
		if !strings.Contains(message.Text, expected) {
			t.Fatalf("expected message text to contain %q, got %q", expected, message.Text)
		}
	}
}
